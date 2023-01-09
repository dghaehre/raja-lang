package main

import (
	"fmt"
	"os"
)

type builtinFn func([]Value) (Value, *runtimeError)

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

func (c *Context) LoadBuiltins() {
	c.LoadFunc("print", c.rajaPrint)
	c.LoadFunc("string", c.rajaString)
	c.LoadFunc("get_args", c.rajaGetArgs)

	_, err := c.LoadLib("base")
	if err != nil {
		panic(err)
	}
}

func (c *Context) LoadFunc(name string, fn builtinFn) {
	c.scope.put(name, BuiltinFnValue{
		name: name,
		fn:   fn,
	}, pos{})
}

func (c *Context) requireArgLen(fnName string, args []Value, count int) *runtimeError {
	if len(args) < count {
		return &runtimeError{
			reason: fmt.Sprintf("%s requires %d arguments, got %d", fnName, count, len(args)),
		}
	}
	return nil
}

// Builtin functions

func (c *Context) rajaString(args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("string", args, 1); err != nil {
		return nil, err
	}
	switch arg := args[0].(type) {
	case *StringValue:
		return arg, nil
	default:
		return StringValue(arg.String()), nil
	}
}

func (c *Context) rajaPrint(args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("print", args, 1); err != nil {
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

func (c *Context) rajaGetArgs(_ []Value) (Value, *runtimeError) {
	goArgs := os.Args
	args := make(ListValue, len(goArgs))
	for i, arg := range goArgs {
		args[i] = StringValue(arg)
	}
	return &args, nil
}
