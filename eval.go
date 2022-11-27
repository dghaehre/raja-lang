package main

import (
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

// Scope

func (sc *scope) put(name string, v Value) {
	sc.vars[name] = v
}

// update "name" with new Value
func (sc *scope) update(name string, v Value) *runtimeError {
	if _, ok := sc.vars[name]; ok {
		sc.vars[name] = v
		return nil
	}
	if sc.parent != nil {
		return sc.parent.update(name, v)
	}
	return &runtimeError{
		reason: fmt.Sprintf("%s is undefined", name),
	}
}

// Eval

func (c *Context) Eval(reader io.Reader) error {
	program, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	tokenizer := newTokenizer(string(program))
	tokens := tokenizer.tokenize()
	// fmt.Printf("tokens: %v\n", tokens)
	parser := newParser(tokens)
	nodes, err := parser.parse()
	if err != nil {
		return err
	}
	// fmt.Println("Parsed:")
	// for _, n := range nodes {
	// 	fmt.Println(n)
	// }
	v, runtimeErr := c.evalNodes(nodes)
	if runtimeErr != nil {
		return runtimeErr
	}
	if v != nil {
		fmt.Println(v)
	}
	return nil
}

func (c *Context) evalExpr(node astNode, sc scope) (Value, *runtimeError) {
	switch n := node.(type) {
	case intNode:
		return IntValue(n.payload), nil
	case floatNode:
		return FloatValue(n.payload), nil
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
	var returnVal Value = nil
	var err *runtimeError
	for _, expr := range nodes {
		returnVal, err = c.evalExpr(expr, c.scope)
		// fmt.Println(returnVal)
		if err != nil {
			return nil, err
		}
	}
	return returnVal, nil

}
