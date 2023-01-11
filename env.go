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
	c.LoadFunc("__print", c.rajaPrint)
	c.LoadFunc("__index", c.rajaIndex)
	c.LoadFunc("__string", c.rajaString)
	c.LoadFunc("__args", c.rajaArgs)
	c.LoadFunc("__panic", c.rajaPanic)

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

func (c *Context) rajaPrint(args []Value) (Value, *runtimeError) {
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

func (c *Context) rajaArgs(_ []Value) (Value, *runtimeError) {
	goArgs := os.Args
	args := make(ListValue, len(goArgs))
	for i, arg := range goArgs {
		args[i] = StringValue(arg)
	}
	return &args, nil
}

// TODO: add stacktrace
func (c *Context) rajaPanic(args []Value) (Value, *runtimeError) {
	if err := c.requireArgLen("__index", args, 1); err != nil {
		return nil, err
	}
	return nil, &runtimeError{
		reason: fmt.Sprintf("Panic: %s.", args[0]),
	}
}

// Currently only supports list with int as index
// Returns a Maybe if third argument is false
func (c *Context) rajaIndex(args []Value) (Value, *runtimeError) {
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

	switch list := args[0].(type) {
	case *ListValue:
		switch i := args[1].(type) {
		case IntValue:
			l := *list
			if unsafe {
				return l[i], nil
			}
			if len(l) > int(i) {
				res := make(ListValue, 2)
				res[0] = StringValue("some")
				res[1] = l[i]
				return &res, nil
			}
			res := make(ListValue, 1)
			res[0] = StringValue("none")
			return &res, nil
		default:
			return nil, &runtimeError{
				reason: fmt.Sprintf("Unexpected argument to __index: %s. Expected an int as index.", args[1]),
			}
		}
	default:
		return nil, &runtimeError{
			reason: fmt.Sprintf("Unexpected argument to __index: %s. Expected a list.", args[0]),
		}
	}
}
