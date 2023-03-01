package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	tokens []Token
	index  int
}

func NewParser(tokens []Token) parser {
	return parser{
		tokens: tokens,
		index:  0,
	}
}

type parseError struct {
	reason string
	Pos
}

func (e parseError) Error() string {
	return fmt.Sprintf("Parse error at %s: %s", e.Pos.String(), e.reason)
}

func (p *parser) isEOF() bool {
	return p.index == len(p.tokens)
}

func (p *parser) peek() Token {
	return p.tokens[p.index]
}

func (p *parser) peekAhead(n int) Token {
	if p.index+n > len(p.tokens) {
		return Token{Kind: IndentEndStatment}
	}
	return p.tokens[p.index+n]
}

func (p *parser) next() Token {
	tok := p.tokens[p.index]
	if p.index < len(p.tokens) {
		p.index++
	}
	return tok
}

func (p *parser) back() {
	if p.index > 0 {
		p.index--
	}
}

// When we find '(', it might be the start of a function:
// (a) => {}
// or it might be
// (1 + 2)
// just grouping an expression.
//
// This function assumes that we are at '(', and looks ahead to see if we
// are in a function or just '(1 + 2)'
func (p *parser) isStartOfFunction() bool {
	i := 0
	for {
		if p.index+i > len(p.tokens) {
			return false
		}

		tok := p.tokens[p.index+i]
		switch tok.Kind {
		case RightParen:
			next := p.tokens[p.index+i+1]
			return next.Kind == FnArrow
		case Identifier, Comma, Colon:
			i++
			continue
		default:
			return false
		}
	}
}

func (p *parser) readUntilTokenKind(kind TokKind) []Token {
	tokens := []Token{}
	for !p.isEOF() && p.peek().Kind != kind {
		t := p.next()
		tokens = append(tokens, t)
	}
	return tokens
}

func (p *parser) expect(kind TokKind) (Token, error) {
	tok := Token{Kind: kind}
	if p.isEOF() {
		return Token{Kind: Unknown}, parseError{
			reason: fmt.Sprintf("Unexpected end of input, expected %s", tok),
			Pos:    tok.Pos,
		}
	}
	next := p.next()
	if next.Kind != kind {
		return Token{Kind: Unknown}, parseError{
			reason: fmt.Sprintf("Unexpected token %s, expected %s", next, tok),
			Pos:    next.Pos,
		}
	}
	return next, nil
}

func (p *parser) parseAssignment(left AstNode) (AstNode, error) {
	next := p.next()
	if next.Kind != Assign {
		return nil, parseError{
			reason: fmt.Sprintf("Expected assign token, got: %s", next),
			Pos:    next.Pos,
		}
	}
	node := AssignmentNode{
		Left: left,
		Tok:  &next,
	}

	right, err := p.parseNode()
	if err != nil {
		return nil, err
	}
	node.Right = right
	return node, nil
}

// Does NOT recursively parse the right tree.
func (p *parser) parseBinaryOP(left AstNode) (AstNode, error) {
	// NOTE: maybe add a double check for op actually being a BinaryToken
	op := p.next()
	node := BinaryNode{
		Left: left,
		Op:   op.Kind,
		Tok:  &op,
	}
	right, err := p.parseUnit()
	if err != nil {
		return nil, err
	}

	// dot has the 'ultimate' precedence...
	if !p.isEOF() && p.peek().Kind == Dot {
		right, err = p.parseBinaryDot(right)
		if err != nil {
			return nil, err
		}
	}

	node.Right = right
	return node, nil
}

// Syntactic sugar for piping functions together
//
// res = one.add(1)
// turns into
// res = add(one, 1)
func (p *parser) parseBinaryDot(left AstNode) (AstNode, error) {
	next := p.next() // eat the dot

	subNode, err := p.parseSubNode()
	if err != nil {
		return nil, err
	}

	// Setting left as first argument in fnCallNode
	switch callNode := subNode.(type) {
	case FnCallNode:
		args := []AstNode{left}
		if len(callNode.Args) > 0 {
			args = append(args, callNode.Args...)
		}
		callNode.Args = args
		return callNode, nil

	default:
		return nil, parseError{
			reason: fmt.Sprintf("Expected a callNode, got: %s", callNode),
			Pos:    next.Pos,
		}
	}
}

func (p *parser) parseNumberLiteral(tok Token) (AstNode, error) {
	if strings.ContainsRune(tok.Payload, '.') {
		f, err := strconv.ParseFloat(tok.Payload, 64)
		if err != nil {
			return nil, parseError{reason: err.Error(), Pos: tok.Pos}
		}
		return FloatNode{
			Payload: f,
			Tok:     &tok,
		}, nil
	}
	n, err := strconv.ParseInt(tok.Payload, 10, 64)
	if err != nil {
		return nil, parseError{reason: err.Error(), Pos: tok.Pos}
	}
	return IntNode{
		Payload: n,
		Tok:     &tok,
	}, nil
}

func (p *parser) parseEnum(tok Token) (AstNode, error) {
	parentName := tok.Payload
	p.next() // eat double colon
	name, err := p.expect(Identifier)
	if err != nil {
		return nil, err
	}
	args := []AstNode{}
	for !p.isEOF() && p.peek().Kind == LeftParen {
		_ = p.next() // eat or
		b, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		args = append(args, b)
	}

	if len(args) > 0 {
		_, err = p.expect(RightParen)
		if err != nil {
			return nil, err
		}
	}
	return EnumNode{
		Parent: parentName,
		Name:   name.Payload,
		Args:   args,
		Tok:    &tok,
	}, nil
}

func SplitTokensBy(tokens []Token, kind TokKind) [][]Token {
	newtokens := make([][]Token, 0)
	i := 0
	for _, t := range tokens {
		if t.Kind == kind {
			i++
			continue
		}
		if len(newtokens) == i {
			newtokens = append(newtokens, []Token{t})
		} else {
			newtokens[i] = append(newtokens[i], t)
		}
	}
	return newtokens
}

func (p *parser) parseFunction(tok Token) (AstNode, error) {
	tokens := p.readUntilTokenKind(RightParen)
	p.next() // eat right paren
	if p.peek().Kind != FnArrow {
		return nil, parseError{
			reason: fmt.Sprintf("Expected =>, got %s", p.peek()),
			Pos:    tok.Pos,
		}
	}
	p.next() // eat arrow

	args := []Arg{}
	groupedTokens := SplitTokensBy(tokens, Comma)
	for _, paramToken := range groupedTokens {
		if len(paramToken) == 2 { // makes no sense..
			return nil, parseError{
				reason: "Check your parameters..",
				Pos:    tok.Pos,
			}
		}
		if len(paramToken) == 1 { // No type given
			args = append(args, Arg{Name: paramToken[0].Payload})
			continue
		}
		// type/alias given
		args = append(args, Arg{
			Name:  paramToken[0].Payload,
			Alias: paramToken[2].Payload,
		})
	}

	body, err := p.parseNode()
	if err != nil {
		return nil, err
	}

	return FnNode{
		Args: args,
		Body: body,
		Tok:  &tok,
	}, nil
}

func (p *parser) parseUnit() (AstNode, error) {
	tok := p.next()
	switch tok.Kind {
	case NumberLiteral:
		return p.parseNumberLiteral(tok)
	case TrueLiteral:
		return BoolNode{Payload: true, Tok: &tok}, nil
	case StringLiteral:
		payloadBuilder := bytes.Buffer{}
		runes := []rune(tok.Payload)
		for i := 0; i < len(runes); i++ {
			c := runes[i]

			if c == '\\' {
				if i+1 >= len(runes) {
					break
				}
				i++
				c = runes[i]

				switch c {
				case 't':
					_ = payloadBuilder.WriteByte('\t')
				case 'n':
					_ = payloadBuilder.WriteByte('\n')
				case 'r':
					_ = payloadBuilder.WriteByte('\r')
				case 'f':
					_ = payloadBuilder.WriteByte('\f')
				case 'x':
					if i+2 >= len(runes) {
						_ = payloadBuilder.WriteByte('x')
						continue
					}

					hexCode, err := strconv.ParseUint(string(runes[i+1])+string(runes[i+2]), 16, 8)
					if err == nil {
						i += 2
						_ = payloadBuilder.WriteByte(uint8(hexCode))
					} else {
						_ = payloadBuilder.WriteByte('x')
					}
				default:
					_, _ = payloadBuilder.WriteRune(c)
				}
			} else {
				_, _ = payloadBuilder.WriteRune(c)
			}
		}
		return StringNode{Payload: payloadBuilder.Bytes(), Tok: &tok}, nil
		// return stringNode{payload: tok.payload, tok: &tok}, nil
	case FalseLiteral:
		return BoolNode{Payload: false, Tok: &tok}, nil
	case Underscore:
		return UnderscoreNode{tok: &tok}, nil
	case Identifier:
		for !p.isEOF() && p.peek().Kind == DoubleColon {
			return p.parseEnum(tok)
		}
		return IdentifierNode{Payload: tok.Payload, Tok: &tok}, nil
	case LeftParen:
		if p.isStartOfFunction() {
			return p.parseFunction(tok)
		}
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(RightParen)
		if err != nil {
			return nil, err
		}
		return node, nil

	case AliasKeyword:
		name, err := p.expect(Identifier)
		if err != nil {
			return nil, err
		}
		_, err = p.expect(Assign)
		if err != nil {
			return nil, err
		}
		body, err := p.parseUnit()
		if err != nil {
			return nil, err
		}
		// ...
		targets := []AstNode{body}
		for !p.isEOF() && p.peek().Kind == Or {
			_ = p.next() // eat or
			b, err := p.parseUnit()
			if err != nil {
				return nil, err
			}
			targets = append(targets, b)
		}
		return AliasNode{
			Name:    name.Payload,
			Targets: targets,
			Tok:     &tok,
		}, nil

	case MatchKeyword:
		var cond AstNode
		branches := []MatchBranch{}
		// if no explicit condition is provided (i.e. if the keyword is
		// followed by a { ... }), we assume the condition is "true" to allow
		// for the useful `if { case, case ... }` pattern.
		var err error
		if p.peek().Kind == LeftBrace {
			cond = BoolNode{
				Payload: true,
				Tok:     &tok,
			}
		} else {
			cond, err = p.parseNode()
			if err != nil {
				return nil, err
			}
		}

		_, err = p.expect(LeftBrace)
		if err != nil {
			return nil, err
		}

		for !p.isEOF() && p.peek().Kind != RightBrace {
			targets := []AstNode{}
			for !p.isEOF() && p.peek().Kind != BranchArrow {
				// You can separatte multiple targets "within" a branch.
				// It just really desugars to multiple targets with the same body.
				target, err := p.parseNode()
				if err != nil {
					return nil, err
				}
				targets = append(targets, target)
				if p.peek().Kind == Comma {
					p.next()
				} else {
					break
				}
			}

			if _, err := p.expect(BranchArrow); err != nil {
				return nil, err
			}

			body, err := p.parseNode()
			if err != nil {
				return nil, err
			}

			for _, target := range targets {
				branches = append(branches, MatchBranch{
					Target: target,
					Body:   body,
				})
			}
		}

		if _, err := p.expect(RightBrace); err != nil {
			return nil, err
		}

		return MatchNode{
			Cond:     cond,
			Branches: branches,
			Tok:      &tok,
		}, nil

	case LeftBrace:
		firstExpr, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		nodes := []AstNode{firstExpr}
		for !p.isEOF() && p.peek().Kind != RightBrace {
			node, err := p.parseNode()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
		p.next() // eat rightBrace
		return BlockNode{
			Exprs: nodes,
			Tok:   &tok,
		}, nil

	case LeftBracket:
		nodes := []AstNode{}
		for !p.isEOF() && p.peek().Kind != RightBracket {
			node, err := p.parseNode()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
			if p.peek().Kind == Comma {
				p.next()
			} else {
				break
			}
		}
		p.next() // eat rightBracket
		return ListNode{
			Elems: nodes,
			Tok:   &tok,
		}, nil
	}
	return nil, parseError{
		reason: fmt.Sprintf("Unexpected token %s at start of unit", tok),
		Pos:    tok.Pos,
	}
}

// Used for:
// - unary and binary expressions.
// - function calls
// Sits between parseUnit and parseNode.
func (p *parser) parseSubNode() (AstNode, error) {
	node, err := p.parseUnit()
	if err != nil {
		return nil, err
	}

	for !p.isEOF() {
		switch p.peek().Kind {
		case LeftParen: // Function call
			next := p.next() // eat the leftParen
			args := []AstNode{}
			for !p.isEOF() && p.peek().Kind != RightParen {
				arg, err := p.parseNode()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.peek().Kind == Comma {
					p.next()
				} else {
					break
				}
			}
			if _, err := p.expect(RightParen); err != nil {
				return nil, err
			}
			// Setting the "node" from parseUnit as the function caller
			// and we are only parsing the arguments here
			node = FnCallNode{
				Fn:   node,
				Args: args,
				Tok:  &next,
			}
		default:
			return node, nil
		}
	}
	return node, nil
}

// parseNode returns the next top-level astNode from the parser
func (p *parser) parseNode() (AstNode, error) {
	node, err := p.parseSubNode()
	if err != nil {
		return nil, err
	}

	for !p.isEOF() {
		switch p.peek().Kind {
		case Assign:
			return p.parseAssignment(node)
		case Plus, Minus, Times, Divide, PlusOther, Eq, Neq, Greater, Less, Modulus, Geq, Leq:
			// TODO: add: and, or:
			//
			// We keep looping here because we want to adhere to order of operations.
			// Which means that there might be more binary operations coming, and we need to catch them here.
			node, err = p.parseBinaryOP(node)
			if err != nil {
				return nil, err
			}
		case Dot:
			// We keep looping here because we want to adhere to order of operations.
			// Which means that there might be more binary operations coming, and we need to catch them here.
			node, err = p.parseBinaryDot(node)
			if err != nil {
				return nil, err
			}
		default:
			return node, nil
		}
	}
	return node, nil
}

func (p *parser) Parse() ([]AstNode, error) {
	nodes := []AstNode{}
	for !p.isEOF() {
		node, err := p.parseNode()
		if err != nil {
			return nodes, err
		}
		// _ = p.next()
		// if _, err = p.expect(comma); err != nil {
		// 	return nodes, err
		// }
		nodes = append(nodes, node)
	}

	return nodes, nil
}
