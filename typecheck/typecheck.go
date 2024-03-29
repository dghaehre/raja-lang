package typecheck

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"dghaehre/raja/ast"
	"dghaehre/raja/lib"

	color "github.com/dghaehre/termcolor"
)

type typecheckError struct {
	reason string
	ast.Pos
}

func (e typecheckError) Error() string {
	head := color.Str(color.Red, "Type error")
	return fmt.Sprintf("%s: at %s:\n%s", head, e.Pos, e.reason)
}

type paramMismatchError struct {
	callNode     ast.FnCallNode
	fns          []typedFnNode // Must be more than 0, unless the error does not make sense
	argsProvided typedArgs
	ast.Pos
}

func (e paramMismatchError) Error() string {
	head := color.Str(color.Red, "Parameter mismatch in function call")
	reason := ""
	if len(e.fns) == 1 {
		fnMatch := e.fns[0]
		reason = fmt.Sprintf("%s has 1 implementation at %s\n", e.callNode.Fn, fnMatch.pos())
	} else {
		reason = fmt.Sprintf("%s has %d implementations:\n", e.callNode.Fn, len(e.fns))
	}
	for _, fn := range e.fns {
		reason += fmt.Sprintf("%s\n", fn)
	}
	reason += fmt.Sprintf("\nBut was provided: %s at %s", e.argsProvided, e.Pos)
	return fmt.Sprintf("%s\n%s", head, reason)
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
	vars map[string]TypedAstNode

	// used to keep track of recursion
	currentFn string
}

func (sc *typecheckScope) putCurrentFn(name string) {
	sc.currentFn = name
}

func (sc *typecheckScope) isRecursion(name string) bool {
	if sc.currentFn == name {
		return true
	}
	if sc.parent == nil {
		return false
	}
	return sc.parent.isRecursion(name)
}

// TODO:
// - changing a mutable variable
func (sc *typecheckScope) put(name string, typed TypedAstNode, pos ast.Pos) error {
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
				Pos:    pos,
			}
		}
	default:
		// TODO: mutable?
		sc.vars[name] = typed
	}
	return nil
}

func (sc *typecheckScope) get(name string, pos ast.Pos) (TypedAstNode, error) {
	if v, ok := sc.vars[name]; ok {
		return v, nil
	}
	if sc.parent != nil {
		return sc.parent.get(name, pos)
	}

	return nil, &typecheckError{
		reason: fmt.Sprintf("%s is not defined", name),
		Pos:    pos,
	}
}

type TypecheckContext struct {
	typecheckScope
	multipleErrors
}

func NewTypecheckContext() TypecheckContext {
	return TypecheckContext{
		typecheckScope: typecheckScope{
			parent:    nil,
			vars:      map[string]TypedAstNode{},
			currentFn: "",
		},
	}
}

type TypedAstNode interface {
	String() string
	pos() ast.Pos

	Eq(TypedAstNode) bool
	// payload We might not need this one either
}

type typedArg struct {
	name  string
	alias TypedAstNode
}

func (a typedArg) String() string {
	return fmt.Sprintf("%s:%s", a.name, a.alias)
}

func (a typedArg) pos() ast.Pos {
	return ast.Pos{}
}

// TODO: alias should never be nil
// so this isnt really necesarry
func (a typedArg) Eq(b TypedAstNode) bool {
	if a.alias != nil {
		return a.alias.Eq(b)
	} else {
		return true
	}
}

type untypedArg struct {
	name string
}

func (a untypedArg) String() string {
	return a.name
}

func (a untypedArg) pos() ast.Pos {
	return ast.Pos{}
}

func (a untypedArg) Eq(b TypedAstNode) bool {
	return true
}

type typedArgs []TypedAstNode

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
	tok    *ast.Token
}

func (n typedEnumNode) String() string {
	if n.name == "" {
		return n.parent
	}
	if len(n.args) > 0 {
		return fmt.Sprintf("%s::%s%s", n.parent, n.name, n.args)
	}
	return fmt.Sprintf("%s::%s", n.parent, n.name)
}

func (n typedEnumNode) pos() ast.Pos {
	if n.tok != nil {
		return n.tok.Pos
	}
	return ast.Pos{}
}

func (a typedEnumNode) Eq(b TypedAstNode) bool {
	switch b := b.(type) {
	case typedAnyNode:
		return true
	case typedEnumNode:
		return b.parent == a.parent && b.name == a.name
	case typedAliasNode:
		return b.Eq(a)
	}
	return false
}

type typedAliasNode struct {
	name    string
	targets []TypedAstNode
}

func (n typedAliasNode) String() string {
	if n.name != "" {
		return n.name
	}
	stringValues := make([]string, len(n.targets))
	for i, s := range n.targets {
		stringValues[i] = s.String()
	}
	if len(stringValues) <= 1 {
		return fmt.Sprintf("%s", strings.Join(stringValues, " | "))
	}
	return fmt.Sprintf("(%s)", strings.Join(stringValues, " | "))
}

func (n typedAliasNode) pos() ast.Pos {
	return ast.Pos{}
}

func (a typedAliasNode) Eq(b TypedAstNode) bool {
	for _, t := range a.targets {
		if t.Eq(b) {
			return true
		}
	}
	return false
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

func (n typedAnyNode) pos() ast.Pos {
	return ast.Pos{}
}

func (a typedAnyNode) Eq(b TypedAstNode) bool {
	return true
}

// Usecase for AnyFn:
// when user creates map = (a, f:Fn) => ...
type typedAnyFnNode struct{}

func (n typedAnyFnNode) String() string {
	return "Fn"
}

func (n typedAnyFnNode) pos() ast.Pos {
	return ast.Pos{}
}

func (a typedAnyFnNode) Eq(b TypedAstNode) bool {
	switch b := b.(type) {
	case typedAnyNode, typedFnNode, typedAnyFnNode, typedFnNodes:
		return true
	case typedAliasNode:
		return b.Eq(a)
	default:
		return false
	}
}

type typedIntNode struct {
	tok *ast.Token
}

func (n typedIntNode) String() string {
	return "Int"
}

func (n typedIntNode) pos() ast.Pos {
	if n.tok != nil {
		return n.tok.Pos
	}
	return ast.Pos{}
}

func (a typedIntNode) Eq(b TypedAstNode) bool {
	switch b.(type) {
	case typedAnyNode, typedIntNode:
		return true
	case typedAliasNode:
		return b.Eq(a)
	default:
		return false
	}
}

type typedFloatNode struct {
	tok *ast.Token
}

func (f typedFloatNode) String() string {
	return "Float"
}

func (n typedFloatNode) pos() ast.Pos {
	if n.tok != nil {
		return n.tok.Pos
	}
	return ast.Pos{}
}

func (a typedFloatNode) Eq(b TypedAstNode) bool {
	switch b.(type) {
	case typedAnyNode, typedFloatNode:
		return true
	case typedAliasNode:
		return b.Eq(a)
	default:
		return false
	}
}

type typedBoolNode struct {
	tok *ast.Token
}

func (n typedBoolNode) String() string {
	return "Bool"
}

func (n typedBoolNode) pos() ast.Pos {
	if n.tok != nil {
		return n.tok.Pos
	}
	return ast.Pos{}
}

func (a typedBoolNode) Eq(b TypedAstNode) bool {
	switch b.(type) {
	case typedAnyNode, typedBoolNode:
		return true
	case typedAliasNode:
		return b.Eq(a)
	default:
		return false
	}
}

type typedStringNode struct {
	tok *ast.Token
}

func (s typedStringNode) String() string {
	return "Str"
}

func (s typedStringNode) pos() ast.Pos {
	if s.tok != nil {
		return s.tok.Pos
	}
	return ast.Pos{}
}

func (a typedStringNode) Eq(b TypedAstNode) bool {
	switch b := b.(type) {
	case typedAnyNode, typedStringNode:
		return true
	case typedAliasNode:
		return b.Eq(a)
	default:
		return false
	}
}

type typedListNode struct {
	tok *ast.Token
}

func (s typedListNode) String() string {
	return "List"
}

func (s typedListNode) pos() ast.Pos {
	if s.tok != nil {
		return s.tok.Pos
	}
	return ast.Pos{}
}

func (a typedListNode) Eq(b TypedAstNode) bool {
	switch b.(type) {
	case typedAnyNode, typedListNode:
		return true
	case typedAliasNode:
		return b.Eq(a)
	default:
		return false
	}
}

type typedFnNode struct {
	tok  *ast.Token
	args typedArgs
	body TypedAstNode
}

func (n typedFnNode) String() string {
	return fmt.Sprintf("(%s) => {}", n.args)
}

func (n typedFnNode) pos() ast.Pos {
	if n.tok != nil {
		return n.tok.Pos
	}
	return ast.Pos{}
}

func (a typedFnNode) Eq(b TypedAstNode) bool {
	switch b.(type) {
	case typedAnyNode, typedFnNode:
		return true
	case typedAliasNode:
		return b.Eq(a)
	default:
		return false
	}
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
func (v typedFnNodes) pos() ast.Pos {
	panic("typedFnnode.pos() should never be used")
}

// NOTE: Should never be used
func (a typedFnNodes) Eq(b TypedAstNode) bool {
	panic("typedFnnode.Eq() should never be used")
}

func isOneOfType(a TypedAstNode, bs ...TypedAstNode) bool {
	t := reflect.TypeOf(a)
	for _, v := range bs {
		if t == reflect.TypeOf(v) {
			return true
		}
	}
	return false
}

func (c *TypecheckContext) toMaybeTypedArgs(args []ast.Arg, sc typecheckScope) []TypedAstNode {
	typed := make(typedArgs, 0)
	for _, arg := range args {
		if arg.Alias == "" {
			typed = append(typed, untypedArg{
				name: arg.Name,
			})
			continue
		}
		a, err := sc.get(arg.Alias, ast.Pos{})
		if err != nil {
			c.errors = append(c.errors, err)
			a = typedAnyNode{}
		}
		typed = append(typed, typedArg{
			name:  arg.Name,
			alias: a,
		})
	}
	return typed
}

func isNum(typed TypedAstNode) bool {
	switch n := typed.(type) {
	case typedIntNode, typedFloatNode, typedAnyNode:
		return true
	case typedAliasNode:
		return n.Eq(typedIntNode{}) || n.Eq(typedFloatNode{})
	}
	return false
}

func isAliasWithName(t TypedAstNode, name string) bool {
	a, ok := t.(typedAliasNode)
	if !ok {
		return false
	}
	return a.name == name
}

// Expects left and right to be either
// - Int
// - Float
// - Int (alias)
// - Float (alias)
// - Num (alias)

// This is ugly as fuck...
// Relying on tests here to make this consistent
func getNumTypeFromBinOp(left TypedAstNode, right TypedAstNode) TypedAstNode {
	switch left := left.(type) {
	case typedIntNode:
		if isAliasWithName(right, "Int") {
			return intAlias
		}
		if isAliasWithName(right, "Float") {
			return floatAlias
		}
		_, ok := right.(typedIntNode)
		if ok {
			return typedIntNode{}
		}
		_, ok = right.(typedFloatNode)
		if ok {
			return typedFloatNode{}
		}
	case typedAliasNode:
		if left.name == "Int" {
			if isAliasWithName(right, "Int") {
				return intAlias
			}
			_, ok := right.(typedIntNode)
			if ok {
				return intAlias
			}
			if isAliasWithName(right, "Float") {
				return floatAlias
			}
			_, ok = right.(typedFloatNode)
			if ok {
				return floatAlias
			}
		}
		if left.name == "Float" {
			if isAliasWithName(right, "Int") || isAliasWithName(right, "Float") {
				return floatAlias
			}
		}
	case typedFloatNode:
		if isAliasWithName(right, "Int") || isAliasWithName(right, "Float") {
			return floatAlias
		}
		_, ok := right.(typedFloatNode)
		if ok {
			return typedFloatNode{}
		}
		_, ok = right.(typedIntNode)
		if ok {
			return typedFloatNode{}
		}
	}
	return numAlias
}

func getTypeFromMatchBodies(types []TypedAstNode) TypedAstNode {
	typedNodes := []TypedAstNode{}
	for _, t := range types {
		_, isAny := t.(typedAnyNode)
		if isAny {
			return typedAnyNode{}
		}
		if !isOneOfType(t, typedNodes...) {
			typedNodes = append(typedNodes, t)
		}
	}
	return typedAliasNode{
		targets: typedNodes,
	}
}

// Given a list of all List, return List
// otherwise return Str
func getIteratorType(typed ...TypedAstNode) TypedAstNode {
	for _, t := range typed {
		_, ok := t.(typedListNode)
		if !ok {
			return typedStringNode{}
		}
	}
	return typedListNode{}
}

func isString(ast TypedAstNode) bool {
	switch ast.(type) {
	case typedStringNode:
		return true
	}
	return false
}

func isList(a TypedAstNode) bool {
	switch a.(type) {
	case typedListNode:
		return true
	}
	return false
}

func isBool(a TypedAstNode) bool {
	switch a.(type) {
	case typedBoolNode:
		return true
	}
	return false
}

func isIterator(ast TypedAstNode) bool {
	switch n := ast.(type) {
	case typedListNode, typedStringNode, typedAnyNode:
		return true
	case typedAliasNode:
		return n.Eq(typedStringNode{}) || n.Eq(typedListNode{})
	}
	return false
}

// TODO: remove
func anyUnknowns(l ...TypedAstNode) bool {
	for _, v := range l {
		if v.String() == "unknown" {
			return true
		}
	}
	return false
}

func (c *TypecheckContext) typecheckFnCallNode(callNode ast.FnCallNode, sc typecheckScope) (TypedAstNode, error) {
	fn, err := c.typecheckExpr(callNode.Fn, sc)
	if err != nil {
		i, isIdentifier := callNode.Fn.(ast.IdentifierNode)
		if isIdentifier {
			if sc.isRecursion(i.Payload) {
				return typedAnyNode{}, nil
			}
		}
		return nil, err
	}

	switch n := fn.(type) {
	case typedFnNodes:
		// TODO:
		// - find the matching functions with the same amount of args
		// - find the matching function(s) with correct types
		//   - maybe a warning if there are multiple functions that matches?
		argsProvided := make([]TypedAstNode, 0)
		for _, v := range callNode.Args {
			arg, err := c.typecheckExpr(v, sc)
			if err != nil {
				return nil, err
			}
			argsProvided = append(argsProvided, arg)
		}

		// matchingArgsLength is the functions that have the same args length
		matchingArgsLength := make([]typedFnNode, 0)
		for _, n := range n.values {
			if len(n.args) == len(argsProvided) {
				matchingArgsLength = append(matchingArgsLength, n)
			}
		}

		if len(matchingArgsLength) == 0 {
			c.errors = append(c.errors, &paramMismatchError{
				callNode:     callNode,
				argsProvided: argsProvided,
				fns:          n.values,
				Pos:          callNode.Pos(),
			})
			return typedAnyNode{}, nil
		}

		// fullMatch is list of functions where the length of args is 'same'
		// and "type" given is also correct
		// fullMatch := callNode.Args.Eq(nodes.values)
		fullMatch := make([]typedFnNode, 0)
		for i := 0; i < len(matchingArgsLength); i++ {
			argsMatching := true
			for j := 0; j < len(argsProvided); j++ {
				if !matchingArgsLength[i].args[j].Eq(argsProvided[j]) {
					argsMatching = false
				}
			}
			if argsMatching {
				fullMatch = append(fullMatch, matchingArgsLength[i])
			}
		}

		if len(fullMatch) == 0 {
			c.errors = append(c.errors, &paramMismatchError{
				callNode:     callNode,
				argsProvided: argsProvided,
				fns:          n.values,
				Pos:          callNode.Pos(),
			})
			return typedAnyNode{}, nil
		}

		if len(fullMatch) > 1 {
			// TODO: maybe create a warning here that we are matching more than one?
		}
		return fullMatch[0].body, nil
	case typedAnyNode, typedAliasNode, typedAnyFnNode:
		// ^ Some of these might need some improvement
		return typedAnyNode{}, nil
	default:
		c.errors = append(c.errors, &typecheckError{
			reason: fmt.Sprintf("%s is not a function.", fn),
			Pos:    callNode.Pos(),
		})
		return typedAnyNode{}, nil
	}
}

// Returns typeAstNode of branch.Body
func (c *TypecheckContext) typecheckMatchBranch(branch ast.MatchBranch, sc typecheckScope) (TypedAstNode, error) {
	bodyScope := typecheckScope{
		parent: &sc,
		vars:   map[string]TypedAstNode{},
	}
	// Target might be an EnumNode, which needs to be handled
	// differently when its in a match target as it might put variables into scope
	switch t := branch.Target.(type) {
	case ast.IdentifierNode:
		bodyScope.put(t.Payload, typedAnyNode{}, t.Pos())
	case ast.ListNode:
		for _, v := range t.Elems {
			identifier, isIdentifier := v.(ast.IdentifierNode)
			if isIdentifier {
				err := bodyScope.put(identifier.Payload, typedAnyNode{}, identifier.Pos())
				if err != nil {
					return nil, err
				}
			}
		}
	case ast.EnumNode:
		for _, v := range t.Args {
			identifier, isIdentifier := v.(ast.IdentifierNode)
			if isIdentifier {
				err := bodyScope.put(identifier.Payload, typedAnyNode{}, identifier.Pos())
				if err != nil {
					return nil, err
				}
			}
		}
	default:
		_, err := c.typecheckExpr(branch.Target, bodyScope)
		if err != nil {
			return nil, err
		}
	}
	return c.typecheckExpr(branch.Body, bodyScope)
}

func (c *TypecheckContext) typecheckBinaryNode(n ast.BinaryNode, sc typecheckScope) (TypedAstNode, error) {
	leftComputed, err := c.typecheckExpr(n.Left, sc)
	if err != nil {
		return nil, err
	}
	rightComputed, err := c.typecheckExpr(n.Right, sc)
	if err != nil {
		return nil, err
	}

	// NOTE: this is just to make sure we dont acidentally say something is wrong when it isnt.
	// Should be removed eventually.
	if anyUnknowns(leftComputed, rightComputed) {
		return typedAnyNode{}, nil
	}

	switch n.Op {
	case ast.And, ast.Or:
		if !isBool(leftComputed) || !isBool(rightComputed) {
			c.errors = append(c.errors, &typecheckError{
				reason: fmt.Sprintf("%s operator only works with bool. %s and %s was used", n, leftComputed, rightComputed),
				Pos:    n.Pos(),
			})
			return typedAnyNode{}, nil
		}
		return typedBoolNode{tok: n.Tok}, nil
	case ast.Plus, ast.Divide, ast.Modulus, ast.Times:
		if !isNum(leftComputed) || !isNum(rightComputed) {
			c.errors = append(c.errors, &typecheckError{
				reason: fmt.Sprintf("%s operator only works with ints and floats. %s and %s was used",
					n.Tok, color.Str(color.Yellow, leftComputed.String()), color.Str(color.Yellow, rightComputed.String())),
				Pos: n.Pos(),
			})
			// If we find an error, we return unknown to avoid more errors.
			return typedAnyNode{}, nil
		}
		return getNumTypeFromBinOp(leftComputed, rightComputed), nil
	case ast.PlusOther:
		if !isIterator(leftComputed) || !isIterator(rightComputed) {
			c.errors = append(c.errors, &typecheckError{
				reason: fmt.Sprintf("++ operator only works with iterators (list and string). %s and %s was used",
					color.Str(color.Yellow, leftComputed.String()), color.Str(color.Yellow, rightComputed.String())),
				Pos: n.Pos(),
			})
			return typedAnyNode{}, nil
		}
		return getIteratorType(leftComputed, rightComputed), nil
	default:
		return typedAnyNode{}, nil
	}
}

// typecheckExpr is the only function that does not 'insert' typecheckError into TypecheckContext.
// This means that we can insert typeccheckError at the boundaries like `typecheckNodes` which is at the "beginnig" for parsing
// a root node, and like typecheckBinaryNode which is at "the end".
func (c *TypecheckContext) typecheckExpr(node ast.AstNode, sc typecheckScope) (TypedAstNode, error) {
	switch n := node.(type) {
	case ast.UnderscoreNode:
		return typedAnyNode{}, nil
	case ast.IntNode:
		return typedIntNode{
			tok: n.Tok,
		}, nil
	case ast.FloatNode:
		return typedFloatNode{
			tok: n.Tok,
		}, nil
	case ast.BoolNode:
		return typedBoolNode{
			tok: n.Tok,
		}, nil
	case ast.StringNode:
		return typedStringNode{
			tok: n.Tok,
		}, nil
	case ast.BinaryNode:
		return c.typecheckBinaryNode(n, sc)
	case ast.IdentifierNode:
		return sc.get(n.Payload, n.Pos())
	case ast.AssignmentNode:
		_, isFn := n.Right.(ast.FnNode)
		if isFn {
			left, isIdentifier := n.Left.(ast.IdentifierNode)
			if isIdentifier {
				sc.putCurrentFn(left.Payload)
			}
		}
		assignedNode, err := c.typecheckExpr(n.Right, sc)
		if err != nil {
			return nil, err
		}
		switch left := n.Left.(type) {
		case ast.IdentifierNode:
			err := sc.put(left.Payload, assignedNode, n.Pos())
			return assignedNode, err
		default:
			return nil, &typecheckError{
				reason: fmt.Sprintf("Invalid assignment target %s", left.String()),
				Pos:    n.Pos(),
			}
		}
	case ast.BlockNode:
		blockScope := typecheckScope{
			parent: &sc,
			vars:   map[string]TypedAstNode{},
		}

		last := len(n.Exprs) - 1
		for _, expr := range n.Exprs[:last] {
			_, err := c.typecheckExpr(expr, blockScope)
			if err != nil {
				return nil, err
			}
		}
		return c.typecheckExpr(n.Exprs[last], blockScope)
	case ast.AliasNode:
		aliasScope := typecheckScope{
			parent: &sc,
			vars:   map[string]TypedAstNode{},
		}
		targets := make([]TypedAstNode, len(n.Targets))
		for i, expr := range n.Targets {
			typed, err := c.typecheckExpr(expr, aliasScope)
			if err != nil {
				return nil, err
			}
			targets[i] = typed
		}
		typedAlias := typedAliasNode{
			name:    n.Name,
			targets: targets,
		}
		err := sc.put(n.Name, typedAlias, n.Pos())
		if err != nil {
			return nil, err
		}
		return typedAlias, nil
	case ast.FnNode:
		fnScope := typecheckScope{
			parent: &sc,
			vars:   map[string]TypedAstNode{},
		}
		args := c.toMaybeTypedArgs(n.Args, sc)
		for _, a := range args {
			switch arg := a.(type) {
			case typedArg:
				err := fnScope.put(arg.name, arg.alias, n.Pos())
				if err != nil {
					return nil, err
				}
			case untypedArg:
				err := fnScope.put(arg.name, typedAnyNode{}, n.Pos())
				if err != nil {
					return nil, err
				}
			default:
				panic("unreachable")
			}
		}

		body, err := c.typecheckExpr(n.Body, fnScope)
		if err != nil {
			// If the body is not typechecking, we want to report that, but the function "signature" might still be 'correct'
			c.errors = append(c.errors, err)
			body = typedAnyNode{}
		}
		return typedFnNode{
			args: args,
			tok:  n.Tok,
			body: body,
		}, nil
	case ast.FnCallNode:
		return c.typecheckFnCallNode(n, sc)
	case ast.EnumNode:
		args := []TypedAstNode{}
		for _, v := range n.Args {
			arg, err := c.typecheckExpr(v, sc)
			if err != nil {
				c.errors = append(c.errors, err)
				return typedAnyNode{}, nil
			}
			args = append(args, arg)
		}
		return typedEnumNode{
			parent: n.Parent,
			name:   n.Name,
			args:   args,
		}, nil
	case ast.MatchNode:
		_, err := c.typecheckExpr(n.Cond, sc)
		if err != nil {
			c.errors = append(c.errors, err)
			return typedAnyFnNode{}, nil
		}

		bodies := make([]TypedAstNode, 0)
		for _, branch := range n.Branches {
			body, err := c.typecheckMatchBranch(branch, sc)
			if err != nil {
				c.errors = append(c.errors, err)
				bodies = append(bodies, typedAnyNode{})
			} else {
				bodies = append(bodies, body)
			}
		}
		return getTypeFromMatchBodies(bodies), nil
	default:
		// TODO: remove default when we have handled everything
		// This is just a pillow
		return typedAnyNode{}, nil
	}
}

func (c *TypecheckContext) typecheckNodes(nodes []ast.AstNode) (TypedAstNode, error) {
	var returnValue TypedAstNode = nil
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

func (c *TypecheckContext) LoadLibs() error {
	base, ok := lib.Stdlibs["base"]
	if !ok {
		return fmt.Errorf("Could not load lib/base.raja")
	}
	_, err := c.Typecheck(strings.NewReader(base), "base")
	return err
}

func (c *TypecheckContext) Typecheck(reader io.Reader, filename string) (TypedAstNode, error) {
	program, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	tokenizer := ast.NewTokenizer(string(program), filename)
	tokens := tokenizer.Tokenize()
	parser := ast.NewParser(tokens)
	nodes, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	v, typecheckErr := c.typecheckNodes(nodes)
	if typecheckErr != nil {
		return nil, typecheckErr
	}
	return v, nil
}
