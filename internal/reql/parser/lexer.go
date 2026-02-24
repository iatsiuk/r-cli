package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// tokenType identifies the kind of lexical token.
type tokenType int

const (
	tokenEOF tokenType = iota
	tokenIdent
	tokenDot
	tokenLParen
	tokenRParen
	tokenLBracket
	tokenRBracket
	tokenLBrace
	tokenRBrace
	tokenComma
	tokenColon
	tokenString
	tokenNumber
	tokenBool
	tokenNull
	tokenArrow
)

// token is a single lexical unit with its type, raw value, and rune index.
type token struct {
	Type  tokenType
	Value string
	Pos   int
}

// lexer converts a ReQL string into a flat token slice.
type lexer struct {
	input []rune
	pos   int
}

// newLexer creates a lexer for the given input string.
func newLexer(input string) *lexer {
	return &lexer{input: []rune(input)}
}

// tokenize returns all tokens including a trailing EOF, or the first error.
func (l *lexer) tokenize() ([]token, error) {
	var tokens []token
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == tokenEOF {
			break
		}
	}
	return tokens, nil
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *lexer) next() (token, error) {
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		return token{Type: tokenEOF, Pos: l.pos}, nil
	}
	ch := l.input[l.pos]
	if ch == '=' {
		return l.readArrow()
	}
	if tok, ok := l.punctToken(ch); ok {
		return tok, nil
	}
	return l.readValue(ch)
}

func (l *lexer) readArrow() (token, error) {
	start := l.pos
	l.pos++ // consume '='
	if l.pos < len(l.input) && l.input[l.pos] == '>' {
		l.pos++ // consume '>'
		return token{Type: tokenArrow, Value: "=>", Pos: start}, nil
	}
	return token{}, fmt.Errorf("unexpected character '=' at position %d", start)
}

// punctTypes maps single-character punctuation to its token type.
var punctTypes = map[rune]tokenType{
	'.': tokenDot,
	'(': tokenLParen,
	')': tokenRParen,
	'[': tokenLBracket,
	']': tokenRBracket,
	'{': tokenLBrace,
	'}': tokenRBrace,
	',': tokenComma,
	':': tokenColon,
}

// punctToken returns a single-character punctuation token if ch matches.
func (l *lexer) punctToken(ch rune) (token, bool) {
	typ, ok := punctTypes[ch]
	if !ok {
		return token{}, false
	}
	start := l.pos
	l.pos++
	return token{Type: typ, Value: string(ch), Pos: start}, true
}

func (l *lexer) readValue(ch rune) (token, error) {
	start := l.pos
	switch {
	case ch == '"' || ch == '\'':
		return l.readString(ch)
	case ch == '-' || unicode.IsDigit(ch):
		return l.readNumber()
	case unicode.IsLetter(ch) || ch == '_':
		return l.readIdent()
	default:
		return token{}, fmt.Errorf("unexpected character %q at position %d", string(ch), start)
	}
}

func (l *lexer) readString(quote rune) (token, error) {
	start := l.pos
	l.pos++ // skip opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		switch ch {
		case '\\':
			l.pos++
			if l.pos >= len(l.input) {
				return token{}, fmt.Errorf("unterminated string at position %d", start)
			}
			esc := l.input[l.pos]
			r, ok := unescapeChar(esc)
			if !ok {
				return token{}, fmt.Errorf("unknown escape sequence '\\%c' at position %d", esc, l.pos-1)
			}
			sb.WriteRune(r)
			l.pos++
		case quote:
			l.pos++
			return token{Type: tokenString, Value: sb.String(), Pos: start}, nil
		default:
			sb.WriteRune(ch)
			l.pos++
		}
	}
	return token{}, fmt.Errorf("unterminated string at position %d", start)
}

func unescapeChar(ch rune) (rune, bool) {
	switch ch {
	case '"':
		return '"', true
	case '\'':
		return '\'', true
	case '\\':
		return '\\', true
	case 'n':
		return '\n', true
	case 't':
		return '\t', true
	case 'r':
		return '\r', true
	default:
		return 0, false
	}
}

func (l *lexer) readDigits() {
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.pos++
	}
}

func (l *lexer) readExponent() {
	if l.pos >= len(l.input) {
		return
	}
	if l.input[l.pos] != 'e' && l.input[l.pos] != 'E' {
		return
	}
	l.pos++
	if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
		l.pos++
	}
	l.readDigits()
}

func (l *lexer) readNumber() (token, error) {
	start := l.pos
	if l.input[l.pos] == '-' {
		l.pos++
		if l.pos >= len(l.input) || !unicode.IsDigit(l.input[l.pos]) {
			return token{}, fmt.Errorf("unexpected character '-' at position %d", start)
		}
	}
	l.readDigits()
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		l.pos++
		l.readDigits()
	}
	l.readExponent()
	return token{Type: tokenNumber, Value: string(l.input[start:l.pos]), Pos: start}, nil
}

func (l *lexer) readIdent() (token, error) {
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsLetter(l.input[l.pos]) || unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}
	val := string(l.input[start:l.pos])
	switch val {
	case "true", "false":
		return token{Type: tokenBool, Value: val, Pos: start}, nil
	case "null":
		return token{Type: tokenNull, Value: val, Pos: start}, nil
	}
	return token{Type: tokenIdent, Value: val, Pos: start}, nil
}
