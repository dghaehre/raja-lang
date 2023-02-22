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

// func TestSimpleHappyTypecheck(t *testing.T) {
// 	p := `
// add_one = (i:Int) => i + 1
// add_one(1)`
// 	expectTypecheckToReturn(t, p, typedIntNode{nil})
// }
