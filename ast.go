package main

import (
	"strconv"
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
