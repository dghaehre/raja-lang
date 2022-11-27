package main

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

type pos struct {
	fileName string
	line     int
	col      int
}

func (p pos) String() string {
	return fmt.Sprintf("%s[%d:%d]", p.fileName, p.line, p.col)
}

type tokKind int

const (
	comment tokKind = iota
	unknown

	comma
	dot
	leftParen
	rightParen
	leftBracket
	rightBracket
	leftBrace
	rightBrace
	assign
	fnArrow
	colon

	// indent stuff
	// TODO: remove indent
	indentOpen
	indentClose
	indentEndStatment

	// binary operators
	// TODO: remove this, and let the standard lib handle this
	plus
	minus
	times
	divide
	and
	or
	greater
	less
	eq

	// temp tokKin
	lineBreak

	// keywords
	matchKeyword
	singlePipeArrow
	doublePipeArrow

	// identifiers and literals
	underscore
	identifier
	trueLiteral
	falseLiteral
	stringLiteral
	numberLiteral
)

type token struct {
	kind tokKind
	pos
	payload string
}

func (t token) String() string {
	switch t.kind {
	case comment:
		return fmt.Sprintf("//(%s)", t.payload)
	case comma:
		return ","
	case dot:
		return "."
	case leftParen:
		return "("
	case rightParen:
		return ")"
	case leftBracket:
		return "["
	case rightBracket:
		return "]"
	case leftBrace:
		return "{"
	case rightBrace:
		return "}"
	case assign:
		return "="
	case fnArrow:
		return "=>"
	case colon:
		return ":"
	case plus:
		return "+"
	case minus:
		return "-"
	case times:
		return "*"
	case divide:
		return "/"
	case and:
		return "&"
	case or:
		return "|"
	case greater:
		return ">"
	case less:
		return "<"
	case eq:
		return "=="
	case lineBreak:
		return "⏎"
	case matchKeyword:
		return "match"
	case underscore:
		return "_"
	case indentOpen:
		return "<indent>"
	case indentClose:
		return "<deindent>"
	case indentEndStatment:
		return "<statement end>"
	case identifier:
		return fmt.Sprintf("var(%s)", t.payload)
	case trueLiteral:
		return "true"
	case falseLiteral:
		return "false"
	case unknown:
		return "<unknown>"
	case stringLiteral:
		return fmt.Sprintf("string(%s)", strconv.Quote(t.payload))
	case numberLiteral:
		return fmt.Sprintf("number(%s)", t.payload)
	default:
		return "(unknown token)"
	}
}

func newTokenizer(sourceString string) tokenizer {
	return tokenizer{
		source:   []rune(sourceString),
		index:    0,
		fileName: "",
		line:     1,
		col:      0,
	}
}

func (t *tokenizer) currentPos() pos {
	return pos{
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
	// if char == '\n' {
	// t.line++
	// 	t.col = 0
	// } else {
	t.col++
	// }
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
func (t *tokenizer) nextToken() token {
	c := t.next()

	switch c {
	case ',':
		return token{kind: comma, pos: t.currentPos()}
	case '(':
		return token{kind: leftParen, pos: t.currentPos()}
	case ')':
		return token{kind: rightParen, pos: t.currentPos()}
	case '[':
		return token{kind: leftBracket, pos: t.currentPos()}
	case '\n':
		t.line++
		t.col = 0
		return token{kind: lineBreak, pos: t.currentPos()}
	case ']':
		return token{kind: rightBracket, pos: t.currentPos()}
	case '{':
		return token{kind: leftBrace, pos: t.currentPos()}
	case '}':
		return token{kind: rightBrace, pos: t.currentPos()}
	case '=':
		if !t.isEOF() && t.peek() == '>' {
			t.next()
			return token{kind: fnArrow, pos: t.currentPos()}
		}
		if !t.isEOF() && t.peek() == '=' {
			t.next()
			return token{kind: eq, pos: t.currentPos()}
		}
		return token{kind: assign, pos: t.currentPos()}
	case '+':
		return token{kind: plus, pos: t.currentPos()}
	case '/':
		if !t.isEOF() && t.peek() == '/' {
			pos := t.currentPos()
			t.next()
			commentString := strings.TrimSpace(t.readUntilRune('\n'))
			return token{
				kind:    comment,
				pos:     pos,
				payload: commentString,
			}
		}
		return token{kind: divide, pos: t.currentPos()}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		pos := t.currentPos()
		payload := string(c) + t.readValidNumeral()
		return token{
			kind:    numberLiteral,
			pos:     pos,
			payload: payload,
		}
	default:
		pos := t.currentPos()
		payload := string(c) + t.readValidIdentifier()
		switch payload {
		case "_":
			return token{kind: underscore, pos: pos}
		case "match":
			return token{kind: matchKeyword, pos: pos}
		case "true":
			return token{kind: trueLiteral, pos: pos}
		case "false":
			return token{kind: falseLiteral, pos: pos}
		default:
			return token{kind: identifier, pos: pos, payload: payload}
		}
	}

}

func (t *tokenizer) tokenize() []token {
	tokens := []token{}

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
	last := token{}
	for !t.isEOF() {
		next := t.nextToken()

		// Dont include multiple linebreaks in a row
		if !(last.kind == lineBreak && next.kind == lineBreak) {
			tokens = append(tokens, next)
		}
		last = next

		// snip whitespace after
		for !t.isEOF() && unicode.IsSpace(t.peek()) {
			t.next()
		}
	}

	return tokens
}
