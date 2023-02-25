package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func expectTypecheckToReturn(t *testing.T, program string, expected typedAstNode) {
	ctx := NewTypecheckContext()
	val, err := ctx.Typecheck(strings.NewReader(program), "test")
	if err != nil {
		t.Errorf("Did not expect program to typecheck with error: \n%s", err.Error())
	}
	if val == nil {
		t.Errorf("Return value of program should not be nil")
		return
	}
	if expected.String() != val.String() {
		t.Errorf(fmt.Sprintf("Expected and returned values don't match: %s != %s",
			strconv.Quote(expected.String()),
			strconv.Quote(val.String())))
	}
}

func expectTypecheckToError(t *testing.T, program string, expected []error) {
	ctx := NewTypecheckContext()
	_, err := ctx.Typecheck(strings.NewReader(program), "test")
	if err == nil {
		t.Errorf("Did not expect program to typecheck with no error")
		return
	}
	multiErrors, ok := err.(multipleErrors)
	if !ok {
		t.Errorf("Not a multiple error")
		t.Log(multiErrors)
		t.Log(err)
		return
	}
	if len(expected) != len(multiErrors.errors) {
		// TODO: Check that the errors are the same
		t.Errorf("Expected %d errors, got %d\n\n%s", len(expected), len(multiErrors.errors), multiErrors)
	}
}

func TestSimpleAdditionTypecheck(t *testing.T) {
	p := `
a = 1
b = 2
a + b`
	expectTypecheckToReturn(t, p, typedIntNode{nil})
}

func TestSimpleAdditionFloatTypecheck(t *testing.T) {
	p := `
a = 1
b = 2.1
a + b`
	expectTypecheckToReturn(t, p, typedFloatNode{nil})
}

func TestSimpleFunctionTypecheck(t *testing.T) {
	p := `
add_one = (i:Int) => i + 1
add_one(1)`
	expectTypecheckToReturn(t, p, typedFnNode{
		args: []typedAstNode{typedArg{name: "i", alias: typedIntNode{}}},
	})
}

func TestSimpleFunctionErrorTypecheck(t *testing.T) {
	p := `
add_one = (i:Int) => i + 1
add_one(1, 2)`
	expectTypecheckToError(t, p, []error{paramMismatchError{}})
}
