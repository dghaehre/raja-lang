package main

import (
	"fmt"
	"io"
)

// TODO: we want to collect all of the typecheck errors instead of failing once we hit one.
type typecheckError struct {
	reason string
	pos
}

func (e *typecheckError) Error() string {
	head := colorize(ColorRed, "Type error")
	return fmt.Sprintf("%s at %s:\n%s", head, e.pos, e.reason)
}

type typecheckScope struct {
	parent *typecheckScope

	// vars needs to be extended to handle multiple functions with the same name
	vars map[string]astNode
}

func (sc *typecheckScope) put(name string, v astNode, pos pos) *typecheckError {
	sc.vars[name] = v
	return nil
}

func (sc *typecheckScope) get(name string) (astNode, *typecheckError) {
	if v, ok := sc.vars[name]; ok {
		return v, nil
	}
	if sc.parent != nil {
		return sc.parent.get(name)
	}
	return nil, &typecheckError{
		reason: fmt.Sprintf("%s is not defined", name),
	}
}

type TypecheckContext struct {
	typecheckScope
}

func NewTypecheckContext() TypecheckContext {
	return TypecheckContext{
		typecheckScope: typecheckScope{
			parent: nil,
			vars:   map[string]astNode{},
		},
	}
}

func isNum(ast astNode) bool {
	switch ast.(type) {
	case intNode, floatNode:
		return true
	}
	return false
}

func isString(ast astNode) bool {
	switch ast.(type) {
	case stringNode:
		return true
	}
	return false
}

func isList(ast astNode) bool {
	switch ast.(type) {
	case listNode:
		return true
	}
	return false
}

func isBool(ast astNode) bool {
	switch ast.(type) {
	case boolNode:
		return true
	}
	return false
}

func isIterator(ast astNode) bool {
	switch ast.(type) {
	case listNode, stringNode:
		return true
	}
	return false
}

func anyUnknowns(l ...astNode) bool {
	for _, v := range l {
		if v.TypeName() == "unknown" {
			return true
		}
	}
	return false
}

func (c *TypecheckContext) typecheckFnCallNode(n fnCallNode, args []astNode, sc typecheckScope) (astNode, *typecheckError) {
	// What to do here?
	// try to find a function in scope that has the right amount of parameters.
	return n, nil

}

func (c *TypecheckContext) typecheckBinaryNode(n binaryNode, sc typecheckScope) (astNode, *typecheckError) {
	leftComputed, err := c.typecheckExpr(n.left, sc)
	if err != nil {
		return nil, err
	}
	rightComputed, err := c.typecheckExpr(n.right, sc)
	if err != nil {
		return nil, err
	}

	// NOTE: this is just to make sure we dont acidentally say something is wrong when it isnt.
	// Should be removed eventually.
	if anyUnknowns(leftComputed, rightComputed) {
		return n, nil
	}
	switch n.op {
	case and, or:
		if !isBool(leftComputed) || !isBool(rightComputed) {
			return nil, &typecheckError{
				reason: fmt.Sprintf("%s operator only works with bool. %s and %s was used", n, leftComputed, rightComputed),
				pos:    n.pos(),
			}
		}
		return n, nil
	case plus, divide, modulus, times:
		if !isNum(leftComputed) || !isNum(rightComputed) {
			return nil, &typecheckError{
				reason: fmt.Sprintf("%s operator only works with ints and floats. %s and %s was used", n, leftComputed.TypeName(), rightComputed.TypeName()),
				pos:    n.pos(),
			}
		}
		return n, nil
	case plusOther:
		if !isIterator(leftComputed) || !isIterator(rightComputed) {
			return nil, &typecheckError{
				reason: fmt.Sprintf("++ operator only works with iterators (list and string). %s and %s was used", leftComputed.TypeName(), rightComputed.TypeName()),
				pos:    n.pos(),
			}
		}
		return n, nil
	default:
		return n, nil
	}
}

func (c *TypecheckContext) typecheckExpr(node astNode, sc typecheckScope) (astNode, *typecheckError) {
	switch n := node.(type) {
	case binaryNode:
		return c.typecheckBinaryNode(n, sc)
	case identifierNode:
		val, err := sc.get(n.payload)
		if err != nil {
			err.pos = n.pos()
		}
		return val, err
	case assignmentNode:
		assignedNode, err := c.typecheckExpr(n.right, sc)
		if err != nil {
			return nil, err
		}
		switch left := n.left.(type) {
		case identifierNode:
			err := sc.put(left.payload, assignedNode, n.pos())
			return assignedNode, err
		default:
			return nil, &typecheckError{
				reason: fmt.Sprintf("Invalid assignment target %s", left.String()),
				pos:    n.pos(),
			}
		}
	case blockNode:
		blockScope := typecheckScope{
			parent: &sc,
			vars:   map[string]astNode{},
		}

		last := len(n.exprs) - 1
		for _, expr := range n.exprs[:last] {
			_, err := c.typecheckExpr(expr, blockScope)
			if err != nil {
				return nil, err
			}
		}
		return c.typecheckExpr(n.exprs[last], blockScope)
	case fnCallNode:
		args := make([]astNode, 0, len(n.args))
		for _, a := range n.args {
			v, err := c.typecheckExpr(a, sc)
			if err != nil {
				return nil, err
			}
			args = append(args, v)
		}
		return c.typecheckFnCallNode(n, args, sc)
	default:
		return n, nil
	}
}

func (c *TypecheckContext) typecheckNodes(nodes []astNode) (astNode, *typecheckError) {
	var returnValue astNode = nil
	var err *typecheckError
	for _, expr := range nodes {
		returnValue, err = c.typecheckExpr(expr, c.typecheckScope)
		if err != nil {
			return nil, err
		}
	}
	return returnValue, nil
}

func (c *TypecheckContext) Typecheck(reader io.Reader, filename string) (astNode, error) {
	program, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	tokenizer := newTokenizer(string(program), filename)
	tokens := tokenizer.tokenize()
	parser := newParser(tokens)
	nodes, err := parser.parse()
	if err != nil {
		return nil, err
	}
	v, typecheckErr := c.typecheckNodes(nodes)
	if typecheckErr != nil {
		return nil, typecheckErr
	}
	return v, nil
}
