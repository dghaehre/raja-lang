package typecheck

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
	c.LoadAlias("Bool", typedBoolNode{})
	c.LoadAlias("Int", typedIntNode{})
	c.LoadAlias("Float", typedFloatNode{})
	c.LoadAlias("Str", typedStringNode{})
	c.LoadAlias("List", typedListNode{})
	c.LoadAlias("Fn", typedFnNode{})
	c.LoadAlias("Enum", typedEnumNode{})
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

func (c *TypecheckContext) LoadAlias(name string, returnType typedAstNode) {
	c.typecheckScope.put(name, typedAliasNode{
		targets: []typedAstNode{returnType},
	}, ast.Pos{})

}
