package eval

import (
	"dghaehre/raja/ast"
	"fmt"
	"os"
	"strconv"
)

type builtinFn func(string, []Value) (Value, *runtimeError)

type BuiltinFnValue struct {
	name string
	fn   builtinFn
}

func (v BuiltinFnValue) String() string {
	return fmt.Sprintf("<native function %s>", v.name)
}
func (v BuiltinFnValue) Eq(u Value) bool {
	if w, ok := u.(BuiltinFnValue); ok {
		return v.name == w.name
	}
	return false
}

type aliasFn = func(u Value) bool

type BuiltinAliasValue struct {
	name string
	eqFn aliasFn
}

func (v BuiltinAliasValue) String() string {
	return "alias = " + v.name
}

func (v BuiltinAliasValue) Eq(u Value) bool {
	return v.eqFn(u)
}

func (c *Context) LoadBuiltins() {
	c.LoadFunc("__print", c.rajaPrint)
	c.LoadFunc("__index", c.rajaIndex)
	c.LoadFunc("__string", c.rajaString)
	c.LoadFunc("__int", c.rajaInt)
	c.LoadFunc("__args", c.rajaArgs)
	c.LoadFunc("__exit", c.rajaExit)
	c.LoadFunc("__read_file", c.rajaReadFile)
	c.LoadFunc("update", c.rajaUpdate)

	// Types/Alias
	c.LoadAlias("Int", c.rajaAliasInt)
	c.LoadAlias("Float", c.rajaAliasFloat)
	c.LoadAlias("Str", c.rajaAliasStr)
	c.LoadAlias("List", c.rajaAliasList)
	c.LoadAlias("Fn", c.rajaAliasFn)
	c.LoadAlias("Enum", c.rajaAliasEnum)
	// TODO: Bool

	_, err := c.LoadLib("base")
	if err != nil {
		panic(err)
	}
}

func (c *Context) LoadFunc(name string, fn builtinFn) {
	c.scope.put(name, BuiltinFnValue{
		name: name,
		fn:   fn,
	}, ast.Pos{})
}

func (c *Context) LoadAlias(name string, fn aliasFn) {
	c.scope.put(name, BuiltinAliasValue{
		name: name,
		eqFn: fn,
	}, ast.Pos{})
}

func (c *Context) requireArgLen(fnName string, args []Value, count int) *runtimeError {
	if len(args) < count {
		return &runtimeError{
			reason: fmt.Sprintf("%s requires %d arguments, got %d", fnName, count, len(args)),
		}
	}
	return nil
}

func toSome(v Value) Value {
	return EnumValue{
		parent: "Maybe",
		name:   "Some",
		args:   []Value{v},
	}
}

func toNone() Value {
	return EnumValue{
		parent: "Maybe",
		name:   "None",
		args:   []Value{},
	}
}

func toOk(v Value) Value {
	return EnumValue{
		parent: "Result",
		name:   "Ok",
		args:   []Value{v},
	}
}

func toErr(v Value) Value {
	return EnumValue{
		parent: "Result",
		name:   "Err",
		args:   []Value{v},
	}
}

// Builtin functions

func (c *Context) rajaString(_ string, args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__string", args, 1); err != nil {
		return nil, err
	}
	switch arg := args[0].(type) {
	case *StringValue:
		return arg, nil
	default:
		return StringValue(arg.String()), nil
	}
}

func (c *Context) rajaInt(_ string, args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__int", args, 1); err != nil {
		return nil, err
	}
	switch arg := args[0].(type) {
	case StringValue:
		i, err := strconv.Atoi(string(arg))
		if err != nil {
			return toErr(StringValue(err.Error())), nil
		}
		return toOk(IntValue(i)), nil
	default:
		return toErr(StringValue(fmt.Sprintf("Cannot cast %s to int", arg))), nil
	}
}

func (c *Context) rajaPrint(_ string, args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__print", args, 1); err != nil {
		return nil, err
	}

	outputString, ok := args[0].(StringValue)
	if !ok {
		return nil, &runtimeError{
			reason: fmt.Sprintf("Unexpected argument to print: %s", args[0]),
		}
	}

	n, _ := os.Stdout.Write(outputString)
	return IntValue(n), nil
}

func (c *Context) rajaReadFile(_ string, args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__read_file", args, 1); err != nil {
		return nil, err
	}

	s, ok := args[0].(StringValue)
	if !ok {
		return nil, &runtimeError{
			reason: fmt.Sprintf("Unexpected argument to print: %s", args[0]),
		}
	}

	bs, err := os.ReadFile(string(s))
	if err != nil {
		return nil, &runtimeError{
			reason: fmt.Sprintf("Could not read file: %s. Error: %s", s, err),
		}
	}
	return StringValue(string(bs)), nil
}

func (c *Context) rajaArgs(_ string, _ []Value) (Value, *runtimeError) {
	goArgs := os.Args
	args := make(ListValue, len(goArgs))
	for i, arg := range goArgs {
		args[i] = StringValue(arg)
	}
	return &args, nil
}

func (c *Context) rajaExit(_ string, args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__exit", args, 1); err != nil {
		return nil, err
	}

	switch arg := args[0].(type) {
	case IntValue:
		os.Exit(int(arg))
		// unreachable
		return IntValue(int(arg)), nil
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Mismatched types in call exit(%s)", args[0]),
		}
	}
}

func (c *Context) rajaUpdate(name string, args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__exit", args, 2); err != nil {
		return nil, err
	}
	err := c.scope.update(name, args[1], ast.Pos{})
	return args[1], err
}

// Supports:
// - List
// - Str
//
// which is: Iterator
//
// Returns a Maybe if third argument is false
func (c *Context) rajaIndex(_ string, args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__index", args, 3); err != nil {
		return nil, err
	}
	var unsafe bool
	switch u := args[2].(type) {
	case BoolValue:
		unsafe = bool(u)
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Unexpected argument to __index: %s. Expected a bool as the third argument.", args[2]),
		}
	}

	switch v := args[0].(type) {
	case *ListValue:
		switch i := args[1].(type) {
		case IntValue:
			l := *v
			if unsafe {
				return l[i], nil
			}
			if len(l) > int(i) {
				return toSome(l[i]), nil
			}
			return toNone(), nil
		default:
			return nil, &runtimeError{
				reason: fmt.Sprintf("Unexpected argument to __index: %s. Expected an int as index.", args[1]),
			}
		}
	case EnumValue:
		switch i := args[1].(type) {
		case IntValue:
			if unsafe {
				return v.args[i], nil
			}
			if len(v.args) > int(i) {
				return toSome(v.args[i]), nil
			}
			return toNone(), nil
		default:
			return nil, &runtimeError{
				reason: fmt.Sprintf("Unexpected argument to __index: %s. Expected an int as index.", args[1]),
			}
		}
	case StringValue:
		switch i := args[1].(type) {
		case IntValue:
			if unsafe {
				return StringValue([]byte{v[i]}), nil
			}
			if len(v) > int(i) {
				return toSome(StringValue([]byte{v[i]})), nil
			}
			return toNone(), nil
		default:
			return nil, &runtimeError{
				reason: fmt.Sprintf("Unexpected argument to __index: %s. Expected an int as index.", args[1]),
			}
		}
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Unexpected argument to __index: %s. Expected an Iterator.", args[0]),
		}
	}
}

func (c *Context) rajaAliasInt(u Value) bool {
	switch u.(type) {
	case IntValue:
		return true
	default:
		return false
	}
}

func (c *Context) rajaAliasFloat(u Value) bool {
	switch u.(type) {
	case FloatValue:
		return true
	default:
		return false
	}
}

func (c *Context) rajaAliasStr(u Value) bool {
	switch u.(type) {
	case StringValue:
		return true
	default:
		return false
	}
}

func (c *Context) rajaAliasList(u Value) bool {
	switch u.(type) {
	case *ListValue:
		return true
	default:
		return false
	}
}

func (c *Context) rajaAliasEnum(u Value) bool {
	switch u.(type) {
	case EnumValue:
		return true
	default:
		return false
	}
}

func (c *Context) rajaAliasFn(u Value) bool {
	switch u.(type) {
	case FnValue, FnValues, BuiltinFnValue:
		return true
	default:
		return false
	}
}
