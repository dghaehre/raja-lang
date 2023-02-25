package main

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	color "github.com/dghaehre/termcolor"
)

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
	fns          []typedFnNode // Must be more than 0
	argsProvided typedArgs
	pos
}

func (e paramMismatchError) Error() string {
	head := color.Str(color.Red, "Param mismatch")
	reason := ""
	if len(e.fns) == 1 {
		fnMatch := e.fns[0]
		reason = fmt.Sprintf("%s has 1 implementation at %s and is expecting: %s\n\nBut was provided: %s", e.callNode.fn, fnMatch.pos(), fnMatch.args, e.argsProvided)
	} else {
		reason = fmt.Sprintf("%s has %d implementations:\n", e.callNode.fn, len(e.fns))
		for _, fn := range e.fns {
			reason += fmt.Sprintf("%s\n", fn)
		}
		reason += fmt.Sprintf("\nBut was provided: %s", e.argsProvided)
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

	if len(me.errors) > 1 {
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

// TODO:
// - changing a mutable variable
func (sc *typecheckScope) put(name string, typed typedAstNode, pos pos) error {
	switch n := typed.(type) {
	case typedFnNode:
		scvalue, ok := sc.vars[name]
		if !ok {
			sc.vars[name] = typedFnNodes{
				values: []typedFnNode{n},
			}
			return nil
		}
		switch scvalue := scvalue.(type) {
		case typedFnNodes:
			scvalue.values = append(scvalue.values, n)
			sc.vars[name] = scvalue
			return nil
		default:
			return &typecheckError{
				reason: fmt.Sprintf("Expected fn value (TODO)"),
				pos:    pos,
			}
		}
	default:
		// TODO: mutable?
		sc.vars[name] = typed
	}
	return nil
}

func (sc *typecheckScope) get(name string, pos pos) (typedAstNode, error) {
	if v, ok := sc.vars[name]; ok {
		return v, nil
	}
	if sc.parent != nil {
		return sc.parent.get(name, pos)
	}

	// TODO: what if the variable is defined later?
	return nil, &typecheckError{
		reason: fmt.Sprintf("%s is not defined", name),
		pos:    pos,
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

type typedArg struct {
	name  string
	alias typedAstNode
}

func (a typedArg) String() string {
	return fmt.Sprintf("%s:%s", a.name, a.alias)
}

func (a typedArg) pos() pos {
	return pos{}
}

type untypedArg struct {
	name string
}

func (a untypedArg) String() string {
	return a.name
}

func (a untypedArg) pos() pos {
	return pos{}
}

type typedArgs []typedAstNode

func (args typedArgs) String() string {
	res := "("
	for i, a := range args {
		res += a.String()
		if i != len(args)-1 {
			res += ", "
		}
	}
	res += ")"
	return res
}

type typedEnumNode struct {
	parent string
	name   string
	args   typedArgs
	tok    *token
}

func (n typedEnumNode) String() string {
	if n.name == "" {
		return n.parent
	}
	return fmt.Sprintf("%s::%s", n.parent, n.name)
}

func (n typedEnumNode) pos() pos {
	return n.tok.pos
}

// Usecase for the any type:
//   - when we don't know the type of a variable,
//     and because our typechecking is note checking everything
//   - when we encounter an error, we can return with a typedAnyNode
//     and we will not cause anymore errors down the chain
type typedAnyNode struct{}

func (n typedAnyNode) String() string {
	return "Any"
}

func (n typedAnyNode) pos() pos {
	return pos{}
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
	args typedArgs
	body typedAstNode
}

func (n typedFnNode) String() string {
	return fmt.Sprintf("(%s) => {}", n.args)
}

func (n typedFnNode) pos() pos {
	return n.tok.pos
}

type typedFnNodes struct {
	values []typedFnNode
}

func (v typedFnNodes) String() string {
	stringValues := make([]string, len(v.values))
	for i, s := range v.values {
		stringValues[i] = s.String()
	}
	return strings.Join(stringValues, ", ")
}

// NOTE: Should never be used
func (v typedFnNodes) pos() pos {
	return pos{}
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

func toMaybeTypedArgs(args []Arg) []typedAstNode {
	typed := make(typedArgs, 0)
	for _, arg := range args {
		if arg.alias == "" {
			typed = append(typed, untypedArg{
				name: arg.name,
			})
			continue
		}
		var alias typedAstNode
		switch arg.alias {
		case "Int":
			alias = typedIntNode{}
		case "Str":
			alias = typedStringNode{}
		default:
			alias = typedEnumNode{
				parent: arg.alias,
				name:   "",
				tok:    nil,
			}
		}

		typed = append(typed, typedArg{
			name:  arg.name,
			alias: alias,
		})
	}
	return typed
}

func isNum(typed typedAstNode) bool {
	switch typed.(type) {
	case typedIntNode, typedFloatNode:
		return true
	}
	return false
}

// Given a list of all Ints, return Int
// otherwise return Float
func getNumType(typed ...typedAstNode) typedAstNode {
	for _, t := range typed {
		_, ok := t.(typedIntNode)
		if !ok {
			return typedFloatNode{}
		}
	}
	return typedIntNode{}
}

// Given a list of all List, return List
// otherwise return Str
func getIteratorType(typed ...typedAstNode) typedAstNode {
	for _, t := range typed {
		_, ok := t.(typedListNode)
		if !ok {
			return typedStringNode{}
		}
	}
	return typedListNode{}
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

	switch nodes := fn.(type) {
	case typedFnNodes:
		// TODO:
		// - find the matching functions with the same amount of args
		// - find the matching function(s) with correct types
		//   - maybe a warning if there are multiple functions that matches?

		argsProvided := make([]typedAstNode, 0)
		for _, v := range callNode.args {
			arg, err := c.typecheckExpr(v, sc)
			if err != nil {
				return nil, err
			}
			argsProvided = append(argsProvided, arg)
		}

		matchingArgsLength := make([]typedFnNode, 0)
		for _, n := range nodes.values {
			if len(n.args) == len(argsProvided) {
				matchingArgsLength = append(matchingArgsLength, n)
			}
		}

		if len(matchingArgsLength) == 0 {
			c.errors = append(c.errors, &paramMismatchError{
				callNode:     callNode,
				argsProvided: argsProvided,
				fns:          nodes.values,
				pos:          callNode.pos(),
			})
			return typedAnyNode{}, nil
		}

		fullMatch := make([]typedFnNode, 0)
		for _, n := range matchingArgsLength {
			// TODO
			fullMatch = append(fullMatch, n)
		}

		if len(fullMatch) == 0 {
			c.errors = append(c.errors, &paramMismatchError{
				callNode:     callNode,
				argsProvided: argsProvided,
				fns:          nodes.values,
				pos:          callNode.pos(),
			})
			return typedAnyNode{}, nil
		}

		if len(fullMatch) > 1 {
			// TODO: maybe create a warning here that we are matching more than one?
		}

		return fullMatch[0], nil
	default:
		c.errors = append(c.errors, &typecheckError{
			reason: fmt.Sprintf("%s is not a function.", fn),
			pos:    callNode.pos(),
		})
	}

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
			// If we find an error, we return unknown to avoid more errors.
			return typedAnyNode{}, nil
		}
		return getNumType(leftComputed, rightComputed), nil
	case plusOther:
		if !isIterator(leftComputed) || !isIterator(rightComputed) {
			c.errors = append(c.errors, &typecheckError{
				reason: fmt.Sprintf("++ operator only works with iterators (list and string). %s and %s was used",
					color.Str(color.Yellow, leftComputed.String()), color.Str(color.Yellow, rightComputed.String())),
				pos: n.pos(),
			})
			return typedAnyNode{}, nil
		}
		return getIteratorType(leftComputed, rightComputed), nil // TODO
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
		return sc.get(n.payload, n.pos())
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
		fnScope := typecheckScope{
			parent: &sc,
			vars:   map[string]typedAstNode{},
		}
		args := toMaybeTypedArgs(n.args)
		for _, a := range args {
			arg, ok := a.(typedArg)
			if ok { // NOTE: Should always be a typed arg
				err := fnScope.put(arg.name, arg.alias, n.pos())
				if err != nil {
					return nil, err
				}
			}
		}

		body, err := c.typecheckExpr(n.body, fnScope)
		if err != nil {
			// If the body is not typechecking, we want to report that, but the function "signature" might still be 'correct'
			c.errors = append(c.errors, err)
		}
		return typedFnNode{
			args: args,
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
