package main

import (
	"fmt"
	"strconv"
	"strings"
)

type astNode interface {
	String() string
	pos() pos
}

type intNode struct {
	payload int64
	tok     *token
}

func (n intNode) String() string {
	return strconv.FormatInt(n.payload, 10)
}
func (n intNode) pos() pos {
	return n.tok.pos
}

type floatNode struct {
	payload float64
	tok     *token
}

func (n floatNode) String() string {
	return strconv.FormatFloat(n.payload, 'g', -1, 64)
}
func (n floatNode) pos() pos {
	return n.tok.pos
}

type boolNode struct {
	payload bool
	tok     *token
}

func (n boolNode) String() string {
	if n.payload {
		return "true"
	}
	return "false"
}
func (n boolNode) pos() pos {
	return n.tok.pos
}

type identifierNode struct {
	payload string
	tok     *token
}

func (n identifierNode) String() string {
	return n.payload
}
func (n identifierNode) pos() pos {
	return n.tok.pos
}

type stringNode struct {
	payload string
	tok     *token
}

func (n stringNode) String() string {
	return n.payload
}
func (n stringNode) pos() pos {
	return n.tok.pos
}

// TODO: isLocal
type assignmentNode struct {
	left  astNode
	right astNode
	tok   *token
}

func (n assignmentNode) String() string {
	return n.left.String() + " = " + n.right.String()
}
func (n assignmentNode) pos() pos {
	return n.tok.pos
}

type binaryNode struct {
	op    tokKind
	left  astNode
	right astNode
	tok   *token
}

func (n binaryNode) String() string {
	opTok := token{kind: n.op}
	return "(" + n.left.String() + " " + opTok.String() + " " + n.right.String() + ")"
}
func (n binaryNode) pos() pos {
	return n.tok.pos
}

type blockNode struct {
	exprs []astNode
	tok   *token
}

func (n blockNode) String() string {
	exprStrings := make([]string, len(n.exprs))
	for i, ex := range n.exprs {
		exprStrings[i] = ex.String()
	}
	return "{ " + strings.Join(exprStrings, ", ") + " }"
}

func (n blockNode) pos() pos {
	return n.tok.pos
}

type fnNode struct {
	args []string
	body astNode
	tok  *token
}

func (n fnNode) String() string {
	return fmt.Sprintf("(%s) => %s", strings.Join(n.args, ", "), n.body.String())
}

func (n fnNode) pos() pos {
	return n.tok.pos
}

type fnCallNode struct {
	fn   astNode
	args []astNode
	tok  *token
}

func (n fnCallNode) String() string {
	argStrings := make([]string, len(n.args))
	for i, arg := range n.args {
		argStrings[i] = arg.String()
	}
	return fmt.Sprintf("fncall[%s](%s)", n.fn, strings.Join(argStrings, ", "))
}
func (n fnCallNode) pos() pos {
	return n.tok.pos
}

type listNode struct {
	elems []astNode
	tok   *token
}

func (n listNode) String() string {
	elemStrings := make([]string, len(n.elems))
	for i, el := range n.elems {
		elemStrings[i] = el.String()
	}
	return "[" + strings.Join(elemStrings, ", ") + "]"
}
func (n listNode) pos() pos {
	return n.tok.pos
}
