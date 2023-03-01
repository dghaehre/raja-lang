package ast

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenizer struct {
	source []rune
	index  int

	fileName string
	line     int
	col      int
}

type Pos struct {
	fileName string
	line     int
	col      int
}

func (p Pos) String() string {
	return fmt.Sprintf("%s[%d:%d]", p.fileName, p.line, p.col)
}

type TokKind int

const (
	Comment TokKind = iota
	Unknown

	EmptyToken // Used as: nothing here
	Comma
	Dot
	LeftParen
	RightParen
	LeftBracket
	RightBracket
	LeftBrace
	RightBrace
	Assign
	FnArrow
	BranchArrow
	Colon
	DoubleColon

	// binary operators
	Plus
	PlusOther
	Minus
	Modulus
	Times
	Divide
	And
	Or
	Greater
	Less
	Eq
	Neq
	Geq
	Leq

	IndentEndStatment

	// keywords
	MatchKeyword
	AliasKeyword
	SinglePipeArrow
	DoublePipeArrow

	// identifiers and literals
	Underscore
	Identifier
	TrueLiteral
	FalseLiteral
	StringLiteral
	NumberLiteral
)

type Token struct {
	Kind TokKind
	Pos
	Payload string
}

func (t Token) String() string {
	switch t.Kind {
	case Comment:
		return fmt.Sprintf("#(%s)", t.Payload)
	case Comma:
		return ","
	case Dot:
		return "."
	case LeftParen:
		return "("
	case RightParen:
		return ")"
	case LeftBracket:
		return "["
	case RightBracket:
		return "]"
	case LeftBrace:
		return "{"
	case RightBrace:
		return "}"
	case Assign:
		return "="
	case FnArrow:
		return "=>"
	case BranchArrow:
		return "->"
	case Colon:
		return ":"
	case DoubleColon:
		return "::"
	case Plus:
		return "+"
	case Modulus:
		return "%"
	case PlusOther:
		return "++"
	case Minus:
		return "-"
	case Times:
		return "*"
	case Divide:
		return "/"
	case And:
		return "&"
	case Or:
		return "|"
	case Greater:
		return ">"
	case Less:
		return "<"
	case Eq:
		return "=="
	case Neq:
		return "!="
	case Geq:
		return ">="
	case Leq:
		return "<="
	case MatchKeyword:
		return "match"
	case AliasKeyword:
		return "alias"
	case Underscore:
		return "_"
	case Identifier:
		return fmt.Sprintf("var(%s)", t.Payload)
	case TrueLiteral:
		return "true"
	case FalseLiteral:
		return "false"
	case Unknown:
		return "<unknown>"
	case StringLiteral:
		return fmt.Sprintf("string(%s)", strconv.Quote(t.Payload))
	case NumberLiteral:
		return fmt.Sprintf("number(%s)", t.Payload)
	default:
		return "(unknown token)"
	}
}

func NewTokenizer(sourceString string, filename string) tokenizer {
	return tokenizer{
		source:   []rune(sourceString),
		index:    0,
		fileName: filename,
		line:     1,
		col:      0,
	}
}

func (t *tokenizer) currentPos() Pos {
	return Pos{
		fileName: t.fileName,
		line:     t.line,
		col:      t.col,
	}
}

func (t *tokenizer) isEOF() bool {
	return t.index == len(t.source)
}

func (t *tokenizer) peek() rune {
	return t.source[t.index]
}

func (t *tokenizer) peekAhead(n int) rune {
	if t.index+n >= len(t.source) {
		// NOTE that the empty ' ' is being used as a "non" value here.
		// might have to change this later..
		return ' '
	}
	return t.source[t.index+n]
}

// Consume and return the next rune
func (t *tokenizer) next() rune {
	char := t.source[t.index]
	if t.index < len(t.source) {
		t.index++
	}
	if char == '\n' {
		t.line++
		t.col = 0
	} else {
		t.col++
	}
	return char
}

func (t *tokenizer) back() {
	if t.index > 0 {
		t.index--
	}
	// TODO: reset col correctly if we need to go back up a line
	// This should be checked thourugly so that we parse the idents correctly
	if t.source[t.index] == '\n' {
		t.line--
	} else {
		t.col--
	}
}

func (t *tokenizer) readUntilRune(c rune) string {
	accumulator := []rune{}
	for !t.isEOF() && t.peek() != c {
		accumulator = append(accumulator, t.next())
	}
	return string(accumulator)
}

func (t *tokenizer) readValidIdentifier() string {
	accumulator := []rune{}
	for {
		if t.isEOF() {
			break
		}
		c := t.next()
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '_' || c == '?' || c == '!' {
			accumulator = append(accumulator, c)
		} else {
			t.back()
			break
		}
	}
	return string(accumulator)
}

func (t *tokenizer) readValidString() string {
	accumulator := []rune{}
	for {
		if t.isEOF() {
			break
		}
		c := t.next()
		if c == '"' {
			break
		} else {
			accumulator = append(accumulator, c)
		}
	}
	return string(accumulator)
}

func (t *tokenizer) readValidNumeral() string {
	sawDot := false
	accumulator := []rune{}
	for {
		if t.isEOF() {
			break
		}
		c := t.next()
		if unicode.IsDigit(c) {
			accumulator = append(accumulator, c)
		} else if c == '.' && !sawDot {
			sawDot = true
			accumulator = append(accumulator, c)
		} else {
			t.back()
			break
		}
	}
	return string(accumulator)
}
func (t *tokenizer) nextToken() Token {
	c := t.next()

	switch c {
	case ',':
		return Token{Kind: Comma, Pos: t.currentPos()}
	case '.':
		return Token{Kind: Dot, Pos: t.currentPos()}
	case '|':
		return Token{Kind: Or, Pos: t.currentPos()}
	case '(':
		return Token{Kind: LeftParen, Pos: t.currentPos()}
	case ')':
		return Token{Kind: RightParen, Pos: t.currentPos()}
	case '[':
		return Token{Kind: LeftBracket, Pos: t.currentPos()}
	case '\n':
		t.line++
		t.col = 0
		return Token{Kind: EmptyToken}
	case ']':
		return Token{Kind: RightBracket, Pos: t.currentPos()}
	case '{':
		return Token{Kind: LeftBrace, Pos: t.currentPos()}
	case '}':
		return Token{Kind: RightBrace, Pos: t.currentPos()}
	case ':':
		if !t.isEOF() && t.peek() == ':' {
			t.next()
			return Token{Kind: DoubleColon, Pos: t.currentPos()}
		}
		return Token{Kind: Colon, Pos: t.currentPos()}
	case '=':
		if !t.isEOF() && t.peek() == '>' {
			t.next()
			return Token{Kind: FnArrow, Pos: t.currentPos()}
		}
		if !t.isEOF() && t.peek() == '=' {
			t.next()
			return Token{Kind: Eq, Pos: t.currentPos()}
		}
		return Token{Kind: Assign, Pos: t.currentPos()}
	case '+':
		if !t.isEOF() && t.peek() == '+' {
			t.next()
			return Token{Kind: PlusOther, Pos: t.currentPos()}
		}
		return Token{Kind: Plus, Pos: t.currentPos()}
	case '*':
		return Token{Kind: Times, Pos: t.currentPos()}
	case '%':
		return Token{Kind: Modulus, Pos: t.currentPos()}
	case '>':
		if !t.isEOF() && t.peek() == '=' {
			t.next()
			return Token{Kind: Geq, Pos: t.currentPos()}
		}
		return Token{Kind: Greater, Pos: t.currentPos()}
	case '<':
		if !t.isEOF() && t.peek() == '=' {
			t.next()
			return Token{Kind: Leq, Pos: t.currentPos()}
		}
		return Token{Kind: Less, Pos: t.currentPos()}
	case '-':
		if !t.isEOF() && t.peek() == '>' {
			t.next()
			return Token{Kind: BranchArrow, Pos: t.currentPos()}
		}
		return Token{Kind: Minus, Pos: t.currentPos()}
	case '"':
		pos := t.currentPos()
		val := t.readValidString()
		return Token{
			Kind:    StringLiteral,
			Pos:     pos,
			Payload: val,
		}
	case '/':
		return Token{Kind: Divide, Pos: t.currentPos()}
	case '#':
		pos := t.currentPos()
		t.next()
		commentString := strings.TrimSpace(t.readUntilRune('\n'))
		return Token{
			Kind:    Comment,
			Pos:     pos,
			Payload: commentString,
		}

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		pos := t.currentPos()
		payload := string(c) + t.readValidNumeral()
		return Token{
			Kind:    NumberLiteral,
			Pos:     pos,
			Payload: payload,
		}
	default:
		pos := t.currentPos()
		payload := string(c) + t.readValidIdentifier()
		switch payload {
		case "_":
			return Token{Kind: Underscore, Pos: pos}
		case "!=":
			return Token{Kind: Neq, Pos: pos}
		case "match":
			return Token{Kind: MatchKeyword, Pos: pos}
		case "alias":
			return Token{Kind: AliasKeyword, Pos: pos}
		case "true":
			return Token{Kind: TrueLiteral, Pos: pos}
		case "false":
			return Token{Kind: FalseLiteral, Pos: pos}
		default:
			return Token{Kind: Identifier, Pos: pos, Payload: payload}
		}
	}

}

func (t *tokenizer) Tokenize() []Token {
	tokens := []Token{}

	if !t.isEOF() && t.peek() == '#' && t.peekAhead(1) == '!' {
		// shebang-style ignored line, keep taking until EOL
		t.readUntilRune('\n')
		if !t.isEOF() {
			t.next()
		}
	}

	// snip whitespace before
	for !t.isEOF() && unicode.IsSpace(t.peek()) {
		t.next()
	}

	// Tokenize rest of file
	for !t.isEOF() {
		next := t.nextToken()

		// Dont include comments (yet)
		if !(next.Kind == EmptyToken || next.Kind == Comment) {
			tokens = append(tokens, next)
		}

		// snip whitespace after
		for !t.isEOF() && unicode.IsSpace(t.peek()) {
			t.next()
		}
	}

	return tokens
}
