package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType identifies the kind of lexical token.
type TokenType int

const (
	tokenEOF TokenType = iota
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
)

// Token is a single lexical unit with its type, raw value, and byte offset.
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer converts a ReQL string into a flat token slice.
type Lexer struct {
	input []rune
	pos   int
}

// newLexer creates a Lexer for the given input string.
func newLexer(input string) *Lexer {
	return &Lexer{input: []rune(input)}
}

// tokenize returns all tokens including a trailing EOF, or the first error.
func (l *Lexer) tokenize() ([]Token, error) {
	var tokens []Token
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

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *Lexer) next() (Token, error) {
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		return Token{Type: tokenEOF, Pos: l.pos}, nil
	}
	ch := l.input[l.pos]
	if tok, ok := l.punctToken(ch); ok {
		return tok, nil
	}
	return l.readValue(ch)
}

// punctTypes maps single-character punctuation to its token type.
var punctTypes = map[rune]TokenType{
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
func (l *Lexer) punctToken(ch rune) (Token, bool) {
	typ, ok := punctTypes[ch]
	if !ok {
		return Token{}, false
	}
	start := l.pos
	l.pos++
	return Token{Type: typ, Value: string(ch), Pos: start}, true
}

func (l *Lexer) readValue(ch rune) (Token, error) {
	start := l.pos
	switch {
	case ch == '"' || ch == '\'':
		return l.readString(ch)
	case ch == '-' || unicode.IsDigit(ch):
		return l.readNumber()
	case unicode.IsLetter(ch) || ch == '_':
		return l.readIdent()
	default:
		return Token{}, fmt.Errorf("unexpected character %q at position %d", string(ch), start)
	}
}

func (l *Lexer) readString(quote rune) (Token, error) {
	start := l.pos
	l.pos++ // skip opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		switch ch {
		case '\\':
			l.pos++
			if l.pos >= len(l.input) {
				return Token{}, fmt.Errorf("unterminated string at position %d", start)
			}
			esc := l.input[l.pos]
			r, ok := unescapeChar(esc)
			if !ok {
				return Token{}, fmt.Errorf("unknown escape sequence '\\%c' at position %d", esc, l.pos-1)
			}
			sb.WriteRune(r)
			l.pos++
		case quote:
			l.pos++
			return Token{Type: tokenString, Value: sb.String(), Pos: start}, nil
		default:
			sb.WriteRune(ch)
			l.pos++
		}
	}
	return Token{}, fmt.Errorf("unterminated string at position %d", start)
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

func (l *Lexer) readDigits() {
	for l.pos < len(l.input) && unicode.IsDigit(l.input[l.pos]) {
		l.pos++
	}
}

func (l *Lexer) readExponent() {
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

func (l *Lexer) readNumber() (Token, error) {
	start := l.pos
	if l.input[l.pos] == '-' {
		l.pos++
		if l.pos >= len(l.input) || !unicode.IsDigit(l.input[l.pos]) {
			return Token{}, fmt.Errorf("unexpected character '-' at position %d", start)
		}
	}
	l.readDigits()
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		l.pos++
		l.readDigits()
	}
	l.readExponent()
	return Token{Type: tokenNumber, Value: string(l.input[start:l.pos]), Pos: start}, nil
}

func (l *Lexer) readIdent() (Token, error) {
	start := l.pos
	for l.pos < len(l.input) && (unicode.IsLetter(l.input[l.pos]) || unicode.IsDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}
	val := string(l.input[start:l.pos])
	switch val {
	case "true", "false":
		return Token{Type: tokenBool, Value: val, Pos: start}, nil
	case "null":
		return Token{Type: tokenNull, Value: val, Pos: start}, nil
	}
	return Token{Type: tokenIdent, Value: val, Pos: start}, nil
}
