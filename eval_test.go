package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func expectProgramToReturn(t *testing.T, program string, expected Value) {
	ctx := NewContext()
	ctx.LoadBuiltins()
	val, err := ctx.Eval(strings.NewReader(program))
	if err != nil {
		t.Errorf("Did not expect program to exit with error: %s", err.Error())
	}
	if val == nil {
		t.Errorf("Return value of program should not be nil")
	} else if !val.Eq(expected) {
		t.Errorf(fmt.Sprintf("Expected and returned values don't match: %s != %s",
			strconv.Quote(expected.String()),
			strconv.Quote(val.String())))
	}
}

func TestVariablesAndAddition(t *testing.T) {
	p := `
  test = 10
  x = 10
  test + x`
	expectProgramToReturn(t, p, IntValue(20))
}

func TestComments(t *testing.T) {
	p := `
  # this is a comment
  test = 10
  # some other comment
  `
	expectProgramToReturn(t, p, IntValue(10))
}

func TestVariablesAndSubtraction(t *testing.T) {
	p := `
  test = 10
  x = 10
  test - x`
	expectProgramToReturn(t, p, IntValue(0))
}

func TestHelloWorld(t *testing.T) {
	p := `
  h = "hello "
  w = "world!" 
  h ++ w`
	expectProgramToReturn(t, p, StringValue([]byte("hello world!")))
}

func TestPrint(t *testing.T) {
	p := `
  hello = "hello world!" 
  print(hello)
  `
	expectProgramToReturn(t, p, IntValue(12))
}

func TestFunctions(t *testing.T) {
	p := `
multiline_func = (x, f) => {
  y = x
  f(y, 1)
}
add = (a, b) => a + b
add_one = (x) => multiline_func(x, add)
add_one(1)
  `
	expectProgramToReturn(t, p, IntValue(2))
}

func TestList(t *testing.T) {
	p := `
  list = [1, 2, "3"]
  `
	expectProgramToReturn(t, p, &ListValue{IntValue(1), IntValue(2), StringValue("3")})
}
