package main

import (
	"dghaehre/raja/ast"
)

func (c *TypecheckContext) LoadBuiltins() {
	c.LoadFunc("__print", typedIntNode{}, typedArg{name: "value"})

	c.LoadFunc("__string", typedStringNode{}, typedArg{name: "value"})
	c.LoadFunc("__int", typedIntNode{}, typedArg{name: "value"})
	c.LoadFunc("__args", typedListNode{})
	c.LoadFunc("__exit", typedAnyNode{}, typedArg{name: "value", alias: typedIntNode{}})
	c.LoadFunc("__read_file", typedStringNode{}, typedArg{name: "filename", alias: typedStringNode{}})

	// TODO: create typedAliasNode
	// - add Maybe and result to return types

	// c.LoadFunc("__index", typedArg{name: "iter", alias: typedAliasNode{}})
	//
	// // Types/Alias
	// c.LoadAlias("Int", c.rajaAliasInt)
	// c.LoadAlias("Float", c.rajaAliasFloat)
	// c.LoadAlias("Str", c.rajaAliasStr)
	// c.LoadAlias("List", c.rajaAliasList)
	// c.LoadAlias("Fn", c.rajaAliasFn)
	// c.LoadAlias("Enum", c.rajaAliasEnum)
	// // TODO: Bool
	//
	// _, err := c.LoadLib("base")
	// if err != nil {
	// 	panic(err)
	// }
}

func (c *TypecheckContext) LoadFunc(name string, returnType typedAstNode, args ...typedAstNode) {
	c.typecheckScope.put(name, typedFnNode{
		args: args,
		body: returnType,
	}, ast.Pos{})
}
