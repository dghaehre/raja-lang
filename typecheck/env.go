package typecheck

import (
	"dghaehre/raja/ast"
)

func maybe() typedAstNode {
	return typedAliasNode{
		targets: []typedAstNode{
			typedEnumNode{
				parent: "Maybe",
				name:   "Some",
				args: []typedAstNode{
					typedAnyNode{},
				},
			},
			typedEnumNode{
				parent: "Maybe",
				name:   "None",
				args:   []typedAstNode{},
			},
		},
	}
}

func iterator() typedAstNode {
	return typedAliasNode{
		targets: []typedAstNode{
			typedStringNode{},
			typedListNode{},
		},
	}
}

func (c *TypecheckContext) LoadBuiltins() {
	c.LoadFunc("__print", typedIntNode{}, typedArg{name: "value"})

	c.LoadFunc("__string", typedStringNode{}, typedArg{name: "value"})
	c.LoadFunc("__int", typedIntNode{}, typedArg{name: "value"})
	c.LoadFunc("__args", typedListNode{})
	c.LoadFunc("__exit", typedAnyNode{}, typedArg{name: "value", alias: typedIntNode{}})
	c.LoadFunc("__read_file", typedStringNode{}, typedArg{name: "filename", alias: typedStringNode{}})
	c.LoadFunc("__index", maybe(), typedArg{name: "iter", alias: iterator()}, typedArg{name: "index", alias: typedIntNode{}}, typedArg{name: "unsafe?", alias: typedBoolNode{}})

	// c.LoadFunc("__index", typedArg{name: "iter", alias: typedAliasNode{}})
	//
	// // Types/Alias
	c.LoadAlias("Bool", typedBoolNode{})
	c.LoadAlias("Int", typedIntNode{})
	c.LoadAlias("Float", typedFloatNode{})
	c.LoadAlias("Str", typedStringNode{})
	c.LoadAlias("List", typedListNode{})
	c.LoadAlias("Fn", typedAnyFnNode{})
	c.LoadAlias("Enum", typedEnumNode{})
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
