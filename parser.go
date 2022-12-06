package main

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	tokens        []token
	index         int
	minBinaryPrec []int
}

func newParser(tokens []token) parser {
	return parser{
		tokens:        tokens,
		index:         0,
		minBinaryPrec: []int{0}, // ?
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

func (p *parser) parseBinaryOP(left astNode) (astNode, error) {
	// NOTE: maybe add a double check for op actually being a BinaryToken
	op := p.next()
	node := binaryNode{
		left: left,
		op:   op.kind,
		tok:  &op,
	}

	right, err := p.parseNode()
	if err != nil {
		return nil, err
	}
	node.right = right
	return node, nil
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

func (p *parser) parseUnit() (astNode, error) {
	tok := p.next()
	switch tok.kind {
	case numberLiteral:
		return p.parseNumberLiteral(tok)
	case trueLiteral:
		return boolNode{payload: true, tok: &tok}, nil
	case stringLiteral:
		return stringNode{payload: tok.payload, tok: &tok}, nil
	case falseLiteral:
		return boolNode{payload: false, tok: &tok}, nil
	case identifier:
		return identifierNode{payload: tok.payload, tok: &tok}, nil
	case leftParen:
		// This might be the start of a function!
		// might have to backtrack incase this is not a function..
		tokens := p.readUntilTokenKind(rightParen)
		p.next() // eat right paren
		// Its a function!
		if p.peek().kind == fnArrow {
			p.next() // eat arrow
			args := []string{}
			for _, t := range tokens {
				// TODO: make sure they are all "identifiers"
				args = append(args, t.String())
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
		} else {
			// TODO: parse (..)
			return nil, parseError{
				reason: fmt.Sprintf("Unhandled.."),
				pos:    tok.pos,
			}
		}
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

	for !p.isEOF() && p.peek().kind != indentEndStatment {
		switch p.peek().kind {
		case assign:
			return p.parseAssignment(node)
		case plus, minus, times, divide, plusString:
			// TODO: add: and, or, greater, less, eq, geq, leq, neq:
			return p.parseBinaryOP(node)
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
