package main

import (
	"fmt"
	"io"
	"reflect"
	"strconv"

	color "github.com/dghaehre/termcolor"
)

// TODO: use something like typedAstValue instead of typedAstNode.
// typedAstNode can be extended.

type typecheckError struct {
	reason string
	pos
}

func (e *typecheckError) Error() string {
	head := color.Str(color.Red, "Type error")
	return fmt.Sprintf("%s: at %s:\n%s", head, e.pos, e.reason)
}

type paramMismatchError struct {
	callNode     fnCallNode
	args         []Arg         // TODO: change this to typedAstNode or similar
	fns          []typedFnNode // Must be more than 0
	argsProvided []typedAstNode
	pos
}

func (e paramMismatchError) Error() string {
	head := color.Str(color.Red, "Param mismatch")
	reason := ""
	if len(e.fns) == 1 {
		reason = fmt.Sprintf("%s has 1 implementation at %s and is expecting the following:\n%s\n\nBut was provided: %s", e.callNode.fn, e.callNode.pos(), e.args, e.argsProvided)
	} else {
		reason = fmt.Sprintf("%s has %d implementations and is expecting one of the following:\n%s\n\nBut was provided: %s", e.callNode.fn, len(e.fns), e.args, e.argsProvided)
	}
	return fmt.Sprintf("%s: at %s:\n%s", head, e.pos, reason)
}

type multipleErrors struct {
	errors []error
}

func (me multipleErrors) Error() string {
	s := ""
	for i, v := range me.errors {
		if i > 0 {
			s += "\n\n"
		}
		s += v.Error()
	}

	if len(me.errors) > 0 {
    s += "\n\n" + color.Str(color.Red, "Errors: ")
		s += strconv.Itoa(len(me.errors))
	}
	return s
}

type typecheckScope struct {
	parent *typecheckScope

	// vars needs to be extended to handle multiple functions with the same name
	vars map[string]typedAstNode
}

func (sc *typecheckScope) put(name string, v typedAstNode, pos pos) error {
	sc.vars[name] = v
	return nil
}

func (sc *typecheckScope) get(name string) (typedAstNode, error) {
	if v, ok := sc.vars[name]; ok {
		return v, nil
	}
	if sc.parent != nil {
		return sc.parent.get(name)
	}

	// TODO: what if the variable is defined later?
	return nil, &typecheckError{
		reason: fmt.Sprintf("%s is not defined", name),
	}
}

type TypecheckContext struct {
	typecheckScope
	multipleErrors
}

func NewTypecheckContext() TypecheckContext {
	return TypecheckContext{
		typecheckScope: typecheckScope{
			parent: nil,
			vars:   map[string]typedAstNode{},
		},
	}
}

type typedAstNode interface {
	String() string
	pos() pos

	// Eq(typedAstNode) bool. We might not need this one..
	// payload We might not need this one either
}

type typedIntNode struct {
	tok *token
}

func (n typedIntNode) String() string {
	return "Int"
}

func (n typedIntNode) pos() pos {
	return n.tok.pos
}

type typedFloatNode struct {
	tok *token
}

func (f typedFloatNode) String() string {
	return "Float"
}

func (n typedFloatNode) pos() pos {
	return n.tok.pos
}

type typedStringNode struct {
	tok *token
}

func (s typedStringNode) String() string {
	return "Str"
}

func (s typedStringNode) pos() pos {
	return s.tok.pos
}

type typedListNode struct {
	tok *token
}

func (s typedListNode) String() string {
	return "List"
}

func (s typedListNode) pos() pos {
	return s.tok.pos
}

type typedFnNode struct {
	tok  *token
	args []Arg
	body typedAstNode
}

func (n typedFnNode) String() string {
	return fmt.Sprintf("Fn: (%s) => {}", StringsJoin(n.args, ", "))
}

func (n typedFnNode) pos() pos {
	return n.tok.pos
}

func isType(a typedAstNode, b typedAstNode) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

func isOneOfType(a typedAstNode, bs ...typedAstNode) bool {
	t := reflect.TypeOf(a)
	for _, v := range bs {
		if t == reflect.TypeOf(v) {
			return true
		}
	}
	return false
}

// TODO: this needs to be fleshed out..
// TODO: handle underscore?
func matchingArgs(as []typedAstNode, bs []typedAstNode) bool {
	if len(as) != len(bs) {
		return false
	}
	for i := 0; i < len(as); i++ {
		if reflect.TypeOf(as[i]) != reflect.TypeOf(bs[i]) {
			return false
		}
	}
	return true
}

// TODO
func toTypedArgs(args []Arg) []typedAstNode {
	return []typedAstNode{}
}

func isNum(typed typedAstNode) bool {
	switch typed.(type) {
	case typedIntNode, typedFloatNode:
		return true
	}
	return false
}

func isString(ast typedAstNode) bool {
	switch ast.(type) {
	case typedStringNode:
		return true
	}
	return false
}

func isList(ast typedAstNode) bool {
	switch ast.(type) {
	case listNode:
		return true
	}
	return false
}

func isBool(ast typedAstNode) bool {
	switch ast.(type) {
	case boolNode:
		return true
	}
	return false
}

func isIterator(ast typedAstNode) bool {
	switch ast.(type) {
	case typedListNode, typedStringNode:
		return true
	}
	return false
}

// TODO: remove
func anyUnknowns(l ...typedAstNode) bool {
	for _, v := range l {
		if v.String() == "unknown" {
			return true
		}
	}
	return false
}

func (c *TypecheckContext) typecheckFnCallNode(callNode fnCallNode, sc typecheckScope) (typedAstNode, error) {
	fn, err := c.typecheckExpr(callNode.fn, sc)
	if err != nil {
		return nil, err
	}

	// TODO: multiple dispatch!!

	// args := n.args
	switch f := fn.(type) {
	case typedFnNode:
		argsProvided := make([]typedAstNode, 0)
		for _, v := range callNode.args {
			arg, err := c.typecheckExpr(v, sc)
			if err != nil {
				return nil, err
			}
			argsProvided = append(argsProvided, arg)
		}

		// argsExpected := TODO

		if !matchingArgs(argsProvided, toTypedArgs(f.args)) {
			c.errors = append(c.errors, &paramMismatchError{
				callNode:     callNode,
				argsProvided: argsProvided,
				args:         f.args,
				fns:          []typedFnNode{f},
				pos:          callNode.pos(),
			})
			return callNode, nil
		}
		// fmt.Println(fn)
	default:
		c.errors = append(c.errors, &typecheckError{
			reason: fmt.Sprintf("%s is not a function.", fn),
			pos:    callNode.pos(),
		})
	}

	// What to do here?
	// try to find a function in scope that has the right amount of parameters.
	return callNode, nil

}

func (c *TypecheckContext) typecheckBinaryNode(n binaryNode, sc typecheckScope) (typedAstNode, error) {
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
			c.errors = append(c.errors, &typecheckError{
				reason: fmt.Sprintf("%s operator only works with bool. %s and %s was used", n, leftComputed, rightComputed),
				pos:    n.pos(),
			})
		}
		return n, nil
	case plus, divide, modulus, times:
		if !isNum(leftComputed) || !isNum(rightComputed) {
			c.errors = append(c.errors, &typecheckError{
				reason: fmt.Sprintf("%s operator only works with ints and floats. %s and %s was used",
					n.tok, color.Str(color.Yellow, leftComputed.String()), color.Str(color.Yellow, rightComputed.String())),
				pos: n.pos(),
			})
		}
		return n, nil
	case plusOther:
		if !isIterator(leftComputed) || !isIterator(rightComputed) {
			c.errors = append(c.errors, &typecheckError{
				reason: fmt.Sprintf("++ operator only works with iterators (list and string). %s and %s was used",
					color.Str(color.Yellow, leftComputed.String()), color.Str(color.Yellow, rightComputed.String())),
				pos: n.pos(),
			})
		}
		return n, nil
	default:
		return n, nil
	}
}

// typecheckExpr is the only function that does not 'insert' typecheckError into TypecheckContext.
// This means that we can insert typeccheckError at the boundaries like `typecheckNodes` which is at the "beginnig" for parsing
// a root node, and like typecheckBinaryNode which is at "the end".
func (c *TypecheckContext) typecheckExpr(node astNode, sc typecheckScope) (typedAstNode, error) {
	switch n := node.(type) {
	case intNode:
		return typedIntNode{
			tok: n.tok,
		}, nil
	case floatNode:
		return typedFloatNode{
			tok: n.tok,
		}, nil
	case stringNode:
		return typedStringNode{
			tok: n.tok,
		}, nil
	// case underscoreNode:
	// 	return underscorevalue, nil
	// case boolNode:
	// 	return BoolValue(n.payload), nil
	// case matchNode:
	// 	return c.evalMatchNode(n, sc)
	case binaryNode:
		return c.typecheckBinaryNode(n, sc)
	case identifierNode:
		return sc.get(n.payload)
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
			vars:   map[string]typedAstNode{},
		}

		last := len(n.exprs) - 1
		for _, expr := range n.exprs[:last] {
			_, err := c.typecheckExpr(expr, blockScope)
			if err != nil {
				return nil, err
			}
		}
		return c.typecheckExpr(n.exprs[last], blockScope)
	case fnNode:
		body, err := c.typecheckExpr(n.body, sc)
		if err != nil {
			// If the body is not typechecking, we want to report that, but the function "signature" might still be 'correct'
			c.errors = append(c.errors, err)
		}
		return typedFnNode{
			args: n.args, // TODO: handle this here?
			tok:  n.tok,
			body: body,
		}, nil
	case fnCallNode:
		return c.typecheckFnCallNode(n, sc)
	default:
		// TODO: remove default when we have handled everything
		// This is just a pillow
		return n, nil
	}
}

func (c *TypecheckContext) typecheckNodes(nodes []astNode) (typedAstNode, error) {
	var returnValue typedAstNode = nil
	for _, expr := range nodes {
		v, err := c.typecheckExpr(expr, c.typecheckScope)
		if err != nil {
			c.errors = append(c.errors, err)
		} else {
			returnValue = v
		}
	}
	if len(c.errors) > 0 {
		return nil, c.multipleErrors
	}
	return returnValue, nil
}

func (c *TypecheckContext) Typecheck(reader io.Reader, filename string) (typedAstNode, error) {
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
