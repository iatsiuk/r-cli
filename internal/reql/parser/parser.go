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

const maxDepth = 256

type parser struct {
	tokens      []token
	pos         int
	depth       int
	paramsStack []map[string]int
	nextVarID   int
}

func (p *parser) inLambda() bool {
	return len(p.paramsStack) > 0
}

func (p *parser) lookupParam(name string) (int, bool) {
	for i := len(p.paramsStack) - 1; i >= 0; i-- {
		if id, ok := p.paramsStack[i][name]; ok {
			return id, true
		}
	}
	return 0, false
}

// pushScope allocates IDs for names, pushes a new scope, and returns the IDs.
// When the stack is empty (top-level lambda), IDs restart from 1 for backward compat.
// When nested, IDs continue from nextVarID+1 to avoid collisions.
func (p *parser) pushScope(names []string) []int {
	if len(p.paramsStack) == 0 {
		p.nextVarID = 0
	}
	scope := make(map[string]int, len(names))
	ids := make([]int, len(names))
	for i, name := range names {
		p.nextVarID++
		scope[name] = p.nextVarID
		ids[i] = p.nextVarID
	}
	p.paramsStack = append(p.paramsStack, scope)
	return ids
}

// popScope removes the innermost scope. If the stack becomes empty, resets nextVarID.
func (p *parser) popScope() {
	if len(p.paramsStack) > 0 {
		p.paramsStack = p.paramsStack[:len(p.paramsStack)-1]
	}
	if len(p.paramsStack) == 0 {
		p.nextVarID = 0
	}
}

func (p *parser) peek() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return token{Type: tokenEOF}
}

func (p *parser) advance() token {
	tok := p.peek()
	if tok.Type != tokenEOF {
		p.pos++
	}
	return tok
}

var tokenNames = map[tokenType]string{
	tokenEOF:       "EOF",
	tokenIdent:     "identifier",
	tokenDot:       "'.'",
	tokenLParen:    "'('",
	tokenRParen:    "')'",
	tokenLBracket:  "'['",
	tokenRBracket:  "']'",
	tokenLBrace:    "'{'",
	tokenRBrace:    "'}'",
	tokenComma:     "','",
	tokenColon:     "':'",
	tokenString:    "string literal",
	tokenNumber:    "number",
	tokenBool:      "bool",
	tokenNull:      "null",
	tokenArrow:     "'=>'",
	tokenSemicolon: "';'",
}

func (p *parser) expect(tt tokenType) (token, error) {
	tok := p.peek()
	if tok.Type != tt {
		name := tokenNames[tt]
		return token{}, fmt.Errorf("expected %s, got %q at position %d", name, tok.Value, tok.Pos)
	}
	return p.advance(), nil
}

func (p *parser) parseExpr() (reql.Term, error) {
	p.depth++
	if p.depth > maxDepth {
		return reql.Term{}, fmt.Errorf("expression too deeply nested (max depth %d)", maxDepth)
	}
	defer func() { p.depth-- }()
	t, err := p.parsePrimary()
	if err != nil {
		return reql.Term{}, err
	}
	return p.parseChain(t)
}

func (p *parser) parsePrimary() (reql.Term, error) {
	tok := p.peek()
	switch {
	case tok.Type == tokenLParen && p.isLambdaAhead():
		return p.parseLambda()
	case tok.Type == tokenLParen:
		// grouped expression: ( expr )
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return expr, nil
	case tok.Type == tokenIdent:
		return p.parseIdentPrimary(tok)
	case tok.Type == tokenLBrace:
		return p.parseObjectTerm()
	case tok.Type == tokenLBracket:
		return p.parseArrayTerm()
	default:
		return p.parseDatumTerm()
	}
}

// parseIdentPrimary handles identifiers: r.* expressions, param vars, bare arrow lambdas, and datum fallback.
func (p *parser) parseIdentPrimary(tok token) (reql.Term, error) {
	// param lookup takes priority over r.* dispatch when inside a lambda
	if p.inLambda() {
		if id, ok := p.lookupParam(tok.Value); ok {
			p.advance()
			return reql.Var(id), nil
		}
	}
	// detect function(params){ ... } syntax
	if tok.Value == "function" && p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == tokenLParen {
		p.advance() // consume "function"
		return p.parseFunctionExpr()
	}
	// bare arrow check before r.* dispatch so that `r => ...` is valid
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == tokenArrow {
		return p.parseBareArrowLambda(tok)
	}
	if tok.Value == "r" {
		p.advance()
		return p.parseRExpr()
	}
	return p.parseDatumTerm()
}

// parseBareArrowLambda parses `ident => body` (no parentheses) and returns a single-param FUNC term.
func (p *parser) parseBareArrowLambda(tok token) (reql.Term, error) {
	if err := validateLambdaParam(tok, nil); err != nil {
		return reql.Term{}, err
	}
	p.advance() // consume ident
	if _, err := p.expect(tokenArrow); err != nil {
		return reql.Term{}, err
	}
	ids := p.pushScope([]string{tok.Value})
	defer p.popScope()
	body, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Func(body, ids...), nil
}

// parseFunctionExpr parses function(params){ return? body ;? } and returns a FUNC term.
// The "function" keyword has already been consumed by the caller.
func (p *parser) parseFunctionExpr() (reql.Term, error) {
	names, err := p.parseLambdaParams()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenLBrace); err != nil {
		return reql.Term{}, err
	}
	// optional "return" keyword
	if p.peek().Type == tokenIdent && p.peek().Value == "return" {
		p.advance()
	}
	ids := p.pushScope(names)
	defer p.popScope()
	body, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenSemicolon {
		p.advance()
	}
	if _, err := p.expect(tokenRBrace); err != nil {
		return reql.Term{}, err
	}
	return reql.Func(body, ids...), nil
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
			// bracket notation: term("field") or term(0)
			var err error
			t, err = p.parseBracketArg(t)
			if err != nil {
				return reql.Term{}, err
			}
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
	if p.inLambda() {
		return reql.Term{}, fmt.Errorf("r.row inside arrow function is ambiguous; use the arrow parameter instead")
	}
	t := reql.Row()
	if p.peek().Type != tokenLParen {
		return t, nil
	}
	return p.parseBracketArg(t)
}

// isLambdaAhead reports whether the current position starts a lambda expression:
// LPAREN (token COMMA)* token? RPAREN ARROW
func (p *parser) isLambdaAhead() bool {
	i := p.pos
	if i >= len(p.tokens) || p.tokens[i].Type != tokenLParen {
		return false
	}
	i = p.skipLambdaParams(i + 1)
	if i >= len(p.tokens) || p.tokens[i].Type != tokenRParen {
		return false
	}
	i++
	return i < len(p.tokens) && p.tokens[i].Type == tokenArrow
}

// skipLambdaParams scans forward past any tokens that could form a parameter list,
// stopping before the closing RPAREN (or at EOF). Returns the new index.
func (p *parser) skipLambdaParams(i int) int {
	for i < len(p.tokens) && p.tokens[i].Type != tokenRParen && p.tokens[i].Type != tokenEOF {
		i++ // accept any token as potential parameter
		if i < len(p.tokens) && p.tokens[i].Type == tokenComma {
			i++ // skip comma
		} else {
			break
		}
	}
	return i
}

// parseLambda parses (param, ...) => body and returns a FUNC term.
// Top-level lambdas start IDs at 1; nested lambdas continue from the current nextVarID.
func (p *parser) parseLambda() (reql.Term, error) {
	names, err := p.parseLambdaParams()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenArrow); err != nil {
		return reql.Term{}, err
	}
	ids := p.pushScope(names)
	defer p.popScope()
	body, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Func(body, ids...), nil
}

// parseLambdaParams parses (ident, ...) and returns the parameter names.
// Validates identifiers, reserved names, and duplicates.
func (p *parser) parseLambdaParams() ([]string, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}
	var names []string
	for p.peek().Type != tokenRParen && p.peek().Type != tokenEOF {
		tok := p.peek()
		if err := validateLambdaParam(tok, names); err != nil {
			return nil, err
		}
		p.advance()
		names = append(names, tok.Value)
		if p.peek().Type == tokenComma {
			p.advance()
			if p.peek().Type == tokenRParen {
				return nil, fmt.Errorf("trailing comma in parameter list at position %d", p.peek().Pos)
			}
		} else {
			break
		}
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("lambda requires at least one parameter")
	}
	return names, nil
}

// validateLambdaParam checks that tok is a valid, non-duplicate parameter name.
func validateLambdaParam(tok token, seen []string) error {
	if tok.Type != tokenIdent {
		return fmt.Errorf("expected identifier in lambda parameter, got %q at position %d", tok.Value, tok.Pos)
	}
	if tok.Value == "return" || tok.Value == "function" {
		return fmt.Errorf("reserved word %q cannot be used as parameter name at position %d", tok.Value, tok.Pos)
	}
	for _, existing := range seen {
		if existing == tok.Value {
			return fmt.Errorf("duplicate parameter name %q at position %d", tok.Value, tok.Pos)
		}
	}
	return nil
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

func parseRMinVal(p *parser) (reql.Term, error) {
	if p.peek().Type == tokenLParen {
		if err := p.parseNoArgs(); err != nil {
			return reql.Term{}, err
		}
	}
	return reql.MinVal(), nil
}

func parseRMaxVal(p *parser) (reql.Term, error) {
	if p.peek().Type == tokenLParen {
		if err := p.parseNoArgs(); err != nil {
			return reql.Term{}, err
		}
	}
	return reql.MaxVal(), nil
}

func parseRBranch(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) < 3 || len(args)%2 == 0 {
		return reql.Term{}, fmt.Errorf("r.branch requires an odd number of arguments (at least 3), got %d", len(args))
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
	if err := p.parseNoArgs(); err != nil {
		return reql.Term{}, err
	}
	return reql.DBList(), nil
}

func parseRNow(p *parser) (reql.Term, error) {
	if err := p.parseNoArgs(); err != nil {
		return reql.Term{}, err
	}
	return reql.Now(), nil
}

func parseRUUID(p *parser) (reql.Term, error) {
	if err := p.parseNoArgs(); err != nil {
		return reql.Term{}, err
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

func parseRTime(p *parser) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	year, month, day, err := parseRTimeYMD(p)
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenNumber {
		return parseRTime7tail(p, year, month, day)
	}
	// 4-arg form: timezone string
	tzTok, err := p.expect(tokenString)
	if err != nil {
		return reql.Term{}, fmt.Errorf("r.time timezone: %w", err)
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return reql.Time(year, month, day, tzTok.Value), nil
}

// parseRTimeYMD parses year, month, day and trailing comma for r.time.
func parseRTimeYMD(p *parser) (year, month, day int, err error) {
	if year, err = p.expectIntArg(); err != nil {
		return 0, 0, 0, fmt.Errorf("r.time year: %w", err)
	}
	if _, err = p.expect(tokenComma); err != nil {
		return 0, 0, 0, err
	}
	if month, err = p.expectIntArg(); err != nil {
		return 0, 0, 0, fmt.Errorf("r.time month: %w", err)
	}
	if _, err = p.expect(tokenComma); err != nil {
		return 0, 0, 0, err
	}
	if day, err = p.expectIntArg(); err != nil {
		return 0, 0, 0, fmt.Errorf("r.time day: %w", err)
	}
	if _, err = p.expect(tokenComma); err != nil {
		return 0, 0, 0, err
	}
	return year, month, day, nil
}

// parseRTime7tail parses hour, minute, second, timezone for the 7-arg r.time form.
func parseRTime7tail(p *parser, year, month, day int) (reql.Term, error) {
	hour, err := p.expectIntArg()
	if err != nil {
		return reql.Term{}, fmt.Errorf("r.time hour: %w", err)
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	minute, err := p.expectIntArg()
	if err != nil {
		return reql.Term{}, fmt.Errorf("r.time minute: %w", err)
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	second, err := p.expectIntArg()
	if err != nil {
		return reql.Term{}, fmt.Errorf("r.time second: %w", err)
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	tzTok, err := p.expect(tokenString)
	if err != nil {
		return reql.Term{}, fmt.Errorf("r.time timezone: %w", err)
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return reql.TimeAt(year, month, day, hour, minute, second, tzTok.Value), nil
}

func parseRBinary(p *parser) (reql.Term, error) {
	arg, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Binary(arg), nil
}

func parseRObject(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args)%2 != 0 {
		return reql.Term{}, fmt.Errorf("r.object requires an even number of arguments (key-value pairs), got %d", len(args))
	}
	pairs := make([]interface{}, len(args))
	for i, a := range args {
		pairs[i] = a
	}
	return reql.Object(pairs...), nil
}

func parseRRange(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) > 2 {
		return reql.Term{}, fmt.Errorf("r.range accepts 0, 1, or 2 arguments, got %d", len(args))
	}
	pairs := make([]interface{}, len(args))
	for i, a := range args {
		pairs[i] = a
	}
	return reql.Range(pairs...), nil
}

func parseRRandom(p *parser) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenRParen {
		p.advance()
		return reql.Random(), nil
	}
	args, err := parseRandomBody(p)
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Random(args...), nil
}

// parseRandomBody parses the body of r.random(...): up to 2 numeric args plus optional opts.
func parseRandomBody(p *parser) ([]interface{}, error) {
	var args []interface{}
	for len(args) < 2 && p.peek().Type != tokenLBrace && p.peek().Type != tokenRParen {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().Type != tokenComma {
			break
		}
		p.advance()
	}
	if p.peek().Type == tokenLBrace {
		opts, err := p.parseOptArgs()
		if err != nil {
			return nil, err
		}
		args = append(args, opts)
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}
	return args, nil
}

func parseRLine(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) < 2 {
		return reql.Term{}, fmt.Errorf("r.line requires at least 2 points, got %d", len(args))
	}
	return reql.Line(args...), nil
}

func parseRPolygon(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) < 3 {
		return reql.Term{}, fmt.Errorf("r.polygon requires at least 3 points, got %d", len(args))
	}
	return reql.Polygon(args...), nil
}

func parseRCircle(p *parser) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	center, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	radTok, err := p.expect(tokenNumber)
	if err != nil {
		return reql.Term{}, err
	}
	radius, err := strconv.ParseFloat(radTok.Value, 64)
	if err != nil {
		return reql.Term{}, fmt.Errorf("invalid radius %q: %w", radTok.Value, err)
	}
	if p.peek().Type == tokenComma {
		p.advance()
		opts, err := p.parseOptArgs()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return reql.Circle(center, radius, opts), nil
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return reql.Circle(center, radius), nil
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
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	doc, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenComma {
		p.advance()
		if p.peek().Type != tokenLBrace {
			return reql.Term{}, fmt.Errorf("insert: second argument must be an optargs object at position %d", p.peek().Pos)
		}
		opts, err := p.parseOptArgs()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return t.Insert(doc, opts), nil
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t.Insert(doc), nil
}

func chainUpdate(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	doc, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenComma {
		p.advance()
		if p.peek().Type != tokenLBrace {
			return reql.Term{}, fmt.Errorf("update: second argument must be an optargs object at position %d", p.peek().Pos)
		}
		opts, err := p.parseOptArgs()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return t.Update(doc, opts), nil
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t.Update(doc), nil
}

func chainDelete(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenRParen {
		p.advance()
		return t.Delete(), nil
	}
	if p.peek().Type != tokenLBrace {
		return reql.Term{}, fmt.Errorf("delete: argument must be an optargs object at position %d", p.peek().Pos)
	}
	opts, err := p.parseOptArgs()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t.Delete(opts), nil
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
	if len(args) == 0 {
		return reql.Term{}, fmt.Errorf("getAll requires at least one key")
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
	if len(args) == 0 {
		return reql.Term{}, fmt.Errorf("contains requires at least one value")
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
		"line":      parseRLine,
		"polygon":   parseRPolygon,
		"circle":    parseRCircle,
		"time":      parseRTime,
		"binary":    parseRBinary,
		"object":    parseRObject,
		"range":     parseRRange,
		"random":    parseRRandom,
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
	m["update"] = chainUpdate
	m["delete"] = chainDelete
	m["replace"] = oneArgChain(func(t, doc reql.Term) reql.Term { return t.Replace(doc) })
	m["between"] = chainBetween
	m["orderBy"] = chainOrderBy
	m["limit"] = chainLimit
	m["skip"] = intArgChain(func(t reql.Term, n int) reql.Term { return t.Skip(n) })
	m["count"] = noArgChain(func(t reql.Term) reql.Term { return t.Count() })
	m["distinct"] = noArgChain(func(t reql.Term) reql.Term { return t.Distinct() })
	m["union"] = chainUnion
	m["nth"] = intArgChain(func(t reql.Term, n int) reql.Term { return t.Nth(n) })
	m["sample"] = intArgChain(func(t reql.Term, n int) reql.Term { return t.Sample(n) })
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
	m["gt"] = oneArgChain(func(t, v reql.Term) reql.Term { return t.Gt(v) })
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

// parseBracketArg parses term("field") or term(0) bracket notation.
// String arg -> Bracket(field); integer arg -> Nth(n); float -> error.
func (p *parser) parseBracketArg(t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	tok := p.peek()
	switch tok.Type {
	case tokenString:
		p.advance()
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return t.Bracket(tok.Value), nil
	case tokenNumber:
		p.advance()
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		n, err := strconv.Atoi(tok.Value)
		if err != nil {
			return reql.Term{}, fmt.Errorf("bracket index must be an integer, got %q at position %d", tok.Value, tok.Pos)
		}
		return t.Nth(n), nil
	default:
		return reql.Term{}, fmt.Errorf("expected string or integer in bracket notation at position %d", tok.Pos)
	}
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

// expectIntArg parses a single tokenNumber token and converts it to int.
// Used internally when parsing structured arg lists (not wrapped in parens).
func (p *parser) expectIntArg() (int, error) {
	tok, err := p.expect(tokenNumber)
	if err != nil {
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
		if p.peek().Type == tokenEOF {
			break
		}
		if p.peek().Type != tokenRParen {
			if _, err := p.expect(tokenComma); err != nil {
				return nil, err
			}
			if p.peek().Type == tokenRParen {
				return nil, fmt.Errorf("trailing comma in argument list at position %d", p.peek().Pos)
			}
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

// parseOptArgs parses {key: val, ...} into a reql.OptArgs.
// Values are restricted to datum literals: string, number, bool, null.
func (p *parser) parseOptArgs() (reql.OptArgs, error) {
	if _, err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}
	opts := reql.OptArgs{}
	for p.peek().Type != tokenRBrace && p.peek().Type != tokenEOF {
		key, err := p.parseObjectKey()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenColon); err != nil {
			return nil, err
		}
		val, err := p.parseOptArgValue()
		if err != nil {
			return nil, err
		}
		opts[key] = val
		if p.peek().Type == tokenComma {
			p.advance()
			if p.peek().Type == tokenRBrace {
				return nil, fmt.Errorf("trailing comma in optargs at position %d", p.peek().Pos)
			}
		}
	}
	if _, err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}
	return opts, nil
}

func (p *parser) parseOptArgValue() (interface{}, error) {
	tok := p.peek()
	switch tok.Type {
	case tokenString:
		p.advance()
		return tok.Value, nil
	case tokenNumber:
		p.advance()
		return parseNumberValue(tok.Value)
	case tokenBool:
		p.advance()
		return tok.Value == "true", nil
	case tokenNull:
		p.advance()
		return nil, nil
	}
	return nil, fmt.Errorf("expected datum literal in optargs at position %d, got %q", tok.Pos, tok.Value)
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
		if p.peek().Type == tokenEOF {
			break
		}
		if p.peek().Type != tokenRParen {
			if _, err := p.expect(tokenComma); err != nil {
				return nil, err
			}
			if p.peek().Type == tokenRParen {
				return nil, fmt.Errorf("trailing comma in argument list at position %d", p.peek().Pos)
			}
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
	var tok1, tok2 token
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
	var tok1, tok2 token
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
	var tok1, tok2 token
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
			if p.peek().Type == tokenRBrace {
				return reql.Term{}, fmt.Errorf("trailing comma in object at position %d", p.peek().Pos)
			}
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
			if p.peek().Type == tokenRBracket {
				return reql.Term{}, fmt.Errorf("trailing comma in array at position %d", p.peek().Pos)
			}
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
	return n, nil
}
