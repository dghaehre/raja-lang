package main

import (
	"bytes"
	"fmt"
	"io"
	"sort"
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

// Currently only sorting by which function that has the most 'aliases'
func (fv MostSpecific) Less(i, j int) bool {
	x := len(Filter(fv[i].fn.args, HasAlias))
	y := len(Filter(fv[j].fn.args, HasAlias))
	return x > y
}
func (fv MostSpecific) Swap(i, j int) { fv[i], fv[j] = fv[j], fv[i] }

type FnValue struct {
	fn *fnNode
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

// Put variable into scope
func (sc *scope) put(name string, v Value, pos pos) *runtimeError {
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
				pos:    pos,
			}
		}
	default:
		if isMutable(name) {
			sc.vars[name] = v
			return nil
		}
		_, exist := sc.vars[name]
		if exist {
			return &runtimeError{
				reason: fmt.Sprintf("%s is not mutable.\nTry renaming the variable to mut_%s", name, name),
				pos:    pos,
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

func floatBinaryOp(op tokKind, left FloatValue, right FloatValue) (Value, *runtimeError) {
	switch op {
	case minus:
		return FloatValue(left - right), nil
	case plus:
		return FloatValue(left + right), nil
	case times:
		return FloatValue(left * right), nil
	case eq:
		return BoolValue(left == right), nil
	case neq:
		return BoolValue(left != right), nil
	default:
		return nil, incompatibleError(op, left, right, pos{})
	}
}

func intBinaryOp(op tokKind, left IntValue, right IntValue) (Value, *runtimeError) {
	switch op {
	case minus:
		return IntValue(left - right), nil
	case plus:
		return IntValue(left + right), nil
	case times:
		return IntValue(left * right), nil
	case eq:
		return BoolValue(left == right), nil
	case neq:
		return BoolValue(left != right), nil
	default:
		return nil, incompatibleError(op, left, right, pos{})
	}
}

func stringBinaryOp(op tokKind, left StringValue, right StringValue) (Value, *runtimeError) {
	switch op {
	case plusOther:
		x := append(left, right...)
		return StringValue(x), nil
	case eq:
		return BoolValue(string(left) == string(right)), nil
	case neq:
		return BoolValue(string(left) != string(right)), nil
	default:
		return nil, incompatibleError(op, left, right, pos{})
	}
}

func listBinaryOp(op tokKind, left *ListValue, right *ListValue) (Value, *runtimeError) {
	switch op {
	case plusOther:
		x := append(*left, *right...)
		newlist := ListValue(x)
		return &newlist, nil
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
				return nil, incompatibleError(n.op, leftComputed, rightComputed, n.pos())
			}
		}
		val, err := listBinaryOp(n.op, left, right)
		if err != nil {
			err.pos = n.pos()
		}
		return val, err

	case FloatValue:
		right, ok := rightComputed.(FloatValue)
		if !ok {
			rightFloat, ok := rightComputed.(IntValue)
			if !ok {
				return nil, incompatibleError(n.op, leftComputed, rightComputed, n.pos())
			}

			right := FloatValue(float64(int64(rightFloat)))
			val, err := floatBinaryOp(n.op, left, right)
			if err != nil {
				err.pos = n.pos()
			}
			return val, err
		}

		val, err := floatBinaryOp(n.op, left, right)
		if err != nil {
			err.pos = n.pos()
		}
		return val, err
	case IntValue:
		right, ok := rightComputed.(IntValue)
		if !ok {
			rightFloat, ok := rightComputed.(FloatValue)
			if !ok {
				return nil, incompatibleError(n.op, leftComputed, rightComputed, n.pos())
			}

			leftFloat := FloatValue(float64(int64(left)))
			val, err := floatBinaryOp(n.op, leftFloat, rightFloat)
			if err != nil {
				err.pos = n.pos()
			}
			return val, err
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

func (c *Context) getCorrectFnValue(n fnCallNode, fnv FnValues, args []Value) (FnValue, *runtimeError) {

	// Filter out functions that does not 'pass' as possible alternatives
	var filterError *runtimeError
	relevant := Filter(fnv.values, func(f FnValue) bool {
		if len(f.fn.args) != len(args) {
			return false
		}
		if len(args) == 0 {
			return true
		}
		for i := 0; i < len(args); i++ {
			if f.fn.args[i].alias == "" {
				continue
			}
			v, err := c.scope.get(f.fn.args[i].alias)
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
			reason: fmt.Sprintf("Cannot call function %s with the supplied args.\nThere are %d function(s) named %s in scope, but none matched the parameters used.", n.fn, len(fnv.values), n.fn),
			pos:    n.pos(),
		}
	}

	sort.Sort(MostSpecific(relevant))
	return relevant[0], nil
}

func (c *Context) evalFnCallNode(n fnCallNode, sc scope, args []Value) (Value, *runtimeError) {
	leftComputed, err := c.evalExpr(n.fn, sc)
	if err != nil {
		return nil, err
	}
	switch left := leftComputed.(type) {
	case BuiltinFnValue:
		return left.fn(args)
	case FnValues: // Multiple Dispatch
		v, err := c.getCorrectFnValue(n, left, args)
		if err != nil {
			return nil, err
		}
		fnScope := scope{
			parent: &v.scope,
			vars:   map[string]Value{},
		}
		for i, a := range v.fn.args {
			if a.name != "" {
				err := fnScope.put(a.name, args[i], n.pos())
				if err != nil {
					return nil, err
				}
			}
		}
		return c.evalExpr(v.fn.body, fnScope)
	case FnValue:
		// Not sure if this will ever happen?
		// Stays here just in case for now..

		// Takes the scope from outside of the defined function.
		fnScope := scope{
			parent: &left.scope,
			vars:   map[string]Value{},
		}
		for i, a := range left.fn.args {
			if a.name != "" {
				err := fnScope.put(a.name, args[i], n.pos())
				if err != nil {
					return nil, err
				}
			}
		}
		return c.evalExpr(left.fn.body, fnScope)
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Cannot call function from %s.", leftComputed),
			pos:    n.pos(),
		}
	}
}

func (c *Context) evalMatchNode(n matchNode, sc scope) (Value, *runtimeError) {
	cond, err := c.evalExpr(n.cond, sc)
	if err != nil {
		return nil, err
	}
	for _, v := range n.branches {
		t, err := c.evalExpr(v.target, sc)
		if err != nil {
			return nil, err
		}
		if cond.Eq(t) {
			return c.evalExpr(v.body, sc)
		}
	}
	return nil, &runtimeError{
		reason: fmt.Sprintf("No patterns matched in match expression: %s", n.String()),
		pos:    n.pos(),
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
	case underscoreNode:
		return underscorevalue, nil
	case binaryNode:
		return c.evalBinaryNode(n, sc)
	case boolNode:
		return BoolValue(n.payload), nil
	case matchNode:
		return c.evalMatchNode(n, sc)
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
			err := sc.put(left.payload, assignedValue, n.pos())
			return assignedValue, err
		default:
			return nil, &runtimeError{
				reason: fmt.Sprintf("Invalid assignment target %s", left.String()),
				pos:    n.pos(),
			}
		}
	case fnCallNode:
		args := make([]Value, 0, len(n.args))
		for _, a := range n.args {
			v, err := c.evalExpr(a, sc)
			if err != nil {
				return nil, err
			}
			args = append(args, v)
		}
		return c.evalFnCallNode(n, sc, args)
	case blockNode:
		blockScope := scope{
			parent: &sc,
			vars:   map[string]Value{},
		}

		last := len(n.exprs) - 1
		for _, expr := range n.exprs[:last] {
			_, err := c.evalExpr(expr, blockScope)
			if err != nil {
				return nil, err
			}
		}
		return c.evalExpr(n.exprs[last], blockScope)
	case listNode:
		var err *runtimeError
		elems := make([]Value, len(n.elems))
		for i, elNode := range n.elems {
			elems[i], err = c.evalExpr(elNode, sc)
			if err != nil {
				return nil, err
			}
		}
		list := ListValue(elems)
		return &list, nil
	case enumNode:
		var err *runtimeError
		elems := make([]Value, len(n.args))
		for i, elNode := range n.args {
			elems[i], err = c.evalExpr(elNode, sc)
			if err != nil {
				return nil, err
			}
		}
		return EnumValue{
			name:   n.name,
			parent: n.parent,
			args:   elems,
		}, nil

	case aliasNode:
		var err *runtimeError
		elems := make([]Value, len(n.targets))
		for i, elNode := range n.targets {
			elems[i], err = c.evalExpr(elNode, sc)
			if err != nil {
				return nil, err
			}
		}
		alias := AliasValue{
			targets: elems,
			scope:   sc,
		}
		err = sc.put(n.name, alias, n.pos())
		return alias, err
	case fnNode:
		return FnValue{
			fn:    &n,
			scope: sc,
		}, nil
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
