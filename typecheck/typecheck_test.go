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
	t.SkipNow()
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
	expectTypecheckToReturn(t, p, typedIntNode{})
}

func TestSimpleAdditionFloatTypecheck(t *testing.T) {
	p := `
a = 1
b = 2.1
a + b`
	expectTypecheckToReturn(t, p, typedFloatNode{})
}

func TestGetNumTypeFromBinOp(t *testing.T) {
	int := typedIntNode{}
	float := typedFloatNode{}

	res := getNumTypeFromBinOp(int, float)
	_, ok := res.(typedFloatNode)
	if !ok {
		t.Errorf("Int + Float should be Float, got %T", res)
	}

	res = getNumTypeFromBinOp(float, int)
	_, ok = res.(typedFloatNode)
	if !ok {
		t.Errorf("Float + Int should be Float, got %T", res)
	}

	res = getNumTypeFromBinOp(int, floatAlias)
	_, ok = res.(typedFloatNode)
	if !isAliasWithName(res, "Float") {
		t.Errorf("Int + Float(alias) should be Float(alias), got %+v with type %T", res, res)
	}

	res = getNumTypeFromBinOp(intAlias, float)
	_, ok = res.(typedFloatNode)
	if !isAliasWithName(res, "Float") {
		t.Errorf("Int(alias) + Float should be Float(alias), got %+v with type %T", res, res)
	}

	res = getNumTypeFromBinOp(int, int)
	_, ok = res.(typedIntNode)
	if !ok {
		t.Errorf("Int + Int should be Int, got %T", res)
	}

	res = getNumTypeFromBinOp(float, float)
	_, ok = res.(typedFloatNode)
	if !ok {
		t.Errorf("Float + Float should be Float, got %T", res)
	}

	res = getNumTypeFromBinOp(int, numAlias)
	if !isAliasWithName(res, "Num") {
		t.Errorf("Int + Num should be Num, got %+v with type %T", res, res)
	}
}

func TestRecursionTypecheck(t *testing.T) {
	t.SkipNow()
	p := `
rec_func = (a:Int) => {
  b = a + 1
  match b {
    10 -> 10
    _  -> rec_func(a + 1)
  }
}
rec_func(0)
`
	expectTypecheckToReturn(t, p, typedIntNode{})
}

func TestMatchTypecheck(t *testing.T) {
	p := `
one = 1.2

match one {
  2 -> "hey"
  a -> a
}
`
	// We might should work for this as a return type instead of Any:
	// stringOrInt := typedAliasNode{
	// 	targets: []typedAstNode{
	// 		typedStringNode{},
	// 		typedIntNode{},
	// 	}}

	expectTypecheckToReturn(t, p, typedAnyNode{})

	pMaybe := `
alias Maybe =
	Maybe::Some(_)
| Maybe::None

m = Maybe::Some(1)

match m {
  Maybe::Some(a) -> a
  _              -> 1
}
	`
	expectTypecheckToReturn(t, pMaybe, typedAnyNode{})
}

func TestIntAndFloatsTypecheck(t *testing.T) {
	pNum := `
alias Num = Float | Int
one = 1
add = (a:Num, b:Num) => a + b
one.add(1).add(1)
`
	expectTypecheckToReturn(t, pNum, numAlias)

	pInt := `
	one = 1
	add = (a:Int, b:Int) => a + b
	one.add(1)
	`
	expectTypecheckToReturn(t, pInt, typedIntNode{})

	pFloat := `
	one = 1
	add = (a:Int, b:Float) => a + b
	one.add(1.3)
	`
	expectTypecheckToReturn(t, pFloat, typedFloatNode{})
}

func TestSimpleGenericFunctionTypecheck(t *testing.T) {
	p := `
do_something = (a) => __string(a)
do_something("hey")`
	expectTypecheckToReturn(t, p, typedStringNode{})
}

func TestSimpleFunctionTypecheck(t *testing.T) {
	p := `
add_one = (i:Int) => i + 1
add_one(1)`
	expectTypecheckToReturn(t, p, typedIntNode{})
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

fold_index = (iter:Iterator, acc, f:Fn, i:Int) => acc

add_one = (acc:Iterator, a:Int, i:Int) => acc ++ [a + 1]

fold_index([1, 2, 3], 0, add_one, 0)
`
	expectTypecheckToReturn(t, p, typedAnyNode{})
}

func TestUnwrapTypecheck(t *testing.T) {
	p := `
alias Result =
		Result::Ok(_)
	| Result::Err(_)

to_ok = (a) => Result::Ok(a)

unwrap = (r:Result) => match r {
	Result::Ok(a) -> a
	Result::Err(_) -> 10
}

to_ok(1).unwrap()
`
	expectTypecheckToReturn(t, p, typedAnyNode{})

}
