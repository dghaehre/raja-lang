package eval

import (
	"bytes"
	"dghaehre/raja/ast"
	"dghaehre/raja/util"
	"fmt"
	color "github.com/dghaehre/termcolor"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

type stackEntry struct {
	name string
	ast.Pos
}

func (e stackEntry) String() string {
	if e.name != "" {
		return fmt.Sprintf("  in function %s %s", e.name, e.Pos)
	}
	return fmt.Sprintf("  in anonymous function %s", e.Pos)
}

type runtimeError struct {
	reason string
	ast.Pos
	stackTrace []stackEntry
}

func (e *runtimeError) Error() string {
	trace := make([]string, len(e.stackTrace))
	for i, entry := range e.stackTrace {
		trace[i] = entry.String()
	}
	header := color.Str(color.Red, "Runtime error")
	return fmt.Sprintf("%s at %s:\n\n%s\n%s", header, e.Pos, e.reason, strings.Join(trace, "\n"))
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

func isMutable(name string) bool {
	return strings.HasPrefix(name, "mut_")
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
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
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
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
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
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
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
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
	if w, ok := u.(StringValue); ok {
		return bytes.Equal(v, w)
	}
	return false
}

type ListValue []Value

func (v *ListValue) String() string {
	stringValues := make([]string, len(*v))
	for i, s := range *v {
		stringValues[i] = s.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(stringValues, ", "))
}

func (v *ListValue) Eq(u Value) bool {
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
	if uu, ok := u.(*ListValue); ok {
		if len(*v) != len(*uu) {
			return false
		}
		for i := 0; i < len(*v); i++ {
			if !(*v)[i].Eq((*uu)[i]) {
				return false
			}
		}
		return true
	}
	return false
}

type AliasValue struct {
	targets []Value
	scope
}

func (a AliasValue) String() string {
	stringValues := make([]string, len(a.targets))
	for i, s := range a.targets {
		stringValues[i] = s.String()
	}
	return fmt.Sprintf("alias = %s", strings.Join(stringValues, " | "))
}

// This might be the function we use to determine if a given "type/alias"
// matches a value
func (a AliasValue) Eq(u Value) bool {
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
	for _, v := range a.targets {
		if v.Eq(u) {
			return true
		}
	}
	return false
}

// TODO: rename args
type EnumValue struct {
	parent string
	name   string
	args   []Value
}

func (e EnumValue) String() string {
	n := fmt.Sprintf("%s::%s", e.parent, e.name)
	if len(e.args) == 0 {
		return n
	}
	stringValues := make([]string, len(e.args))
	for i, s := range e.args {
		stringValues[i] = s.String()
	}
	return n + "(" + strings.Join(stringValues, ", ") + ")"
}

func (e EnumValue) Eq(u Value) bool {
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
	switch uu := u.(type) {
	case EnumValue:
		if e.parent != uu.parent || e.name != uu.name {
			return false
		}
		if len(e.args) != len(uu.args) {
			return false
		}
		for i := 0; i < len(e.args); i++ {
			if !e.args[i].Eq(uu.args[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Used to store FnValue's with the same name. (Multiple dispatch)
//
// Only used in scope
type FnValues struct {
	values []FnValue
}

func (v FnValues) String() string {
	stringValues := make([]string, len(v.values))
	for i, s := range v.values {
		stringValues[i] = s.String()
	}
	return strings.Join(stringValues, ", ")
}

// NOTE: NOT CURRENTLY USED
func (v FnValues) Eq(u Value) bool {
	return false
}

// Implementing sort for FnValues
// Putting the most 'specific' function at [0], regardless of how many arguments a given function have.
type MostSpecific []FnValue

func (fv MostSpecific) Len() int { return len(fv) }

func HasAlias(a ast.Arg) bool {
	return a.Alias != ""
}

// Currently only sorting by which function that has the most 'aliases'
func (fv MostSpecific) Less(i, j int) bool {
	x := len(util.Filter(fv[i].fn.Args, HasAlias))
	y := len(util.Filter(fv[j].fn.Args, HasAlias))
	return x > y
}
func (fv MostSpecific) Swap(i, j int) { fv[i], fv[j] = fv[j], fv[i] }

type FnValue struct {
	fn *ast.FnNode
	scope
}

func (v FnValue) String() string {
	return v.fn.String()
}

func (v FnValue) Eq(u Value) bool {
	if _, ok := u.(UnderscoreValue); ok {
		return true
	}
	if w, ok := u.(FnValue); ok {
		return v.fn == w.fn
	}
	return false
}

type UnderscoreValue byte

// interned "empty" value
const underscorevalue UnderscoreValue = 0

func (v UnderscoreValue) String() string {
	return "_"
}
func (v UnderscoreValue) Eq(u Value) bool {
	return true
}

// Scope

// Update variable
// TODO: how do we not know we want to "update" a variable in the outer scope?
func (sc *scope) update(name string, v Value, pos ast.Pos) *runtimeError {
	_, exist := sc.vars[name]
	if exist {
		if isMutable(name) {
			sc.vars[name] = v
			return nil
		} else {
			return &runtimeError{
				reason: fmt.Sprintf("%s is not mutable.\nTry renaming the variable to mut_%s", name, name),
				Pos:    pos,
			}
		}
	}
	if sc.parent != nil {
		return sc.parent.update(name, v, pos)
	}
	return &runtimeError{
		reason: fmt.Sprintf("Cannot find variable %s to update.\nMake sure you have already created the variable before calling update", name),
		Pos:    pos,
	}
}

// Put variable into scope
func (sc *scope) put(name string, v Value, pos ast.Pos) *runtimeError {
	switch value := v.(type) {
	case FnValue:
		scvalue, ok := sc.vars[name]
		if !ok {
			sc.vars[name] = FnValues{
				values: []FnValue{value},
			}
			return nil
		}
		switch scvalue := scvalue.(type) {
		case FnValues:
			scvalue.values = append(scvalue.values, value)
			sc.vars[name] = scvalue
			return nil
		default:
			return &runtimeError{
				reason: fmt.Sprintf("Should never happen. expected fnValue, got %s.", scvalue),
				Pos:    pos,
			}
		}
	default:
		_, exist := sc.vars[name]
		if exist {
			if isMutable(name) {
				return &runtimeError{
					reason: fmt.Sprintf("To update a variable, use the update function.\nExample: %s.update(%s)", name, v),
					Pos:    pos,
				}
			}
			return &runtimeError{
				reason: fmt.Sprintf("%s is not mutable.\nTry renaming the variable to mut_%s and use the update function\nExample: %s.update(%s)", name, name, name, name),
				Pos:    pos,
			}
		}
		sc.vars[name] = v
	}
	return nil
}

func (sc *scope) get(name string) (Value, *runtimeError) {
	if v, ok := sc.vars[name]; ok {
		return v, nil
	}
	if sc.parent != nil {
		return sc.parent.get(name)
	}
	return nil, &runtimeError{
		reason: fmt.Sprintf("%s is undefined", name),
	}
}

// Eval

func (c *Context) Eval(reader io.Reader, filename string) (Value, error) {
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
	v, runtimeErr := c.evalNodes(nodes)
	if runtimeErr != nil {
		return nil, runtimeErr
	}
	return v, nil
}

func incompatibleError(op ast.TokKind, left, right Value, position ast.Pos) *runtimeError {
	return &runtimeError{
		reason: fmt.Sprintf("Cannot %s incompatible values %s, %s",
			ast.Token{Kind: op}, left, right),
		Pos: position,
	}
}

var divisionByZeroErr = runtimeError{
	reason: fmt.Sprintf("Division by zero"),
}

func floatBinaryOp(op ast.TokKind, left FloatValue, right FloatValue) (Value, *runtimeError) {
	switch op {
	case ast.Minus:
		return FloatValue(left - right), nil
	case ast.Plus:
		return FloatValue(left + right), nil
	case ast.Divide:
		if right == 0 {
			return nil, &divisionByZeroErr
		}
		return FloatValue(left / right), nil
	case ast.Modulus:
		if right == 0 {
			return nil, &divisionByZeroErr
		}
		return FloatValue(math.Mod(float64(left), float64(right))), nil
	case ast.Times:
		return FloatValue(left * right), nil
	case ast.Eq:
		return BoolValue(left == right), nil
	case ast.Geq:
		return BoolValue(left >= right), nil
	case ast.Greater:
		return BoolValue(left > right), nil
	case ast.Less:
		return BoolValue(left < right), nil
	case ast.Leq:
		return BoolValue(left <= right), nil
	case ast.Neq:
		return BoolValue(left != right), nil
	default:
		return nil, incompatibleError(op, left, right, ast.Pos{})
	}
}

func intBinaryOp(op ast.TokKind, left IntValue, right IntValue) (Value, *runtimeError) {
	switch op {
	case ast.Minus:
		return IntValue(left - right), nil
	case ast.Plus:
		return IntValue(left + right), nil
	case ast.Times:
		return IntValue(left * right), nil
	case ast.Divide:
		if right == 0 {
			return nil, &divisionByZeroErr
		}
		return IntValue(left / right), nil
	case ast.Modulus:
		if right == 0 {
			return nil, &divisionByZeroErr
		}
		return IntValue(left % right), nil
	case ast.Greater:
		return BoolValue(left > right), nil
	case ast.Less:
		return BoolValue(left < right), nil
	case ast.Geq:
		return BoolValue(left >= right), nil
	case ast.Leq:
		return BoolValue(left <= right), nil
	case ast.Eq:
		return BoolValue(left == right), nil
	case ast.Neq:
		return BoolValue(left != right), nil
	default:
		return nil, incompatibleError(op, left, right, ast.Pos{})
	}
}

func stringBinaryOp(op ast.TokKind, left StringValue, right StringValue) (Value, *runtimeError) {
	switch op {
	case ast.PlusOther:
		x := append(left, right...)
		return StringValue(x), nil
	case ast.Eq:
		return BoolValue(string(left) == string(right)), nil
	case ast.Neq:
		return BoolValue(string(left) != string(right)), nil
	default:
		return nil, incompatibleError(op, left, right, ast.Pos{})
	}
}

func listBinaryOp(op ast.TokKind, left *ListValue, right *ListValue) (Value, *runtimeError) {
	switch op {
	case ast.PlusOther:
		x := append(*left, *right...)
		newlist := ListValue(x)
		return &newlist, nil
	default:
		return nil, incompatibleError(op, left, right, ast.Pos{})
	}
}

func (c *Context) evalBinaryNode(n ast.BinaryNode, sc scope) (Value, *runtimeError) {
	leftComputed, err := c.evalExpr(n.Left, sc)
	if err != nil {
		return nil, err
	}
	rightComputed, err := c.evalExpr(n.Right, sc)
	if err != nil {
		return nil, err
	}
	if n.Op == ast.Eq {
		return BoolValue(leftComputed.Eq(rightComputed)), nil
	}
	switch left := leftComputed.(type) {
	case *ListValue:
		right, ok := rightComputed.(*ListValue)
		if !ok {
			switch x := rightComputed.(type) {
			case IntValue, FloatValue, StringValue: // TODO: extend
				elem := make([]Value, 1)
				elem[0] = x
				l := ListValue(elem)
				right = &l
			default:
				return nil, incompatibleError(n.Op, leftComputed, rightComputed, n.Pos())
			}
		}
		val, err := listBinaryOp(n.Op, left, right)
		if err != nil {
			err.Pos = n.Pos()
		}
		return val, err

	case FloatValue:
		right, ok := rightComputed.(FloatValue)
		if !ok {
			rightFloat, ok := rightComputed.(IntValue)
			if !ok {
				return nil, incompatibleError(n.Op, leftComputed, rightComputed, n.Pos())
			}

			right := FloatValue(float64(int64(rightFloat)))
			val, err := floatBinaryOp(n.Op, left, right)
			if err != nil {
				err.Pos = n.Pos()
			}
			return val, err
		}

		val, err := floatBinaryOp(n.Op, left, right)
		if err != nil {
			err.Pos = n.Pos()
		}
		return val, err
	case IntValue:
		right, ok := rightComputed.(IntValue)
		if !ok {
			rightFloat, ok := rightComputed.(FloatValue)
			if !ok {
				return nil, incompatibleError(n.Op, leftComputed, rightComputed, n.Pos())
			}

			leftFloat := FloatValue(float64(int64(left)))
			val, err := floatBinaryOp(n.Op, leftFloat, rightFloat)
			if err != nil {
				err.Pos = n.Pos()
			}
			return val, err
		}

		val, err := intBinaryOp(n.Op, left, right)
		if err != nil {
			err.Pos = n.Pos()
		}
		return val, err
	case StringValue:
		right, ok := rightComputed.(StringValue)
		if !ok {
			return nil, incompatibleError(n.Op, leftComputed, rightComputed, n.Pos())
		}
		val, err := stringBinaryOp(n.Op, left, right)
		if err != nil {
			err.Pos = n.Pos()
		}
		return val, err
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Binary operator %s is not defined for values %s, %s",
				ast.Token{Kind: n.Op}, leftComputed, rightComputed),
			Pos: n.Pos(),
		}
	}
}

func (c *Context) getCorrectFnValue(n ast.FnCallNode, fnv FnValues, args []Value) (FnValue, *runtimeError) {

	// Filter out functions that does not 'pass' as possible alternatives
	var filterError *runtimeError
	relevant := util.Filter(fnv.values, func(f FnValue) bool {
		if len(f.fn.Args) != len(args) {
			return false
		}
		if len(args) == 0 {
			return true
		}
		for i := 0; i < len(args); i++ {
			if f.fn.Args[i].Alias == "" {
				continue
			}
			v, err := c.scope.get(f.fn.Args[i].Alias)
			if err != nil {
				filterError = err
				return false
			}
			if !v.Eq(args[i]) {
				return false
			}
		}
		return true
	})
	if filterError != nil {
		return FnValue{}, filterError
	}

	if len(relevant) == 0 {
		return FnValue{}, &runtimeError{
			reason: fmt.Sprintf("Cannot call function %s with the supplied args.\nThere are %d function(s) named %s in scope, but none matched the parameters used.", n.Fn, len(fnv.values), n.Fn),
			Pos:    n.Pos(),
		}
	}

	sort.Sort(MostSpecific(relevant))
	return relevant[0], nil
}

func (c *Context) evalFnCallNode(n ast.FnCallNode, sc scope, args []Value) (Value, *runtimeError) {
	leftComputed, err := c.evalExpr(n.Fn, sc)
	if err != nil {
		return nil, err
	}
	switch left := leftComputed.(type) {
	case BuiltinFnValue:
		return left.fn(n.FirstArgName(), args)
	case FnValues: // Multiple Dispatch
		v, err := c.getCorrectFnValue(n, left, args)
		if err != nil {
			return nil, err
		}
		fnScope := scope{
			parent: &v.scope,
			vars:   map[string]Value{},
		}
		for i, a := range v.fn.Args {
			if a.Name != "" {
				err := fnScope.put(a.Name, args[i], n.Pos())
				if err != nil {
					return nil, err
				}
			}
		}
		return c.evalExpr(v.fn.Body, fnScope)
	case FnValue:
		// Not sure if this will ever happen?
		// Stays here just in case for now..

		// Takes the scope from outside of the defined function.
		fnScope := scope{
			parent: &left.scope,
			vars:   map[string]Value{},
		}
		for i, a := range left.fn.Args {
			if a.Name != "" {
				err := fnScope.put(a.Name, args[i], n.Pos())
				if err != nil {
					return nil, err
				}
			}
		}
		return c.evalExpr(left.fn.Body, fnScope)
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Cannot call function from %s.", leftComputed),
			Pos:    n.Pos(),
		}
	}
}

func (c *Context) evalMatchNode(n ast.MatchNode, sc scope) (Value, *runtimeError) {
	cond, err := c.evalExpr(n.Cond, sc)
	if err != nil {
		return nil, err
	}
	for _, v := range n.Branches {
		t, bodyScope, err := c.evalMatchBranchExpr(v.Target, sc, cond)
		if err != nil {
			return nil, err
		}
		if cond.Eq(t) {
			return c.evalExpr(v.Body, bodyScope)
		}
	}
	return nil, &runtimeError{
		reason: fmt.Sprintf("No patterns matched in match expression: %s", n.String()),
		Pos:    n.Pos(),
	}
}

// Return a list of values from:
// - EnumValue
// - ListValue
//
// Other Values returns an empty list
func getIndexValuesFromValue(value Value, max int) []Value {
	condArgs := []Value{}
	switch v := value.(type) {
	case EnumValue:
		condArgs = v.args
	case *ListValue:
		for i, vv := range *v {
			if i > max {
				break
			}
			condArgs = append(condArgs, vv)
		}
	}
	return condArgs
}

// This is a wrapper around evalExpr to make the pattern matching with the match keyword better.
// It handles the listed nodes in a special way:
// - identifierNode
// - enumNode
// - listNode
//
// If the given node is one of these nodes, it will look for identifierNode's inside the original node.
// If it finds one, it will:
// - substitue the identifierNode with an underscoreNode
// - put that identifierNode/identifierValue into scope to be used in the body of that branch
func (c *Context) evalMatchBranchExpr(node ast.AstNode, sc scope, cond Value) (Value, scope, *runtimeError) {
	// Creating a new scope for the body of the target branch.
	bodyScope := scope{
		parent: &sc,
		vars:   map[string]Value{},
	}

	switch n := node.(type) {
	case ast.IdentifierNode:
		bodyScope.put(n.Payload, cond, n.Pos())
		return underscorevalue, bodyScope, nil
	case ast.EnumNode:
		condArgs := getIndexValuesFromValue(cond, len(n.Args))
		var err *runtimeError
		elems := make([]Value, len(n.Args))
		for i, elNode := range n.Args {
			if i >= len(condArgs) {
				// This is to prevent us from causing a panic with condArgs[i]
				break
			}
			switch en := elNode.(type) {
			case ast.IdentifierNode:
				bodyScope.put(en.Payload, condArgs[i], n.Pos())
				elems[i] = underscorevalue
			default:
				elems[i], err = c.evalExpr(elNode, sc)
				if err != nil {
					return nil, sc, err
				}
			}
		}
		return EnumValue{
			name:   n.Name,
			parent: n.Parent,
			args:   elems,
		}, bodyScope, nil
	case ast.ListNode:
		condArgs := getIndexValuesFromValue(cond, len(n.Elems))
		listValue := make(ListValue, len(n.Elems))
		for i, elNode := range n.Elems {
			if i >= len(condArgs) {
				// This is to prevent us from causing a panic with condArgs[i]
				break
			}
			switch en := elNode.(type) {
			case ast.IdentifierNode:
				bodyScope.put(en.Payload, condArgs[i], n.Pos())
				listValue[i] = underscorevalue
			default:
				v, err := c.evalExpr(elNode, sc)
				listValue[i] = v
				if err != nil {
					return nil, sc, err
				}
			}
		}
		return &listValue, bodyScope, nil
	default:
		v, err := c.evalExpr(node, sc)
		return v, sc, err
	}
}

func (c *Context) evalExpr(node ast.AstNode, sc scope) (Value, *runtimeError) {
	switch n := node.(type) {
	case ast.IntNode:
		return IntValue(n.Payload), nil
	case ast.FloatNode:
		return FloatValue(n.Payload), nil
	case ast.StringNode:
		return StringValue(n.Payload), nil
	case ast.UnderscoreNode:
		return underscorevalue, nil
	case ast.BinaryNode:
		return c.evalBinaryNode(n, sc)
	case ast.BoolNode:
		return BoolValue(n.Payload), nil
	case ast.MatchNode:
		return c.evalMatchNode(n, sc)
	case ast.IdentifierNode:
		val, err := sc.get(n.Payload)
		if err != nil {
			err.Pos = n.Pos()
		}
		return val, err
	case ast.AssignmentNode:
		assignedValue, err := c.evalExpr(n.Right, sc)
		if err != nil {
			return nil, err
		}
		switch left := n.Left.(type) {
		case ast.IdentifierNode:
			err := sc.put(left.Payload, assignedValue, n.Pos())
			return assignedValue, err
		default:
			return nil, &runtimeError{
				reason: fmt.Sprintf("Invalid assignment target %s", left.String()),
				Pos:    n.Pos(),
			}
		}
	case ast.FnCallNode:
		args := make([]Value, 0, len(n.Args))
		for _, a := range n.Args {
			v, err := c.evalExpr(a, sc)
			if err != nil {
				return nil, err
			}
			args = append(args, v)
		}
		return c.evalFnCallNode(n, sc, args)
	case ast.BlockNode:
		blockScope := scope{
			parent: &sc,
			vars:   map[string]Value{},
		}
		last := len(n.Exprs) - 1
		for _, expr := range n.Exprs[:last] {
			_, err := c.evalExpr(expr, blockScope)
			if err != nil {
				return nil, err
			}
		}
		return c.evalExpr(n.Exprs[last], blockScope)
	case ast.ListNode:
		var err *runtimeError
		elems := make([]Value, len(n.Elems))
		for i, elNode := range n.Elems {
			elems[i], err = c.evalExpr(elNode, sc)
			if err != nil {
				return nil, err
			}
		}
		list := ListValue(elems)
		return &list, nil
	case ast.EnumNode:
		var err *runtimeError
		elems := make([]Value, len(n.Args))
		for i, elNode := range n.Args {
			elems[i], err = c.evalExpr(elNode, sc)
			if err != nil {
				return nil, err
			}
		}
		return EnumValue{
			name:   n.Name,
			parent: n.Parent,
			args:   elems,
		}, nil

	case ast.AliasNode:
		var err *runtimeError
		elems := make([]Value, len(n.Targets))
		for i, elNode := range n.Targets {
			elems[i], err = c.evalExpr(elNode, sc)
			if err != nil {
				return nil, err
			}
		}
		alias := AliasValue{
			targets: elems,
			scope:   sc,
		}
		err = sc.put(n.Name, alias, n.Pos())
		return alias, err
	case ast.FnNode:
		return FnValue{
			fn:    &n,
			scope: sc,
		}, nil
	}
	panic(fmt.Sprintf("Unexpected astNode type: %s", node))
}

func (c *Context) evalNodes(nodes []ast.AstNode) (Value, *runtimeError) {
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
