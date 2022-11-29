package main

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type stackEntry struct {
	name string
	pos
}

func (e stackEntry) String() string {
	if e.name != "" {
		return fmt.Sprintf("  in function %s %s", e.name, e.pos)
	}
	return fmt.Sprintf("  in anonymous function %s", e.pos)
}

type runtimeError struct {
	reason string
	pos
	stackTrace []stackEntry
}

func (e *runtimeError) Error() string {
	trace := make([]string, len(e.stackTrace))
	for i, entry := range e.stackTrace {
		trace[i] = entry.String()
	}
	return fmt.Sprintf("Runtime error %s: %s\n%s", e.pos, e.reason, strings.Join(trace, "\n"))
}

type scope struct {
	parent *scope

	// vars needs to be extended to handle multiple functions with the same name
	vars map[string]Value
}

type Context struct {
	scope
}

func NewContext() Context {
	return Context{
		scope: scope{
			parent: nil,
			vars:   map[string]Value{},
		},
	}
}

// Value

type Value interface {
	String() string

	// NOTE: we might have to do something smart about these equality checks for
	// getting multiple dispatch to work
	Eq(Value) bool
}

type IntValue int64

func (v IntValue) String() string {
	return strconv.FormatInt(int64(v), 10)
}

func (v IntValue) Eq(u Value) bool {
	if w, ok := u.(IntValue); ok {
		return v == w
	} else if w, ok := u.(FloatValue); ok {
		return FloatValue(v) == w
	}
	return false
}

type BoolValue bool

func (v BoolValue) String() string {
	if v {
		return "true"
	} else {
		return "false"
	}
}

func (v BoolValue) Eq(u Value) bool {
	if w, ok := u.(BoolValue); ok {
		return v == w
	}
	return false
}

type FloatValue float64

func (v FloatValue) String() string {
	return strconv.FormatFloat(float64(v), 'g', -1, 64)
}
func (v FloatValue) Eq(u Value) bool {
	if w, ok := u.(FloatValue); ok {
		return v == w
	} else if w, ok := u.(IntValue); ok {
		return v == FloatValue(w)
	}
	return false
}

type StringValue []byte

func (v StringValue) String() string {
	return string(v)
}

func (v StringValue) Eq(u Value) bool {
	if w, ok := u.(StringValue); ok {
		return bytes.Equal(v, w)
	}
	return false
}

// Scope

// Put variable into scope
func (sc *scope) put(name string, v Value) {
	sc.vars[name] = v
}

func (sc *scope) get(name string) (Value, *runtimeError) {
	v, ok := sc.vars[name]
	if !ok {
		return nil, &runtimeError{
			reason: fmt.Sprintf("%s is undefined", name),
		}
	}
	return v, nil
}

// Eval

func (c *Context) Eval(reader io.Reader) (Value, error) {
	program, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	tokenizer := newTokenizer(string(program))
	tokens := tokenizer.tokenize()
	// fmt.Printf("tokens: %v\n", tokens)
	parser := newParser(tokens)
	nodes, err := parser.parse()
	if err != nil {
		return nil, err
	}
	// fmt.Println("Parsed:")
	// for _, n := range nodes {
	// 	fmt.Println(n)
	// }
	v, runtimeErr := c.evalNodes(nodes)
	if runtimeErr != nil {
		return nil, runtimeErr
	}
	return v, nil
}

func incompatibleError(op tokKind, left, right Value, position pos) *runtimeError {
	return &runtimeError{
		reason: fmt.Sprintf("Cannot %s incompatible values %s, %s",
			token{kind: op}, left, right),
		pos: position,
	}
}

func intBinaryOp(op tokKind, left IntValue, right IntValue) (Value, *runtimeError) {
	switch op {
	case minus:
		return IntValue(left - right), nil
	case plus:
		return IntValue(left + right), nil
	default:
		return nil, incompatibleError(op, left, right, pos{})
	}
}

func stringBinaryOp(op tokKind, left StringValue, right StringValue) (Value, *runtimeError) {
	switch op {
	case plusString:
		x := append(left, right...)
		return StringValue(x), nil
	default:
		return nil, incompatibleError(op, left, right, pos{})
	}
}

func (c *Context) evalBinaryNode(n binaryNode, sc scope) (Value, *runtimeError) {
	leftComputed, err := c.evalExpr(n.left, sc)
	if err != nil {
		return nil, err
	}
	rightComputed, err := c.evalExpr(n.right, sc)
	if err != nil {
		return nil, err
	}
	if n.op == eq {
		return BoolValue(leftComputed.Eq(rightComputed)), nil
	}
	// TODO: add neq (!=)
	switch left := leftComputed.(type) {
	case IntValue:
		// TODO: uncomment to support float binary operations for float values
		right, ok := rightComputed.(IntValue)
		if !ok {
			// rightFloat, ok := rightComputed.(FloatValue)
			// if !ok {
			return nil, incompatibleError(n.op, leftComputed, rightComputed, n.pos())
			// }
			//
			// leftFloat := FloatValue(float64(int64(left)))
			// val, err := floatBinaryOp(n.op, leftFloat, rightFloat)
			// if err != nil {
			// 	err.pos = n.pos()
			// }
			// return val, err
		}

		val, err := intBinaryOp(n.op, left, right)
		if err != nil {
			err.pos = n.pos()
		}
		return val, err
	case StringValue:
		right, ok := rightComputed.(StringValue)
		if !ok {
			return nil, incompatibleError(n.op, leftComputed, rightComputed, n.pos())
		}
		val, err := stringBinaryOp(n.op, left, right)
		if err != nil {
			err.pos = n.pos()
		}
		return val, err
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Binary operator %s is not defined for values %s, %s",
				token{kind: n.op}, leftComputed, rightComputed),
			pos: n.pos(),
		}
	}
}

func (c *Context) evalExpr(node astNode, sc scope) (Value, *runtimeError) {
	switch n := node.(type) {
	case intNode:
		return IntValue(n.payload), nil
	case floatNode:
		return FloatValue(n.payload), nil
	case stringNode:
		return StringValue(n.payload), nil
	case binaryNode:
		return c.evalBinaryNode(n, sc)
	case identifierNode:
		val, err := sc.get(n.payload)
		if err != nil {
			err.pos = n.pos()
		}
		return val, err
	case assignmentNode:
		assignedValue, err := c.evalExpr(n.right, sc)
		if err != nil {
			return nil, err
		}
		switch left := n.left.(type) {
		case identifierNode:
			sc.put(left.payload, assignedValue)
			return assignedValue, nil
		default:
			return nil, &runtimeError{
				reason: fmt.Sprintf("Invalid assignment target %s", left.String()),
				pos:    n.pos(),
			}
		}
	}
	panic(fmt.Sprintf("Unexpected astNode type: %s", node))
}

func (c *Context) evalNodes(nodes []astNode) (Value, *runtimeError) {
	var returnValue Value = nil
	var err *runtimeError
	for _, expr := range nodes {
		returnValue, err = c.evalExpr(expr, c.scope)
		if err != nil {
			return nil, err
		}
	}
	return returnValue, nil
}
