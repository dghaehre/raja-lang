package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	tokens []token
	index  int
}

func newParser(tokens []token) parser {
	return parser{
		tokens: tokens,
		index:  0,
	}
}

type parseError struct {
	reason string
	pos
}

func (e parseError) Error() string {
	return fmt.Sprintf("Parse error at %s: %s", e.pos.String(), e.reason)
}
func (p *parser) isEOF() bool {
	return p.index == len(p.tokens)
}

func (p *parser) peek() token {
	return p.tokens[p.index]
}

func (p *parser) peekAhead(n int) token {
	if p.index+n > len(p.tokens) {
		return token{kind: indentEndStatment}
	}
	return p.tokens[p.index+n]
}

func (p *parser) next() token {
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
		switch tok.kind {
		case rightParen:
			next := p.tokens[p.index+i+1]
			return next.kind == fnArrow
		case identifier, comma, colon:
			i++
			continue
		default:
			return false
		}
	}
}

func (p *parser) readUntilTokenKind(kind tokKind) []token {
	tokens := []token{}
	for !p.isEOF() && p.peek().kind != kind {
		t := p.next()
		tokens = append(tokens, t)
	}
	return tokens
}

func (p *parser) expect(kind tokKind) (token, error) {
	tok := token{kind: kind}
	if p.isEOF() {
		return token{kind: unknown}, parseError{
			reason: fmt.Sprintf("Unexpected end of input, expected %s", tok),
			pos:    tok.pos,
		}
	}
	next := p.next()
	if next.kind != kind {
		return token{kind: unknown}, parseError{
			reason: fmt.Sprintf("Unexpected token %s, expected %s", next, tok),
			pos:    next.pos,
		}
	}
	return next, nil
}

func (p *parser) parseAssignment(left astNode) (astNode, error) {
	next := p.next()
	if next.kind != assign {
		return nil, parseError{
			reason: fmt.Sprintf("Expected assign token, got: %s", next),
			pos:    next.pos,
		}
	}
	node := assignmentNode{
		left: left,
		tok:  &next,
	}

	right, err := p.parseNode()
	if err != nil {
		return nil, err
	}
	node.right = right
	return node, nil
}

// Does NOT recursively parse the right tree.
func (p *parser) parseBinaryOP(left astNode) (astNode, error) {
	// NOTE: maybe add a double check for op actually being a BinaryToken
	op := p.next()
	node := binaryNode{
		left: left,
		op:   op.kind,
		tok:  &op,
	}

	right, err := p.parseUnit()
	if err != nil {
		return nil, err
	}
	node.right = right
	return node, nil
}

// Syntactic sugar for piping functions together
//
// res = one.add(1)
// turns into
// res = add(one, 1)
func (p *parser) parseBinaryDot(left astNode) (astNode, error) {
	next := p.next() // eat the dot

	subNode, err := p.parseSubNode()
	if err != nil {
		return nil, err
	}

	// Setting left as first argument in fnCallNode
	switch callNode := subNode.(type) {
	case fnCallNode:
		args := []astNode{left}
		if len(callNode.args) > 0 {
			args = append(args, callNode.args...)
		}
		callNode.args = args
		return callNode, nil

	default:
		return nil, parseError{
			reason: fmt.Sprintf("Expected a callNode, got: %s", callNode),
			pos:    next.pos,
		}
	}
}

func (p *parser) parseNumberLiteral(tok token) (astNode, error) {
	if strings.ContainsRune(tok.payload, '.') {
		f, err := strconv.ParseFloat(tok.payload, 64)
		if err != nil {
			return nil, parseError{reason: err.Error(), pos: tok.pos}
		}
		return floatNode{
			payload: f,
			tok:     &tok,
		}, nil
	}
	n, err := strconv.ParseInt(tok.payload, 10, 64)
	if err != nil {
		return nil, parseError{reason: err.Error(), pos: tok.pos}
	}
	return intNode{
		payload: n,
		tok:     &tok,
	}, nil
}

func (p *parser) parseEnum(tok token) (astNode, error) {
	parentName := tok.payload
	p.next() // eat double colon
	name, err := p.expect(identifier)
	if err != nil {
		return nil, err
	}
	args := []astNode{}
	for !p.isEOF() && p.peek().kind == leftParen {
		_ = p.next() // eat or
		b, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		args = append(args, b)
	}

	if len(args) > 0 {
		_, err = p.expect(rightParen)
		if err != nil {
			 return nil, err
		}
	}
	return enumNode{
		parent: parentName,
		name:   name.payload,
		args:   args,
		tok:    &tok,
	}, nil
}

func (p *parser) parseFunction(tok token) (astNode, error) {
	tokens := p.readUntilTokenKind(rightParen)
	p.next() // eat right paren
	if p.peek().kind != fnArrow {
		return nil, parseError{
			reason: fmt.Sprintf("Expected =>, got %s", p.peek()),
			pos:    tok.pos,
		}
	}
	p.next() // eat arrow

	args := []Arg{}
	groupedTokens := SplitTokensBy(tokens, comma)
	for _, paramToken := range groupedTokens {
		if len(paramToken) == 2 { // makes no sense..
			return nil, parseError{
				reason: "Check your parameters..",
				pos:    tok.pos,
			}
		}
		if len(paramToken) == 1 { // No type given
			args = append(args, Arg{name: paramToken[0].payload})
			continue
		}
		// type/alias given
		args = append(args, Arg{
			name:  paramToken[0].payload,
			alias: paramToken[2].payload,
		})
	}

	body, err := p.parseNode()
	if err != nil {
		return nil, err
	}

	return fnNode{
		args: args,
		body: body,
		tok:  &tok,
	}, nil
}

func (p *parser) parseUnit() (astNode, error) {
	tok := p.next()
	switch tok.kind {
	case numberLiteral:
		return p.parseNumberLiteral(tok)
	case trueLiteral:
		return boolNode{payload: true, tok: &tok}, nil
	case stringLiteral:
		payloadBuilder := bytes.Buffer{}
		runes := []rune(tok.payload)
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
		return stringNode{payload: payloadBuilder.Bytes(), tok: &tok}, nil
		// return stringNode{payload: tok.payload, tok: &tok}, nil
	case falseLiteral:
		return boolNode{payload: false, tok: &tok}, nil
	case underscore:
		return underscoreNode{tok: &tok}, nil
	case identifier:
		for !p.isEOF() && p.peek().kind == doubleColon {
			return p.parseEnum(tok)
		}
		return identifierNode{payload: tok.payload, tok: &tok}, nil
	case leftParen:
		if p.isStartOfFunction() {
			return p.parseFunction(tok)
		}
		node, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		_, err = p.expect(rightParen)
		if err != nil {
			return nil, err
		}
		return node, nil

	case aliasKeyword:
		name, err := p.expect(identifier)
		if err != nil {
			return nil, err
		}
		_, err = p.expect(assign)
		if err != nil {
			return nil, err
		}
		body, err := p.parseUnit()
		if err != nil {
			return nil, err
		}
		// ...
		targets := []astNode{body}
		for !p.isEOF() && p.peek().kind == or {
			_ = p.next() // eat or
			b, err := p.parseUnit()
			if err != nil {
				return nil, err
			}
			targets = append(targets, b)
		}
		return aliasNode{
			name:    name.payload,
			targets: targets,
			tok:     &tok,
		}, nil

	case matchKeyword:
		var cond astNode
		branches := []matchBranch{}
		// if no explicit condition is provided (i.e. if the keyword is
		// followed by a { ... }), we assume the condition is "true" to allow
		// for the useful `if { case, case ... }` pattern.
		var err error
		if p.peek().kind == leftBrace {
			cond = boolNode{
				payload: true,
				tok:     &tok,
			}
		} else {
			cond, err = p.parseNode()
			if err != nil {
				return nil, err
			}
		}

		_, err = p.expect(leftBrace)
		if err != nil {
			return nil, err
		}

		for !p.isEOF() && p.peek().kind != rightBrace {
			targets := []astNode{}
			for !p.isEOF() && p.peek().kind != branchArrow {
				// You can separatte multiple targets "within" a branch.
				// It just really desugars to multiple targets with the same body.
				target, err := p.parseNode()
				if err != nil {
					return nil, err
				}
				targets = append(targets, target)
				if p.peek().kind == comma {
					p.next()
				} else {
					break
				}
			}

			if _, err := p.expect(branchArrow); err != nil {
				return nil, err
			}

			body, err := p.parseNode()
			if err != nil {
				return nil, err
			}

			for _, target := range targets {
				branches = append(branches, matchBranch{
					target: target,
					body:   body,
				})
			}
		}

		if _, err := p.expect(rightBrace); err != nil {
			return nil, err
		}

		return matchNode{
			cond:     cond,
			branches: branches,
			tok:      &tok,
		}, nil

	case leftBrace:
		firstExpr, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		nodes := []astNode{firstExpr}
		for !p.isEOF() && p.peek().kind != rightBrace {
			node, err := p.parseNode()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}
		p.next() // eat rightBrace
		return blockNode{
			exprs: nodes,
			tok:   &tok,
		}, nil

	case leftBracket:
		nodes := []astNode{}
		for !p.isEOF() && p.peek().kind != rightBracket {
			node, err := p.parseNode()
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
			if p.peek().kind == comma {
				p.next()
			} else {
				break
			}
		}
		p.next() // eat rightBracket
		return listNode{
			elems: nodes,
			tok:   &tok,
		}, nil
	}
	return nil, parseError{
		reason: fmt.Sprintf("Unexpected token %s at start of unit", tok),
		pos:    tok.pos,
	}
}

// Used for:
// - unary and binary expressions.
// - function calls
// Sits between parseUnit and parseNode.
func (p *parser) parseSubNode() (astNode, error) {
	node, err := p.parseUnit()
	if err != nil {
		return nil, err
	}

	for !p.isEOF() {
		switch p.peek().kind {
		case leftParen: // Function call
			next := p.next() // eat the leftParen
			args := []astNode{}
			for !p.isEOF() && p.peek().kind != rightParen {
				arg, err := p.parseNode()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				if p.peek().kind == comma {
					p.next()
				} else {
					break
				}
			}
			if _, err := p.expect(rightParen); err != nil {
				return nil, err
			}
			// Setting the "node" from parseUnit as the function caller
			// and we are only parsing the arguments here
			node = fnCallNode{
				fn:   node,
				args: args,
				tok:  &next,
			}
		default:
			return node, nil
		}
	}
	return node, nil
}

// parseNode returns the next top-level astNode from the parser
func (p *parser) parseNode() (astNode, error) {
	node, err := p.parseSubNode()
	if err != nil {
		return nil, err
	}

	for !p.isEOF() {
		switch p.peek().kind {
		case assign:
			return p.parseAssignment(node)
		case plus, minus, times, divide, plusOther, eq, neq:
			// TODO: add: and, or, greater, less, eq, geq, leq, neq:
			//
			// We keep looping here because we want to adhere to order of operations.
			// Which means that there might be more binary operations coming, and we need to catch them here.
			node, err = p.parseBinaryOP(node)
			if err != nil {
				return nil, err
			}
		case dot:
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

func (p *parser) parse() ([]astNode, error) {
	nodes := []astNode{}
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
