package eval

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func expectProgramToReturn(t *testing.T, program string, expected Value) {
	ctx := NewContext()
	ctx.LoadBuiltins()
	val, err := ctx.Eval(strings.NewReader(program), "test")
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

func expectProgramToFail(t *testing.T, program string) {
	ctx := NewContext()
	ctx.LoadBuiltins()
	val, err := ctx.Eval(strings.NewReader(program), "test")
	if err == nil {
		t.Errorf("Did expect program to exit with error, but returned: %s", strconv.Quote(val.String()))
	}
}

func TestVariablesAndAddition(t *testing.T) {
	p := `
  test = 10
  test + 10`
	expectProgramToReturn(t, p, IntValue(20))
}

func TestBinaryFloat(t *testing.T) {
	p := `
  test = 10.0
  x = 10.0
  test * x`
	expectProgramToReturn(t, p, FloatValue(100))
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

func TestOrderOfOperations(t *testing.T) {
	p := `
  res = 1 + 2 * 3
  `
	expectProgramToReturn(t, p, IntValue(9))
}

func TestList(t *testing.T) {
	p := `
  list = [1, 2, "3"]
  `
	expectProgramToReturn(t, p, &ListValue{IntValue(1), IntValue(2), StringValue("3")})
}

func TestBinaryDot(t *testing.T) {
	p := `
add = (a, b) => a + b

make_pretty = (s) => "The answer is: " ++ s

one = 1

one
  .add(1)
  .add(1)
  .string()
  .make_pretty()
  `
	expectProgramToReturn(t, p, StringValue("The answer is: 3"))
}

func TestParens(t *testing.T) {
	p := `
  res = 1 + (1 * 3)
  `
	expectProgramToReturn(t, p, IntValue(4))
}

func TestMatch(t *testing.T) {
	p := `
	match_func = (a) => match a {
		1 -> "yes"
		_ -> "no"
	}
	match_func(1)
  `
	expectProgramToReturn(t, p, StringValue("yes"))
}

func TestMutableVariable(t *testing.T) {
	p := `
  mut_x = 1
  mut_x.update(2)
  mut_x
  `
	expectProgramToReturn(t, p, IntValue(2))
}

func TestMutableVariableNoMut(t *testing.T) {
	p := `
  x = 1
  x.update(2)
  x
  `
	expectProgramToFail(t, p)
}

func TestMutableVariableFail(t *testing.T) {
	p := `
  mut_x = 1
  mut_x = 2
  mut_x
  `
	expectProgramToFail(t, p)
}

func TestAliasAndMultipleDispatch(t *testing.T) {
	p := `
	alias SomeEnum =
			"yes"
		| "no"
	
	get_result = (res:SomeEnum) => match res {
		"yes" -> "yeeees"
		"no"	-> "noooo"
	}
	
	get_result = (a) => a

	[get_result("yes"), get_result("sdff")]
	`
	expectProgramToReturn(t, p, &ListValue{StringValue("yeeees"), StringValue("sdff")})
}

func TestResultAlias(t *testing.T) {
	p := `
	val_ok = "test"
		.to_ok()
		.map((a) => a.append(" !"))
		.unwrap()

	val_err =  "failed"
		.to_err()
		.map((a) => a.append(" !"))
		.map_err((a) => "Err: " ++ a)
		.unwrap_err()

	[val_ok, val_err]
	`
	expectProgramToReturn(t, p, &ListValue{StringValue("test !"), StringValue("Err: failed")})
}

func TestListFunctions(t *testing.T) {
	p := `
	x = [1, 2, 3, 4, 5]
	increment = (n:Num) => n + 1
	x.map(increment).sum()
	`
	expectProgramToReturn(t, p, IntValue(20))
}

func TestPrecedence(t *testing.T) {
	p := `
concat_some_strings = (a, b, c) => {
  a.string() ++ " " ++ b.string() ++ " " ++ c.string()
}
concat_some_strings(1, 2, 3)
  `
	expectProgramToReturn(t, p, StringValue("1 2 3"))
}

func TestBaseTrim(t *testing.T) {
	p := `
	x = " some string  "
	x.trim()`
	expectProgramToReturn(t, p, StringValue("some string"))

	right := `
	x = " some string  "
	x.trim_right()`
	expectProgramToReturn(t, right, StringValue(" some string"))
}

func TestBaseTake(t *testing.T) {
	p := `
	x = "some string"
  x.take(2)
`
	expectProgramToReturn(t, p, StringValue("so"))
}

func TestBaseHasPrefixAt(t *testing.T) {
	p := `
	x = "some string"
  [x.has_prefix_at?("me", 2), x.has_prefix_at?("str", 3)]
`
	expectProgramToReturn(t, p, &ListValue{BoolValue(true), BoolValue(false)})
}

func TestBaseSplitBy(t *testing.T) {
	p := `
	x = "some, string, that does, something"
  x.split_by(", ")
`
	expectProgramToReturn(t, p, &ListValue{StringValue("some"), StringValue("string"), StringValue("that does"), StringValue("something")})
}

func TestBaseSplitByWithMatchingEnding(t *testing.T) {
	p := `
	x = "test\nsdfsdf\nsdfsdf\n"
  x.split_by("\n").length()
`
	expectProgramToReturn(t, p, IntValue(3))
}

func TestBaseMapLast(t *testing.T) {
	p := `
	x = [1, 2, 3]
	times = (n) => (a) => a * n
	x.map_last(times(10))
`
	expectProgramToReturn(t, p, &ListValue{IntValue(1), IntValue(2), IntValue(30)})
}

func TestVariableModificationInClosure(t *testing.T) {
	p := `
	x = [1]
	mut_var = "hello"
	x.map((v) => {
		mut_var.update("world")
	})
	mut_var
	`
	expectProgramToReturn(t, p, StringValue("world"))
}

func TestBaseSort(t *testing.T) {
	p := `
	x = [4, 2, 9, 1, 6, 7, 5, 6, 6]
	x.sort()
`
	expectProgramToReturn(t, p, &ListValue{IntValue(1), IntValue(2), IntValue(4), IntValue(5), IntValue(6), IntValue(6), IntValue(6), IntValue(7), IntValue(9)})
}

func TestBaseMergeHelperSort(t *testing.T) {
	p := `
	x = [1, 4, 9]
	y = [2, 3, 8]
	_merge_msort(x, y, SortOrder::Asc)
`
	expectProgramToReturn(t, p, &ListValue{IntValue(1), IntValue(2), IntValue(3), IntValue(4), IntValue(8), IntValue(9)})
}
