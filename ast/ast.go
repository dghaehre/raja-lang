package ast

import (
	"dghaehre/raja/util"
	"fmt"
	"strconv"
	"strings"
)

type AstNode interface {
	String() string
	Pos() Pos
}

type IntNode struct {
	Payload int64
	Tok     *Token
}

func (n IntNode) String() string {
	return strconv.FormatInt(n.Payload, 10)
}
func (n IntNode) Pos() Pos {
	return n.Tok.Pos
}

type FloatNode struct {
	Payload float64
	Tok     *Token
}

func (n FloatNode) String() string {
	return strconv.FormatFloat(n.Payload, 'g', -1, 64)
}
func (n FloatNode) Pos() Pos {
	return n.Tok.Pos
}

type BoolNode struct {
	Payload bool
	Tok     *Token
}

func (n BoolNode) String() string {
	if n.Payload {
		return "true"
	}
	return "false"
}
func (n BoolNode) Pos() Pos {
	return n.Tok.Pos
}

type IdentifierNode struct {
	Payload string
	Tok     *Token
}

func (n IdentifierNode) String() string {
	return n.Payload
}
func (n IdentifierNode) Pos() Pos {
	return n.Tok.Pos
}

type UnderscoreNode struct {
	tok *Token
}

func (n UnderscoreNode) String() string {
	return "_"
}
func (n UnderscoreNode) Pos() Pos {
	return n.tok.Pos
}

type StringNode struct {
	Payload []byte
	Tok     *Token
}

func (n StringNode) String() string {
	return fmt.Sprintf("%s", strconv.Quote(string(n.Payload)))
}
func (n StringNode) Pos() Pos {
	return n.Tok.Pos
}

// TODO: isLocal
type AssignmentNode struct {
	Left  AstNode
	Right AstNode
	Tok   *Token
}

func (n AssignmentNode) String() string {
	return n.Left.String() + " = " + n.Right.String()
}
func (n AssignmentNode) Pos() Pos {
	return n.Tok.Pos
}

type BinaryNode struct {
	Op    TokKind
	Left  AstNode
	Right AstNode
	Tok   *Token
}

func (n BinaryNode) String() string {
	opTok := Token{Kind: n.Op}
	return "(" + n.Left.String() + " " + opTok.String() + " " + n.Right.String() + ")"
}
func (n BinaryNode) Pos() Pos {
	return n.Tok.Pos
}

type BlockNode struct {
	Exprs []AstNode
	Tok   *Token
}

func (n BlockNode) String() string {
	exprStrings := make([]string, len(n.Exprs))
	for i, ex := range n.Exprs {
		exprStrings[i] = ex.String()
	}
	return "{ " + strings.Join(exprStrings, ", ") + " }"
}

func (n BlockNode) Pos() Pos {
	return n.Tok.Pos
}

type Arg struct {
	Name  string
	Alias string // optional
}

// NOTE: why does this not implement fmt.Stringer?
func (a Arg) String() string {
	if a.Alias == "" {
		return a.Name
	}
	return fmt.Sprintf("%s:%s", a.Name, a.Alias)
}

type FnNode struct {
	Args []Arg
	Body AstNode
	Tok  *Token
}

func (n FnNode) String() string {
	// This is stupid...
	// TODO: gotta be a better way to cast to []fmt.Stringer
	var args []fmt.Stringer
	for _, v := range n.Args {
		args = append(args, v)
	}
	return fmt.Sprintf("(%s) => %s", util.StringsJoin(args, ", "), n.Body.String())
}

func (n FnNode) Pos() Pos {
	return n.Tok.Pos
}

type FnCallNode struct {
	Fn   AstNode
	Args []AstNode
	Tok  *Token
}

func (n FnCallNode) String() string {
	argStrings := make([]string, len(n.Args))
	for i, arg := range n.Args {
		argStrings[i] = arg.String()
	}
	return fmt.Sprintf("fncall[%s](%s)", n.Fn, strings.Join(argStrings, ", "))
}
func (n FnCallNode) Pos() Pos {
	return n.Tok.Pos
}

// Used to get which variable to update when using the builtin update function
func (n FnCallNode) FirstArgName() string {
	if len(n.Args) == 0 {
		return ""
	}
	return n.Args[0].String()
}

type ListNode struct {
	Elems []AstNode
	Tok   *Token
}

func (n ListNode) String() string {
	elemStrings := make([]string, len(n.Elems))
	for i, el := range n.Elems {
		elemStrings[i] = el.String()
	}
	return "[" + strings.Join(elemStrings, ", ") + "]"
}
func (n ListNode) Pos() Pos {
	return n.Tok.Pos
}

// Special in the sense that it is not a node.
type MatchBranch struct {
	Target AstNode // the "pattern" to match. Maybe I should do something fancy here later
	Body   AstNode
}

func (n MatchBranch) String() string {
	return n.Target.String() + " -> " + n.Body.String()
}

type MatchNode struct {
	Cond     AstNode
	Branches []MatchBranch
	Tok      *Token
}

func (n MatchNode) String() string {
	branchStrings := make([]string, len(n.Branches))
	for i, br := range n.Branches {
		branchStrings[i] = br.String()
	}
	return "match " + n.Cond.String() + " {" + strings.Join(branchStrings, " ") + "}"
}

func (n MatchNode) Pos() Pos {
	return n.Tok.Pos
}

type AliasNode struct {
	Name    string
	Targets []AstNode
	Tok     *Token
}

func (t AliasNode) String() string {
	targetStrings := make([]string, len(t.Targets))
	for i, target := range t.Targets {
		targetStrings[i] = target.String()
	}
	return "alias " + t.Name + " = " + strings.Join(targetStrings, " | ")
}

func (t AliasNode) Pos() Pos {
	return t.Tok.Pos
}

type EnumNode struct {
	Parent string
	Name   string
	Args   []AstNode
	Tok    *Token
}

func (e EnumNode) String() string {
	argsStrings := make([]string, len(e.Args))
	for i, target := range e.Args {
		argsStrings[i] = target.String()
	}
	n := e.Parent + "::" + e.Name
	if len(e.Args) > 0 {
		return n + "(" + strings.Join(argsStrings, ", ") + ")"
	}
	return n
}

func (e EnumNode) Pos() Pos {
	return e.Tok.Pos
}
