package parser

import (
	"fmt"
	"strconv"

	"r-cli/internal/reql"
)

// Parse tokenizes input and builds a reql.Term.
func Parse(input string) (reql.Term, error) {
	toks, err := newLexer(input).tokenize()
	if err != nil {
		return reql.Term{}, fmt.Errorf("parse: %w", err)
	}
	p := &parser{tokens: toks}
	t, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type != tokenEOF {
		tok := p.peek()
		return reql.Term{}, fmt.Errorf("unexpected token %q at position %d", tok.Value, tok.Pos)
	}
	return t, nil
}

type parser struct {
	tokens []Token
	pos    int
}

func (p *parser) peek() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: tokenEOF}
}

func (p *parser) advance() Token {
	tok := p.peek()
	if tok.Type != tokenEOF {
		p.pos++
	}
	return tok
}

func (p *parser) expect(tt TokenType) (Token, error) {
	tok := p.peek()
	if tok.Type != tt {
		return Token{}, fmt.Errorf("expected token %d, got %q at position %d", int(tt), tok.Value, tok.Pos)
	}
	return p.advance(), nil
}

func (p *parser) parseExpr() (reql.Term, error) {
	t, err := p.parsePrimary()
	if err != nil {
		return reql.Term{}, err
	}
	return p.parseChain(t)
}

func (p *parser) parsePrimary() (reql.Term, error) {
	tok := p.peek()
	switch {
	case tok.Type == tokenIdent && tok.Value == "r":
		p.advance()
		return p.parseRExpr()
	case tok.Type == tokenLBrace:
		return p.parseObjectTerm()
	case tok.Type == tokenLBracket:
		return p.parseArrayTerm()
	default:
		return p.parseDatumTerm()
	}
}

func (p *parser) parseRExpr() (reql.Term, error) {
	if _, err := p.expect(tokenDot); err != nil {
		return reql.Term{}, err
	}
	method, err := p.expect(tokenIdent)
	if err != nil {
		return reql.Term{}, err
	}
	fn, ok := rBuilders[method.Value]
	if !ok {
		return reql.Term{}, fmt.Errorf("unknown r.%s at position %d", method.Value, method.Pos)
	}
	return fn(p)
}

// rBuilders maps r.method names to builder functions.
// initialized in init() to break the static dependency cycle.
var rBuilders map[string]func(*parser) (reql.Term, error)

func parseRDB(p *parser) (reql.Term, error) {
	name, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.DB(name), nil
}

func parseRRow(p *parser) (reql.Term, error) {
	t := reql.Row()
	if p.peek().Type != tokenLParen {
		return t, nil
	}
	field, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Bracket(field), nil
}

func parseRDesc(p *parser) (reql.Term, error) {
	name, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Desc(name), nil
}

func parseRAsc(p *parser) (reql.Term, error) {
	name, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Asc(name), nil
}

func parseRMinVal(_ *parser) (reql.Term, error) {
	return reql.MinVal(), nil
}

func parseRMaxVal(_ *parser) (reql.Term, error) {
	return reql.MaxVal(), nil
}

func parseRBranch(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	iargs := make([]interface{}, len(args))
	for i, a := range args {
		iargs[i] = a
	}
	return reql.Branch(iargs...), nil
}

func parseRError(p *parser) (reql.Term, error) {
	msg, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Error(msg), nil
}

func parseRArgs(p *parser) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Args(arg), nil
}

func parseRExpr(p *parser) (reql.Term, error) {
	return p.parseOneArg()
}

func (p *parser) parseChain(t reql.Term) (reql.Term, error) {
	for {
		switch p.peek().Type {
		case tokenDot:
			p.advance()
			method, err := p.expect(tokenIdent)
			if err != nil {
				return reql.Term{}, err
			}
			fn, ok := chainBuilders[method.Value]
			if !ok {
				return reql.Term{}, fmt.Errorf("unknown method .%s at position %d", method.Value, method.Pos)
			}
			t, err = fn(p, t)
			if err != nil {
				return reql.Term{}, err
			}
		case tokenLParen:
			// bracket notation: term("field")
			field, err := p.parseOneStringArg()
			if err != nil {
				return reql.Term{}, err
			}
			t = t.Bracket(field)
		default:
			return t, nil
		}
	}
}

// chainBuilders maps method names to chain builder functions.
// initialized in init() to break the static dependency cycle.
var chainBuilders map[string]func(*parser, reql.Term) (reql.Term, error)

func init() {
	rBuilders = map[string]func(*parser) (reql.Term, error){
		"db":     parseRDB,
		"row":    parseRRow,
		"desc":   parseRDesc,
		"asc":    parseRAsc,
		"minval": parseRMinVal,
		"maxval": parseRMaxVal,
		"branch": parseRBranch,
		"error":  parseRError,
		"args":   parseRArgs,
		"expr":   parseRExpr,
	}
	chainBuilders = map[string]func(*parser, reql.Term) (reql.Term, error){
		"table":   chainTable,
		"filter":  chainFilter,
		"get":     chainGet,
		"insert":  chainInsert,
		"orderBy": chainOrderBy,
		"limit":   chainLimit,
		"gt":      chainGt,
	}
}

func chainTable(p *parser, t reql.Term) (reql.Term, error) {
	name, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Table(name), nil
}

func chainFilter(p *parser, t reql.Term) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Filter(arg), nil
}

func chainGet(p *parser, t reql.Term) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Get(arg), nil
}

func chainInsert(p *parser, t reql.Term) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Insert(arg), nil
}

func chainOrderBy(p *parser, t reql.Term) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	iargs := make([]interface{}, len(args))
	for i, a := range args {
		iargs[i] = a
	}
	return t.OrderBy(iargs...), nil
}

func chainLimit(p *parser, t reql.Term) (reql.Term, error) {
	n, err := p.parseOneIntArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Limit(n), nil
}

func chainGt(p *parser, t reql.Term) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Gt(arg), nil
}

// parseOneArg parses (expr) and returns the term.
func (p *parser) parseOneArg() (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	t, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t, nil
}

// parseOneStringArg parses (string_literal) and returns the string value.
func (p *parser) parseOneStringArg() (string, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return "", err
	}
	tok, err := p.expect(tokenString)
	if err != nil {
		return "", err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return "", err
	}
	return tok.Value, nil
}

// parseOneIntArg parses (integer) and returns the int value.
func (p *parser) parseOneIntArg() (int, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return 0, err
	}
	tok, err := p.expect(tokenNumber)
	if err != nil {
		return 0, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(tok.Value)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", tok.Value)
	}
	return n, nil
}

// parseArgList parses (arg1, arg2, ...) and returns a slice of terms.
func (p *parser) parseArgList() ([]reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}
	var args []reql.Term
	for p.peek().Type != tokenRParen && p.peek().Type != tokenEOF {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().Type == tokenComma {
			p.advance()
		}
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}
	return args, nil
}

// parseObjectTerm parses {key: val, ...} and returns a Datum wrapping a map.
func (p *parser) parseObjectTerm() (reql.Term, error) {
	if _, err := p.expect(tokenLBrace); err != nil {
		return reql.Term{}, err
	}
	m := make(map[string]interface{})
	for p.peek().Type != tokenRBrace && p.peek().Type != tokenEOF {
		key, err := p.parseObjectKey()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenColon); err != nil {
			return reql.Term{}, err
		}
		val, err := p.parseExpr()
		if err != nil {
			return reql.Term{}, err
		}
		m[key] = val
		if p.peek().Type == tokenComma {
			p.advance()
		}
	}
	if _, err := p.expect(tokenRBrace); err != nil {
		return reql.Term{}, err
	}
	return reql.Datum(m), nil
}

func (p *parser) parseObjectKey() (string, error) {
	tok := p.peek()
	if tok.Type == tokenIdent || tok.Type == tokenString {
		p.advance()
		return tok.Value, nil
	}
	return "", fmt.Errorf("expected object key at position %d, got %q", tok.Pos, tok.Value)
}

// parseArrayTerm parses [val, ...] and returns a MAKE_ARRAY term.
func (p *parser) parseArrayTerm() (reql.Term, error) {
	if _, err := p.expect(tokenLBracket); err != nil {
		return reql.Term{}, err
	}
	var items []interface{}
	for p.peek().Type != tokenRBracket && p.peek().Type != tokenEOF {
		item, err := p.parseExpr()
		if err != nil {
			return reql.Term{}, err
		}
		items = append(items, item)
		if p.peek().Type == tokenComma {
			p.advance()
		}
	}
	if _, err := p.expect(tokenRBracket); err != nil {
		return reql.Term{}, err
	}
	return reql.Array(items...), nil
}

func (p *parser) parseDatumTerm() (reql.Term, error) {
	tok := p.peek()
	switch tok.Type {
	case tokenString:
		p.advance()
		return reql.Datum(tok.Value), nil
	case tokenNumber:
		p.advance()
		v, err := parseNumberValue(tok.Value)
		if err != nil {
			return reql.Term{}, fmt.Errorf("invalid number %q: %w", tok.Value, err)
		}
		return reql.Datum(v), nil
	case tokenBool:
		p.advance()
		return reql.Datum(tok.Value == "true"), nil
	case tokenNull:
		p.advance()
		return reql.Datum(nil), nil
	default:
		return reql.Term{}, fmt.Errorf("unexpected token %q at position %d", tok.Value, tok.Pos)
	}
}

// parseNumberValue converts a number string to int or float64.
func parseNumberValue(s string) (interface{}, error) {
	for _, c := range s {
		if c == '.' || c == 'e' || c == 'E' {
			return strconv.ParseFloat(s, 64)
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return int(n), nil
}
