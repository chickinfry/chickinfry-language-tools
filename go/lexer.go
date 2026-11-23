package chiklang

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// lexer turns source text into tokens and trivia. Trivia is kept because the
// concrete syntax tree must serialize back to the exact source.
type lexer struct {
	src    string
	pos    int
	line   int
	column int
	tokens []Token
	trivia []Trivia
	diags  []Diagnostic
}

// TriviaKind distinguishes the pieces of a file that carry no meaning.
type TriviaKind int

const (
	// Whitespace is spaces and tabs.
	Whitespace TriviaKind = iota
	// Newline is a line break.
	Newline
	// Comment is a line or block comment.
	Comment
)

// Trivia is text the parser keeps but does not interpret.
type Trivia struct {
	Kind  TriviaKind
	Text  string
	Start int
	End   int
	Range Range
	Block bool
}

func newLexer(src string) *lexer {
	return &lexer{src: src, line: 1, column: 1}
}

func (l *lexer) at(offset int) rune {
	if l.pos+offset >= len(l.src) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.src[l.pos+offset:])
	return r
}

func (l *lexer) advance(n int) {
	for range n {
		if l.pos >= len(l.src) {
			return
		}
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		l.pos += size
		if r == '\n' {
			l.line++
			l.column = 1
		} else {
			l.column++
		}
	}
}

func (l *lexer) here() Pos { return Pos{Line: l.line, Column: l.column} }

func (l *lexer) fail(code string, start Pos, message string) {
	l.diags = append(l.diags, Diagnostic{
		Code: code, Severity: "error", Range: Range{Start: start, End: l.here()}, Message: message,
	})
}

func (l *lexer) addTrivia(kind TriviaKind, start int, startPos Pos, end Pos) {
	l.trivia = append(l.trivia, Trivia{
		Kind: kind, Text: l.src[start:l.pos], Start: start, End: l.pos,
		Range: Range{Start: startPos, End: end}, Block: kind == Comment && strings.HasPrefix(l.src[start:], "/*"),
	})
}

// run lexes the whole input. It always returns what it read so a caller can
// still report positions after a failure.
func (l *lexer) run() ([]Token, []Trivia, []Diagnostic) {
	for l.pos < len(l.src) {
		r := l.at(0)
		switch {
		case r == ' ' || r == '\t' || r == '\r':
			l.runTrivia(Whitespace)
		case r == '\n':
			l.runTrivia(Newline)
		case r == '/' && l.at(1) == '/':
			l.lineComment()
		case r == '/' && l.at(1) == '*':
			l.blockComment()
		case r == '"':
			l.string()
		case r >= '0' && r <= '9' || r == '-' && l.at(1) >= '0' && l.at(1) <= '9':
			l.number()
		case isIdentStart(r):
			l.ident()
		case strings.ContainsRune("{}(),;:=<>?[]-/|.", r):
			start, startPos := l.pos, l.here()
			l.advance(1)
			l.tokens = append(l.tokens, Token{Kind: Punct, Text: string(r), Start: start, End: l.pos, Line: startPos.Line, Column: startPos.Column})
		default:
			startPos := l.here()
			l.advance(1)
			l.fail(CodeUnexpectedCharacter, startPos, fmt.Sprintf("unexpected character %q", r))
		}
	}
	end := l.here()
	l.tokens = append(l.tokens, Token{Kind: EOF, Start: l.pos, End: l.pos, Line: end.Line, Column: end.Column})
	return l.tokens, l.trivia, l.diags
}

func (l *lexer) runTrivia(kind TriviaKind) {
	start, startPos := l.pos, l.here()
	for l.pos < len(l.src) {
		r := l.at(0)
		if kind == Whitespace && (r == ' ' || r == '\t' || r == '\r') {
			l.advance(1)
			continue
		}
		if kind == Newline && r == '\n' {
			l.advance(1)
			continue
		}
		break
	}
	l.addTrivia(kind, start, startPos, l.here())
}

func (l *lexer) lineComment() {
	start, startPos := l.pos, l.here()
	for l.pos < len(l.src) && l.at(0) != '\n' {
		l.advance(1)
	}
	l.addTrivia(Comment, start, startPos, l.here())
}

func (l *lexer) blockComment() {
	start, startPos := l.pos, l.here()
	l.advance(2)
	for {
		if l.pos >= len(l.src) {
			l.fail(CodeUnterminatedComment, startPos, "block comment has no closing */")
			l.addTrivia(Comment, start, startPos, l.here())
			return
		}
		if l.at(0) == '*' && l.at(1) == '/' {
			l.advance(2)
			break
		}
		if l.at(0) == '/' && l.at(1) == '*' {
			l.fail(CodeNestedComment, l.here(), "block comments do not nest")
			return
		}
		l.advance(1)
	}
	l.addTrivia(Comment, start, startPos, l.here())
}

func (l *lexer) string() {
	start, startPos := l.pos, l.here()
	l.advance(1)
	var value strings.Builder
	for {
		if l.pos >= len(l.src) || l.at(0) == '\n' {
			l.fail(CodeUnterminatedString, startPos, "string literal is not closed")
			return
		}
		r := l.at(0)
		if r == '"' {
			l.advance(1)
			l.tokens = append(l.tokens, Token{Kind: String, Text: value.String(), Raw: l.src[start:l.pos], Start: start, End: l.pos, Line: startPos.Line, Column: startPos.Column})
			return
		}
		if r != '\\' {
			value.WriteRune(r)
			l.advance(1)
			continue
		}
		escapePos := l.here()
		l.advance(1)
		escaped := l.at(0)
		l.advance(1)
		switch escaped {
		case '"':
			value.WriteByte('"')
		case '\\':
			value.WriteByte('\\')
		case '/':
			value.WriteByte('/')
		case 'b':
			value.WriteByte('\b')
		case 'f':
			value.WriteByte('\f')
		case 'n':
			value.WriteByte('\n')
		case 'r':
			value.WriteByte('\r')
		case 't':
			value.WriteByte('\t')
		case 'u':
			decoded, ok := l.unicodeEscape()
			if !ok {
				l.fail(CodeInvalidEscape, escapePos, "\\u must be followed by four hex digits")
				return
			}
			value.WriteRune(decoded)
		default:
			l.fail(CodeInvalidEscape, escapePos, fmt.Sprintf("invalid escape \\%c", escaped))
			return
		}
	}
}

// unicodeEscape reads four hex digits and decodes a surrogate pair when the
// first code unit is a high surrogate.
func (l *lexer) unicodeEscape() (rune, bool) {
	first, ok := l.hex4()
	if !ok {
		return 0, false
	}
	if first >= 0xD800 && first <= 0xDBFF {
		if l.at(0) != '\\' || l.at(1) != 'u' {
			return 0, false
		}
		l.advance(2)
		second, ok := l.hex4()
		if !ok || second < 0xDC00 || second > 0xDFFF {
			return 0, false
		}
		return 0x10000 + (first-0xD800)<<10 + (second - 0xDC00), true
	}
	if first >= 0xDC00 && first <= 0xDFFF {
		return 0, false
	}
	return first, true
}

func (l *lexer) hex4() (rune, bool) {
	if l.pos+4 > len(l.src) {
		return 0, false
	}
	var value rune
	for range 4 {
		r := l.at(0)
		var digit rune
		switch {
		case r >= '0' && r <= '9':
			digit = r - '0'
		case r >= 'a' && r <= 'f':
			digit = r - 'a' + 10
		case r >= 'A' && r <= 'F':
			digit = r - 'A' + 10
		default:
			return 0, false
		}
		value = value<<4 | digit
		l.advance(1)
	}
	return value, true
}

// number lexes an integer or decimal. A number is a configuration value, not a
// declaration type, so it keeps its source spelling.
func (l *lexer) number() {
	start, startPos := l.pos, l.here()
	if l.at(0) == '-' {
		l.advance(1)
	}
	for l.pos < len(l.src) && (l.at(0) >= '0' && l.at(0) <= '9' || l.at(0) == '.') {
		l.advance(1)
	}
	l.tokens = append(l.tokens, Token{Kind: Number, Text: l.src[start:l.pos], Raw: l.src[start:l.pos], Start: start, End: l.pos, Line: startPos.Line, Column: startPos.Column})
}

func (l *lexer) ident() {
	start, startPos := l.pos, l.here()
	for l.pos < len(l.src) && isIdentPart(l.at(0)) {
		l.advance(1)
	}
	l.tokens = append(l.tokens, Token{Kind: Ident, Text: l.src[start:l.pos], Start: start, End: l.pos, Line: startPos.Line, Column: startPos.Column})
}

func isIdentStart(r rune) bool {
	return r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || r >= '0' && r <= '9'
}
