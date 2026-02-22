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

// rBuilderFn is the signature for r.* expression parsers.
type rBuilderFn = func(*parser) (reql.Term, error)

// chainFn is the signature for chain method parsers.
type chainFn = func(*parser, reql.Term) (reql.Term, error)

// rBuilders maps r.method names to builder functions.
var rBuilders map[string]rBuilderFn

// chainBuilders maps chained method names to builder functions.
var chainBuilders map[string]chainFn

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

// ---- rBuilder implementations ----

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

func parseRMinVal(_ *parser) (reql.Term, error) { return reql.MinVal(), nil }
func parseRMaxVal(_ *parser) (reql.Term, error) { return reql.MaxVal(), nil }

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

func parseRExprFn(p *parser) (reql.Term, error) { return p.parseOneArg() }

func parseRTable(p *parser) (reql.Term, error) {
	name, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Table(name), nil
}

func parseRDBCreate(p *parser) (reql.Term, error) {
	name, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.DBCreate(name), nil
}

func parseRDBDrop(p *parser) (reql.Term, error) {
	name, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.DBDrop(name), nil
}

func parseRDBList(p *parser) (reql.Term, error) {
	if p.peek().Type == tokenLParen {
		if err := p.parseNoArgs(); err != nil {
			return reql.Term{}, err
		}
	}
	return reql.DBList(), nil
}

func parseRNow(p *parser) (reql.Term, error) {
	if p.peek().Type == tokenLParen {
		if err := p.parseNoArgs(); err != nil {
			return reql.Term{}, err
		}
	}
	return reql.Now(), nil
}

func parseRUUID(p *parser) (reql.Term, error) {
	if p.peek().Type == tokenLParen {
		if err := p.parseNoArgs(); err != nil {
			return reql.Term{}, err
		}
	}
	return reql.UUID(), nil
}

func parseRJSON(p *parser) (reql.Term, error) {
	s, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.JSON(s), nil
}

func parseRISO8601(p *parser) (reql.Term, error) {
	s, err := p.parseOneStringArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.ISO8601(s), nil
}

func parseREpochTime(p *parser) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.EpochTime(arg), nil
}

func parseRLiteral(p *parser) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Literal(arg), nil
}

func parseRPoint(p *parser) (reql.Term, error) {
	lon, lat, err := p.parseTwoFloatArgs()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Point(lon, lat), nil
}

func parseRGeoJSON(p *parser) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.GeoJSON(arg), nil
}

// ---- Chain builder: specific implementations ----

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

func chainEqJoin(p *parser, t reql.Term) (reql.Term, error) {
	field, table, err := p.parseStringThenArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.EqJoin(field, table), nil
}

func chainBetween(p *parser, t reql.Term) (reql.Term, error) {
	lower, upper, err := p.parseTwoArgs()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Between(lower, upper), nil
}

func chainSlice(p *parser, t reql.Term) (reql.Term, error) {
	start, end, err := p.parseTwoInts()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Slice(start, end), nil
}

func chainIndexRename(p *parser, t reql.Term) (reql.Term, error) {
	oldName, newName, err := p.parseTwoStringArgs()
	if err != nil {
		return reql.Term{}, err
	}
	return t.IndexRename(oldName, newName), nil
}

func chainDuring(p *parser, t reql.Term) (reql.Term, error) {
	start, end, err := p.parseTwoArgs()
	if err != nil {
		return reql.Term{}, err
	}
	return t.During(start, end), nil
}

func chainPluck(p *parser, t reql.Term) (reql.Term, error) {
	strs, err := p.parseStringList()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Pluck(strs...), nil
}

func chainWithout(p *parser, t reql.Term) (reql.Term, error) {
	strs, err := p.parseStringList()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Without(strs...), nil
}

func chainHasFields(p *parser, t reql.Term) (reql.Term, error) {
	strs, err := p.parseStringList()
	if err != nil {
		return reql.Term{}, err
	}
	return t.HasFields(strs...), nil
}

func chainWithFields(p *parser, t reql.Term) (reql.Term, error) {
	strs, err := p.parseStringList()
	if err != nil {
		return reql.Term{}, err
	}
	return t.WithFields(strs...), nil
}

func chainIndexWait(p *parser, t reql.Term) (reql.Term, error) {
	strs, err := p.parseStringList()
	if err != nil {
		return reql.Term{}, err
	}
	return t.IndexWait(strs...), nil
}

func chainIndexStatus(p *parser, t reql.Term) (reql.Term, error) {
	strs, err := p.parseStringList()
	if err != nil {
		return reql.Term{}, err
	}
	return t.IndexStatus(strs...), nil
}

func chainGetAll(p *parser, t reql.Term) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	iargs := make([]interface{}, len(args))
	for i, a := range args {
		iargs[i] = a
	}
	return t.GetAll(iargs...), nil
}

func chainUnion(p *parser, t reql.Term) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Union(args...), nil
}

func chainContains(p *parser, t reql.Term) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	iargs := make([]interface{}, len(args))
	for i, a := range args {
		iargs[i] = a
	}
	return t.Contains(iargs...), nil
}

func chainSplit(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenRParen {
		p.advance()
		return t.Split(), nil
	}
	tok, err := p.expect(tokenString)
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t.Split(tok.Value), nil
}

func chainInsertAt(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	ntok, err := p.expect(tokenNumber)
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	val, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	n, err := strconv.Atoi(ntok.Value)
	if err != nil {
		return reql.Term{}, fmt.Errorf("expected integer, got %q", ntok.Value)
	}
	return t.InsertAt(n, val), nil
}

func chainDeleteAt(p *parser, t reql.Term) (reql.Term, error) {
	n, err := p.parseOneIntArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.DeleteAt(n), nil
}

func chainChangeAt(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	ntok, err := p.expect(tokenNumber)
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	val, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	n, err := strconv.Atoi(ntok.Value)
	if err != nil {
		return reql.Term{}, fmt.Errorf("expected integer, got %q", ntok.Value)
	}
	return t.ChangeAt(n, val), nil
}

func chainSpliceAt(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	ntok, err := p.expect(tokenNumber)
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	arr, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	n, err := strconv.Atoi(ntok.Value)
	if err != nil {
		return reql.Term{}, fmt.Errorf("expected integer, got %q", ntok.Value)
	}
	return t.SpliceAt(n, arr), nil
}

func chainInnerJoin(p *parser, t reql.Term) (reql.Term, error) {
	other, fn, err := p.parseTwoArgs()
	if err != nil {
		return reql.Term{}, err
	}
	return t.InnerJoin(other, fn), nil
}

func chainOuterJoin(p *parser, t reql.Term) (reql.Term, error) {
	other, fn, err := p.parseTwoArgs()
	if err != nil {
		return reql.Term{}, err
	}
	return t.OuterJoin(other, fn), nil
}

func chainGrant(p *parser, t reql.Term) (reql.Term, error) {
	user, perms, err := p.parseStringThenArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Grant(user, perms), nil
}

// ---- Generator helpers ----

// noArgChain creates a chain builder for zero-argument methods.
func noArgChain(fn func(reql.Term) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		if err := p.parseNoArgs(); err != nil {
			return reql.Term{}, err
		}
		return fn(t), nil
	}
}

// oneArgChain creates a chain builder for single-Term-argument methods.
func oneArgChain(fn func(reql.Term, reql.Term) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		arg, err := p.parseOneArg()
		if err != nil {
			return reql.Term{}, err
		}
		return fn(t, arg), nil
	}
}

// strArgChain creates a chain builder for single-string-argument methods.
func strArgChain(fn func(reql.Term, string) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		s, err := p.parseOneStringArg()
		if err != nil {
			return reql.Term{}, err
		}
		return fn(t, s), nil
	}
}

// intArgChain creates a chain builder for single-integer-argument methods.
func intArgChain(fn func(reql.Term, int) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		n, err := p.parseOneIntArg()
		if err != nil {
			return reql.Term{}, err
		}
		return fn(t, n), nil
	}
}

// ---- Registration ----

func init() {
	rBuilders = buildRBuilders()
	chainBuilders = buildChainBuilders()
}

func buildRBuilders() map[string]rBuilderFn {
	return map[string]rBuilderFn{
		"db":        parseRDB,
		"row":       parseRRow,
		"desc":      parseRDesc,
		"asc":       parseRAsc,
		"minval":    parseRMinVal,
		"maxval":    parseRMaxVal,
		"branch":    parseRBranch,
		"error":     parseRError,
		"args":      parseRArgs,
		"expr":      parseRExprFn,
		"table":     parseRTable,
		"dbCreate":  parseRDBCreate,
		"dbDrop":    parseRDBDrop,
		"dbList":    parseRDBList,
		"now":       parseRNow,
		"uuid":      parseRUUID,
		"json":      parseRJSON,
		"iso8601":   parseRISO8601,
		"epochTime": parseREpochTime,
		"literal":   parseRLiteral,
		"point":     parseRPoint,
		"geoJSON":   parseRGeoJSON,
	}
}

func buildChainBuilders() map[string]chainFn {
	m := make(map[string]chainFn)
	registerCoreChain(m)
	registerFieldChain(m)
	registerCompareChain(m)
	registerArithChain(m)
	registerStringChain(m)
	registerTimeChain(m)
	registerArrayChain(m)
	registerAdminChain(m)
	return m
}

func registerCoreChain(m map[string]chainFn) {
	m["table"] = chainTable
	m["filter"] = chainFilter
	m["get"] = chainGet
	m["getAll"] = chainGetAll
	m["insert"] = chainInsert
	m["update"] = oneArgChain(func(t, doc reql.Term) reql.Term { return t.Update(doc) })
	m["delete"] = noArgChain(func(t reql.Term) reql.Term { return t.Delete() })
	m["replace"] = oneArgChain(func(t, doc reql.Term) reql.Term { return t.Replace(doc) })
	m["between"] = chainBetween
	m["orderBy"] = chainOrderBy
	m["limit"] = chainLimit
	m["skip"] = intArgChain(func(t reql.Term, n int) reql.Term { return t.Skip(n) })
	m["count"] = noArgChain(func(t reql.Term) reql.Term { return t.Count() })
	m["distinct"] = noArgChain(func(t reql.Term) reql.Term { return t.Distinct() })
	m["union"] = chainUnion
	m["nth"] = intArgChain(func(t reql.Term, n int) reql.Term { return t.Nth(n) })
	m["isEmpty"] = noArgChain(func(t reql.Term) reql.Term { return t.IsEmpty() })
	m["contains"] = chainContains
	m["eqJoin"] = chainEqJoin
	m["innerJoin"] = chainInnerJoin
	m["outerJoin"] = chainOuterJoin
	m["zip"] = noArgChain(func(t reql.Term) reql.Term { return t.Zip() })
}

func registerFieldChain(m map[string]chainFn) {
	m["pluck"] = chainPluck
	m["without"] = chainWithout
	m["getField"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.GetField(s) })
	m["hasFields"] = chainHasFields
	m["merge"] = oneArgChain(func(t, obj reql.Term) reql.Term { return t.Merge(obj) })
	m["withFields"] = chainWithFields
	m["keys"] = noArgChain(func(t reql.Term) reql.Term { return t.Keys() })
	m["values"] = noArgChain(func(t reql.Term) reql.Term { return t.Values() })
	m["typeOf"] = noArgChain(func(t reql.Term) reql.Term { return t.TypeOf() })
	m["coerceTo"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.CoerceTo(s) })
	m["default"] = oneArgChain(func(t, val reql.Term) reql.Term { return t.Default(val) })
	m["map"] = oneArgChain(func(t, fn reql.Term) reql.Term { return t.Map(fn) })
	m["reduce"] = oneArgChain(func(t, fn reql.Term) reql.Term { return t.Reduce(fn) })
	m["group"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.Group(s) })
	m["ungroup"] = noArgChain(func(t reql.Term) reql.Term { return t.Ungroup() })
	m["concatMap"] = oneArgChain(func(t, fn reql.Term) reql.Term { return t.ConcatMap(fn) })
	m["forEach"] = oneArgChain(func(t, fn reql.Term) reql.Term { return t.ForEach(fn) })
	m["sum"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.Sum(s) })
	m["avg"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.Avg(s) })
	m["min"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.Min(s) })
	m["max"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.Max(s) })
}

func registerCompareChain(m map[string]chainFn) {
	m["eq"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Eq(v) })
	m["ne"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Ne(v) })
	m["lt"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Lt(v) })
	m["le"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Le(v) })
	m["gt"] = chainGt
	m["ge"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Ge(v) })
	m["not"] = noArgChain(func(t reql.Term) reql.Term { return t.Not() })
	m["and"] = oneArgChain(func(t, other reql.Term) reql.Term { return t.And(other) })
	m["or"] = oneArgChain(func(t, other reql.Term) reql.Term { return t.Or(other) })
}

func registerArithChain(m map[string]chainFn) {
	m["add"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Add(v) })
	m["sub"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Sub(v) })
	m["mul"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Mul(v) })
	m["div"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Div(v) })
	m["mod"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Mod(v) })
	m["floor"] = noArgChain(func(t reql.Term) reql.Term { return t.Floor() })
	m["ceil"] = noArgChain(func(t reql.Term) reql.Term { return t.Ceil() })
	m["round"] = noArgChain(func(t reql.Term) reql.Term { return t.Round() })
}

func registerStringChain(m map[string]chainFn) {
	m["match"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.Match(s) })
	m["split"] = chainSplit
	m["upcase"] = noArgChain(func(t reql.Term) reql.Term { return t.Upcase() })
	m["downcase"] = noArgChain(func(t reql.Term) reql.Term { return t.Downcase() })
	m["toJSONString"] = noArgChain(func(t reql.Term) reql.Term { return t.ToJSONString() })
	m["toISO8601"] = noArgChain(func(t reql.Term) reql.Term { return t.ToISO8601() })
	m["toEpochTime"] = noArgChain(func(t reql.Term) reql.Term { return t.ToEpochTime() })
}

func registerTimeChain(m map[string]chainFn) {
	m["date"] = noArgChain(func(t reql.Term) reql.Term { return t.Date() })
	m["timeOfDay"] = noArgChain(func(t reql.Term) reql.Term { return t.TimeOfDay() })
	m["timezone"] = noArgChain(func(t reql.Term) reql.Term { return t.Timezone() })
	m["year"] = noArgChain(func(t reql.Term) reql.Term { return t.Year() })
	m["month"] = noArgChain(func(t reql.Term) reql.Term { return t.Month() })
	m["day"] = noArgChain(func(t reql.Term) reql.Term { return t.Day() })
	m["dayOfWeek"] = noArgChain(func(t reql.Term) reql.Term { return t.DayOfWeek() })
	m["dayOfYear"] = noArgChain(func(t reql.Term) reql.Term { return t.DayOfYear() })
	m["hours"] = noArgChain(func(t reql.Term) reql.Term { return t.Hours() })
	m["minutes"] = noArgChain(func(t reql.Term) reql.Term { return t.Minutes() })
	m["seconds"] = noArgChain(func(t reql.Term) reql.Term { return t.Seconds() })
	m["inTimezone"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.InTimezone(s) })
	m["during"] = chainDuring
}

func registerArrayChain(m map[string]chainFn) {
	m["append"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Append(v) })
	m["prepend"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Prepend(v) })
	m["slice"] = chainSlice
	m["difference"] = oneArgChain(func(t, other reql.Term) reql.Term { return t.Difference(other) })
	m["insertAt"] = chainInsertAt
	m["deleteAt"] = chainDeleteAt
	m["changeAt"] = chainChangeAt
	m["spliceAt"] = chainSpliceAt
	m["setInsert"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.SetInsert(v) })
	m["setIntersection"] = oneArgChain(func(t, o reql.Term) reql.Term { return t.SetIntersection(o) })
	m["setUnion"] = oneArgChain(func(t, o reql.Term) reql.Term { return t.SetUnion(o) })
	m["setDifference"] = oneArgChain(func(t, o reql.Term) reql.Term { return t.SetDifference(o) })
}

func registerAdminChain(m map[string]chainFn) {
	m["tableCreate"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.TableCreate(s) })
	m["tableDrop"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.TableDrop(s) })
	m["tableList"] = noArgChain(func(t reql.Term) reql.Term { return t.TableList() })
	m["indexCreate"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.IndexCreate(s) })
	m["indexDrop"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.IndexDrop(s) })
	m["indexList"] = noArgChain(func(t reql.Term) reql.Term { return t.IndexList() })
	m["indexWait"] = chainIndexWait
	m["indexStatus"] = chainIndexStatus
	m["indexRename"] = chainIndexRename
	m["changes"] = noArgChain(func(t reql.Term) reql.Term { return t.Changes() })
	m["config"] = noArgChain(func(t reql.Term) reql.Term { return t.Config() })
	m["status"] = noArgChain(func(t reql.Term) reql.Term { return t.Status() })
	m["sync"] = noArgChain(func(t reql.Term) reql.Term { return t.Sync() })
	m["reconfigure"] = noArgChain(func(t reql.Term) reql.Term { return t.Reconfigure() })
	m["rebalance"] = noArgChain(func(t reql.Term) reql.Term { return t.Rebalance() })
	m["wait"] = noArgChain(func(t reql.Term) reql.Term { return t.Wait() })
	m["grant"] = chainGrant
	m["toGeoJSON"] = noArgChain(func(t reql.Term) reql.Term { return t.ToGeoJSON() })
	m["distance"] = oneArgChain(func(t, o reql.Term) reql.Term { return t.Distance(o) })
	m["intersects"] = oneArgChain(func(t, o reql.Term) reql.Term { return t.Intersects(o) })
	m["includes"] = oneArgChain(func(t, pt reql.Term) reql.Term { return t.Includes(pt) })
	m["getIntersecting"] = oneArgChain(func(t, geo reql.Term) reql.Term { return t.GetIntersecting(geo) })
	m["getNearest"] = oneArgChain(func(t, pt reql.Term) reql.Term { return t.GetNearest(pt) })
	m["fill"] = noArgChain(func(t reql.Term) reql.Term { return t.Fill() })
	m["polygonSub"] = oneArgChain(func(t, o reql.Term) reql.Term { return t.PolygonSub(o) })
}

// ---- Parser helper methods ----

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

// parseNoArgs expects () with no arguments.
func (p *parser) parseNoArgs() error {
	if _, err := p.expect(tokenLParen); err != nil {
		return err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return err
	}
	return nil
}

// parseStringList parses ("s1", "s2", ...) and returns the string values.
func (p *parser) parseStringList() ([]string, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}
	var strs []string
	for p.peek().Type != tokenRParen && p.peek().Type != tokenEOF {
		tok, err := p.expect(tokenString)
		if err != nil {
			return nil, err
		}
		strs = append(strs, tok.Value)
		if p.peek().Type == tokenComma {
			p.advance()
		}
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}
	return strs, nil
}

// parseTwoArgs parses (expr1, expr2) and returns both terms.
func (p *parser) parseTwoArgs() (first, second reql.Term, err error) {
	_, err = p.expect(tokenLParen)
	if err != nil {
		return reql.Term{}, reql.Term{}, err
	}
	first, err = p.parseExpr()
	if err != nil {
		return reql.Term{}, reql.Term{}, err
	}
	_, err = p.expect(tokenComma)
	if err != nil {
		return reql.Term{}, reql.Term{}, err
	}
	second, err = p.parseExpr()
	if err != nil {
		return reql.Term{}, reql.Term{}, err
	}
	_, err = p.expect(tokenRParen)
	if err != nil {
		return reql.Term{}, reql.Term{}, err
	}
	return first, second, nil
}

// parseTwoStringArgs parses ("s1", "s2") and returns both strings.
func (p *parser) parseTwoStringArgs() (s1, s2 string, err error) {
	var tok1, tok2 Token
	_, err = p.expect(tokenLParen)
	if err != nil {
		return "", "", err
	}
	tok1, err = p.expect(tokenString)
	if err != nil {
		return "", "", err
	}
	_, err = p.expect(tokenComma)
	if err != nil {
		return "", "", err
	}
	tok2, err = p.expect(tokenString)
	if err != nil {
		return "", "", err
	}
	_, err = p.expect(tokenRParen)
	if err != nil {
		return "", "", err
	}
	return tok1.Value, tok2.Value, nil
}

// parseStringThenArg parses ("str", expr) for methods like eqJoin.
func (p *parser) parseStringThenArg() (string, reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return "", reql.Term{}, err
	}
	tok, err := p.expect(tokenString)
	if err != nil {
		return "", reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return "", reql.Term{}, err
	}
	t, err := p.parseExpr()
	if err != nil {
		return "", reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return "", reql.Term{}, err
	}
	return tok.Value, t, nil
}

// parseTwoInts parses (n1, n2) for methods like slice.
func (p *parser) parseTwoInts() (n1, n2 int, err error) {
	var tok1, tok2 Token
	_, err = p.expect(tokenLParen)
	if err != nil {
		return 0, 0, err
	}
	tok1, err = p.expect(tokenNumber)
	if err != nil {
		return 0, 0, err
	}
	_, err = p.expect(tokenComma)
	if err != nil {
		return 0, 0, err
	}
	tok2, err = p.expect(tokenNumber)
	if err != nil {
		return 0, 0, err
	}
	_, err = p.expect(tokenRParen)
	if err != nil {
		return 0, 0, err
	}
	n1, err = strconv.Atoi(tok1.Value)
	if err != nil {
		return n1, n2, fmt.Errorf("expected integer, got %q", tok1.Value)
	}
	n2, err = strconv.Atoi(tok2.Value)
	if err != nil {
		return n1, n2, fmt.Errorf("expected integer, got %q", tok2.Value)
	}
	return n1, n2, nil
}

// parseTwoFloatArgs parses (f1, f2) for r.point.
func (p *parser) parseTwoFloatArgs() (v1, v2 float64, err error) {
	var tok1, tok2 Token
	_, err = p.expect(tokenLParen)
	if err != nil {
		return 0, 0, err
	}
	tok1, err = p.expect(tokenNumber)
	if err != nil {
		return 0, 0, err
	}
	_, err = p.expect(tokenComma)
	if err != nil {
		return 0, 0, err
	}
	tok2, err = p.expect(tokenNumber)
	if err != nil {
		return 0, 0, err
	}
	_, err = p.expect(tokenRParen)
	if err != nil {
		return 0, 0, err
	}
	v1, err = strconv.ParseFloat(tok1.Value, 64)
	if err != nil {
		return v1, v2, fmt.Errorf("invalid number %q: %w", tok1.Value, err)
	}
	v2, err = strconv.ParseFloat(tok2.Value, 64)
	if err != nil {
		return v1, v2, fmt.Errorf("invalid number %q: %w", tok2.Value, err)
	}
	return v1, v2, nil
}

// ---- Object / array / datum parsers ----

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
