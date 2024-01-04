package typecheck

import (
	"dghaehre/raja/ast"
)

var maybeAlias TypedAstNode = typedAliasNode{
	name: "Maybe",
	targets: []TypedAstNode{
		typedEnumNode{
			parent: "Maybe",
			name:   "Some",
			args: []TypedAstNode{
				typedAnyNode{},
			},
		},
		typedEnumNode{
			parent: "Maybe",
			name:   "None",
			args:   []TypedAstNode{},
		},
	},
}

var resultAlias TypedAstNode = typedAliasNode{
	name: "Result",
	targets: []TypedAstNode{
		typedEnumNode{
			parent: "Result",
			name:   "Ok",
			args: []TypedAstNode{
				typedAnyNode{},
			},
		},
		typedEnumNode{
			parent: "Result",
			name:   "Err",
			args: []TypedAstNode{
				typedAnyNode{},
			},
		},
	},
}

var iteratorAlias TypedAstNode = typedAliasNode{
	name: "Iterator",
	targets: []TypedAstNode{
		typedStringNode{},
		typedListNode{},
	},
}

var numAlias TypedAstNode = typedAliasNode{
	name: "Num",
	targets: []TypedAstNode{
		typedIntNode{},
		typedFloatNode{},
	},
}

var intAlias TypedAstNode = typedAliasNode{
	name: "Int",
	targets: []TypedAstNode{
		typedIntNode{},
	},
}

var floatAlias TypedAstNode = typedAliasNode{
	name: "Float",
	targets: []TypedAstNode{
		typedFloatNode{},
	},
}

func (c *TypecheckContext) LoadBuiltins() {
	c.LoadFunc("__print", typedIntNode{}, typedArg{name: "value"})

	c.LoadFunc("__string", typedStringNode{}, typedArg{name: "value"})
	c.LoadFunc("__int", resultAlias, typedArg{name: "value"})
	c.LoadFunc("__args", typedListNode{})
	c.LoadFunc("__exit", typedAnyNode{}, typedArg{name: "value", alias: typedIntNode{}})
	c.LoadFunc("__read_file", resultAlias, typedArg{name: "filename", alias: typedStringNode{}})
	c.LoadFunc("__length", typedIntNode{}, typedArg{name: "iter", alias: iteratorAlias})
	c.LoadFunc("__index", maybeAlias, typedArg{name: "iter", alias: iteratorAlias}, typedArg{name: "index", alias: typedIntNode{}}, typedArg{name: "unsafe?", alias: typedBoolNode{}})

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

func (c *TypecheckContext) LoadFunc(name string, returnType TypedAstNode, args ...TypedAstNode) {
	c.typecheckScope.put(name, typedFnNode{
		args: args,
		body: returnType,
	}, ast.Pos{})
}

func (c *TypecheckContext) LoadAlias(name string, returnType TypedAstNode) {
	c.typecheckScope.put(name, typedAliasNode{
		name:    name,
		targets: []TypedAstNode{returnType},
	}, ast.Pos{})
}
