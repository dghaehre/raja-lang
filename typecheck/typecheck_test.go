package typecheck

import (
	"dghaehre/raja/lib"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func expectTypecheckToReturn(t *testing.T, program string, expected typedAstNode) {
	ctx := NewTypecheckContext()
	ctx.LoadBuiltins()
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
	ctx.LoadBuiltins()
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

func TestBaseLib(t *testing.T) {
	ctx := NewTypecheckContext()
	ctx.LoadBuiltins()
	base, ok := lib.Stdlibs["base"]
	if !ok {
		t.Errorf("Could not load lib/base.raja")
	}
	_, err := ctx.Typecheck(strings.NewReader(base), "test")
	if err != nil {
		err = errors.Join(err, errors.New("Did not expect base.raja to find type errors"))
		t.Error(err)
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

func TestSimpleGenericFunctionTypecheck(t *testing.T) {
	p := `
do_something = (a) => __string(a)
do_something("hey")`
	expectTypecheckToReturn(t, p, typedStringNode{})
}

// TODO: should this really be Int?
func TestSimpleFunctionTypecheck(t *testing.T) {
	p := `
add_one = (i:Int) => i + 1
add_one(1)`
	expectTypecheckToReturn(t, p, typedFloatNode{})
}

func TestSimpleFunctionErrorTypecheck(t *testing.T) {
	p := `
add_one = (i:Int) => i + 1
add_one(1, 2)`
	expectTypecheckToError(t, p, []error{paramMismatchError{}})
}

func TestAliasTypecheck(t *testing.T) {
	p := `
alias Bool = true | false
def = (a:Bool) => false
def(true)
`
	expectTypecheckToReturn(t, p, typedBoolNode{})
}

func TestAliasErrorTypecheck(t *testing.T) {
	p := `
alias Bool = true | false
def = (a:Bool) => false
def("hey")
`
	expectTypecheckToError(t, p, []error{paramMismatchError{}})
}

func TestAliasIteratorTypecheck(t *testing.T) {
	p := `
alias Iterator = Str | List
def = (a:Iterator) => false
def("hey")
`
	expectTypecheckToReturn(t, p, typedBoolNode{})
}

func TestAliasIteratorTypecheckError(t *testing.T) {
	p := `
alias Iterator = Str | List
def = (a:Iterator) => false
def(1)
`
	expectTypecheckToError(t, p, []error{paramMismatchError{}})
}

func TestReadFileTypecheck(t *testing.T) {
	p := `
read_file = (a:Str) => __read_file(a)
read_file("hello.txt")
`
	expectTypecheckToReturn(t, p, typedStringNode{})
}

func TestFoldIndexTypecheck(t *testing.T) {
	p := `
alias Iterator = List | Str

alias Maybe =
		Maybe::Some(_)
	| Maybe::None

get = (a:Iterator, b:Int) => __index(a, b, false)

fold_index = (iter:Iterator, acc, f:Fn, i:Int) => match iter.get(i) {
	Maybe::Some(a) -> iter.fold_index(f(acc, a, i), f, i + 1)
		_						 -> acc
}
add_one = (acc:Iterator, a:Int, i:Int) => acc ++ [a + 1]

fold_index([1, 2, 3], 0, add_one, 0)
`
	expectTypecheckToReturn(t, p, typedIntNode{nil})
}
