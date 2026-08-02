package parser

import (
	"fmt"
	"strconv"
	"unicode"

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
	localsStack []map[string]reql.Term
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

// lookupLocal resolves a var/let/const binding, innermost scope first.
func (p *parser) lookupLocal(name string) (reql.Term, bool) {
	for i := len(p.localsStack) - 1; i >= 0; i-- {
		if t, ok := p.localsStack[i][name]; ok {
			return t, true
		}
	}
	return reql.Term{}, false
}

// setLocal binds name in the innermost scope, shadowing a parameter of the same name.
func (p *parser) setLocal(name string, t reql.Term) {
	p.localsStack[len(p.localsStack)-1][name] = t
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
	p.localsStack = append(p.localsStack, map[string]reql.Term{})
	return ids
}

// popScope removes the innermost scope. If the stack becomes empty, resets nextVarID.
func (p *parser) popScope() {
	if len(p.paramsStack) > 0 {
		p.paramsStack = p.paramsStack[:len(p.paramsStack)-1]
	}
	if len(p.localsStack) > 0 {
		p.localsStack = p.localsStack[:len(p.localsStack)-1]
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
	tokenPlus:      "'+'",
	tokenMinus:     "'-'",
	tokenStar:      "'*'",
	tokenSlash:     "'/'",
	tokenPercent:   "'%'",
	tokenAssign:    "'='",
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
	return p.parseAdditive()
}

// infixOps maps an operator token to the builder method it produces.
var infixOps = map[tokenType]func(reql.Term, interface{}) reql.Term{
	tokenPlus:    reql.Term.Add,
	tokenMinus:   reql.Term.Sub,
	tokenStar:    reql.Term.Mul,
	tokenSlash:   reql.Term.Div,
	tokenPercent: reql.Term.Mod,
}

var (
	additiveOps       = map[tokenType]bool{tokenPlus: true, tokenMinus: true}
	multiplicativeOps = map[tokenType]bool{tokenStar: true, tokenSlash: true, tokenPercent: true}
)

// parseBinary parses `next { op next }` left-associatively for the given operator set.
func (p *parser) parseBinary(ops map[tokenType]bool, next func() (reql.Term, error)) (reql.Term, error) {
	left, err := next()
	if err != nil {
		return reql.Term{}, err
	}
	for ops[p.peek().Type] {
		op := p.advance().Type
		right, err := next()
		if err != nil {
			return reql.Term{}, err
		}
		left = infixOps[op](left, right)
	}
	return left, nil
}

func (p *parser) parseAdditive() (reql.Term, error) {
	return p.parseBinary(additiveOps, p.parseMultiplicative)
}

func (p *parser) parseMultiplicative() (reql.Term, error) {
	return p.parseBinary(multiplicativeOps, p.parsePostfix)
}

// parsePostfix parses a primary expression followed by its method and bracket chain.
func (p *parser) parsePostfix() (reql.Term, error) {
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
	// locals and params take priority over r.* dispatch when inside a lambda
	if p.inLambda() {
		if local, ok := p.lookupLocal(tok.Value); ok {
			p.advance()
			return local, nil
		}
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

// parseFunctionExpr parses function(params){ locals* return? body ;? } and returns a FUNC term.
// The "function" keyword has already been consumed by the caller.
func (p *parser) parseFunctionExpr() (reql.Term, error) {
	names, err := p.parseLambdaParams()
	if err != nil {
		return reql.Term{}, err
	}
	ids := p.pushScope(names)
	defer p.popScope()
	body, err := p.parseBlockBody()
	if err != nil {
		return reql.Term{}, err
	}
	return reql.Func(body, ids...), nil
}

// parseBlockBody parses { locals* return? body ;? } and returns the body term.
// The caller must have pushed the enclosing scope, so that parameters are visible
// to the local bindings.
func (p *parser) parseBlockBody() (reql.Term, error) {
	if _, err := p.expect(tokenLBrace); err != nil {
		return reql.Term{}, err
	}
	if err := p.parseLocalBindings(); err != nil {
		return reql.Term{}, err
	}
	// optional "return" keyword
	if p.peek().Type == tokenIdent && p.peek().Value == "return" {
		p.advance()
	}
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
	return body, nil
}

// localKeywords introduce a local binding statement inside a block body.
var localKeywords = map[string]bool{"var": true, "let": true, "const": true}

// reservedLocalNames cannot be bound by a local binding statement.
var reservedLocalNames = map[string]bool{
	"var": true, "let": true, "const": true, "return": true, "function": true,
	"true": true, "false": true, "null": true,
}

// parseLocalBindings parses `var|let|const <ident> = <expr> ;` statements into the
// innermost scope. The bound term is inlined at every use site, so referencing a
// local twice duplicates its subtree in the query.
func (p *parser) parseLocalBindings() error {
	for p.peek().Type == tokenIdent && localKeywords[p.peek().Value] {
		p.advance()
		name := p.peek()
		if err := validateLocalName(name); err != nil {
			return err
		}
		p.advance()
		if _, err := p.expect(tokenAssign); err != nil {
			return err
		}
		value, err := p.parseExpr()
		if err != nil {
			return err
		}
		if _, err := p.expect(tokenSemicolon); err != nil {
			return err
		}
		p.setLocal(name.Value, value)
	}
	return nil
}

// validateLocalName checks that tok can name a local binding.
func validateLocalName(tok token) error {
	if reservedLocalNames[tok.Value] {
		return fmt.Errorf("reserved word %q cannot be used as variable name at position %d", tok.Value, tok.Pos)
	}
	if tok.Type != tokenIdent {
		return fmt.Errorf("expected identifier in variable declaration, got %q at position %d", tok.Value, tok.Pos)
	}
	return nil
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
// An empty list is allowed: FUNC with no parameters is valid ReQL.
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

func parseRTableList(p *parser) (reql.Term, error) {
	if err := p.parseNoArgs(); err != nil {
		return reql.Term{}, err
	}
	return reql.TableList(), nil
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

// termsToIface converts a []reql.Term slice to []interface{} for variadic calls.
func termsToIface(args []reql.Term) []interface{} {
	out := make([]interface{}, len(args))
	for i, a := range args {
		out[i] = a
	}
	return out
}

func parseRObject(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args)%2 != 0 {
		return reql.Term{}, fmt.Errorf("r.object requires an even number of arguments (key-value pairs), got %d", len(args))
	}
	return reql.Object(termsToIface(args)...), nil
}

func parseRRange(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) > 2 {
		return reql.Term{}, fmt.Errorf("r.range accepts 0, 1, or 2 arguments, got %d", len(args))
	}
	return reql.Range(termsToIface(args)...), nil
}

func parseRRandom(p *parser) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	args, err := p.parseRandomArgs()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return reql.Random(args...), nil
}

func (p *parser) parseRandomArgs() ([]interface{}, error) {
	args, err := p.parseRandomNumericArgs()
	if err != nil {
		return nil, err
	}
	switch p.peek().Type {
	case tokenLBrace:
		opts, err := p.parseOptArgs()
		if err != nil {
			return nil, err
		}
		args = append(args, opts)
	case tokenRParen:
		// no opts
	default:
		return nil, fmt.Errorf("r.random accepts 0, 1, or 2 arguments at position %d", p.peek().Pos)
	}
	return args, nil
}

func (p *parser) parseRandomNumericArgs() ([]interface{}, error) {
	var args []interface{}
	for len(args) < 2 {
		next := p.peek().Type
		if next == tokenLBrace || next == tokenRParen {
			break
		}
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().Type != tokenComma {
			break
		}
		p.advance()
		if p.peek().Type == tokenRParen {
			return nil, fmt.Errorf("trailing comma in argument list at position %d", p.peek().Pos)
		}
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

// parseRDo parses r.do(arg1, ..., argN, fn).
// The last argument is the function; preceding arguments are data args.
// Wire format: [64, [fn, arg1, ..., argN]].
func parseRDo(p *parser) (reql.Term, error) {
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) == 0 {
		return reql.Term{}, fmt.Errorf("r.do requires at least a function argument")
	}
	return reql.Do(termsToIface(args)...), nil
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

// argsWithOpts converts parsed terms to a variadic argument list with the
// optional trailing OptArgs appended, as the variadic builders expect.
func argsWithOpts(args []reql.Term, opts reql.OptArgs) []interface{} {
	iargs := make([]interface{}, len(args))
	for i, a := range args {
		iargs[i] = a
	}
	if opts != nil {
		iargs = append(iargs, opts)
	}
	return iargs
}

func chainOrderBy(p *parser, t reql.Term) (reql.Term, error) {
	args, opts, err := p.parseArgListWithOpts()
	if err != nil {
		return reql.Term{}, err
	}
	return t.OrderBy(argsWithOpts(args, opts)...), nil
}

func chainGroup(p *parser, t reql.Term) (reql.Term, error) {
	pos := p.peek().Pos
	args, opts, err := p.parseArgListWithOpts()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) == 0 && opts == nil {
		return reql.Term{}, fmt.Errorf("group: requires at least one key or an optargs object at position %d", pos)
	}
	return t.Group(argsWithOpts(args, opts)...), nil
}

// chainAggregate builds min/max/sum/avg: no args, one field expression, an
// optional trailing OptArgs, or an opts-only form such as max({index:"d"}).
func chainAggregate(name string, build func(reql.Term, ...interface{}) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		pos := p.peek().Pos
		args, opts, err := p.parseArgListWithOpts()
		if err != nil {
			return reql.Term{}, err
		}
		if len(args) > 1 {
			return reql.Term{}, fmt.Errorf("%s: takes at most one field argument at position %d", name, pos)
		}
		return build(t, argsWithOpts(args, opts)...), nil
	}
}

func chainLimit(p *parser, t reql.Term) (reql.Term, error) {
	n, err := p.parseOneIntArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Limit(n), nil
}

func chainEqJoin(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	tok, err := p.expect(tokenString)
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	table, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenComma {
		p.advance()
		if p.peek().Type != tokenLBrace {
			return reql.Term{}, fmt.Errorf("eqJoin: third argument must be an optargs object at position %d", p.peek().Pos)
		}
		opts, err := p.parseOptArgs()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return t.EqJoin(tok.Value, table, opts), nil
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t.EqJoin(tok.Value, table), nil
}

func chainBetween(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	lower, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	upper, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenComma {
		p.advance()
		if p.peek().Type != tokenLBrace {
			return reql.Term{}, fmt.Errorf("between: third argument must be an optargs object at position %d", p.peek().Pos)
		}
		opts, err := p.parseOptArgs()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return t.Between(lower, upper, opts), nil
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t.Between(lower, upper), nil
}

func chainSlice(p *parser, t reql.Term) (reql.Term, error) {
	pos := p.peek().Pos
	bounds, err := p.parseIntArgs()
	if err != nil {
		return reql.Term{}, err
	}
	if len(bounds) == 0 || len(bounds) > 2 {
		return reql.Term{}, fmt.Errorf("slice requires 1 or 2 integer bounds at position %d, got %d", pos, len(bounds))
	}
	return t.Slice(bounds...), nil
}

// chainBranch parses .branch(val1, val2, ...) with the receiver as the condition.
// The receiver counts as the first BRANCH argument, so the branches must be even in number.
func chainBranch(p *parser, t reql.Term) (reql.Term, error) {
	pos := p.peek().Pos
	args, err := p.parseArgList()
	if err != nil {
		return reql.Term{}, err
	}
	if len(args) < 2 || len(args)%2 != 0 {
		return reql.Term{}, fmt.Errorf(
			"branch requires an even number of arguments (at least 2) at position %d, got %d", pos, len(args))
	}
	branches := make([]interface{}, len(args))
	for i, a := range args {
		branches[i] = a
	}
	return t.Branch(branches...), nil
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
	args, err := p.parseFieldSelectors()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Pluck(args...), nil
}

func chainWithout(p *parser, t reql.Term) (reql.Term, error) {
	args, err := p.parseFieldSelectors()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Without(args...), nil
}

func chainHasFields(p *parser, t reql.Term) (reql.Term, error) {
	args, err := p.parseFieldSelectors()
	if err != nil {
		return reql.Term{}, err
	}
	return t.HasFields(args...), nil
}

func chainWithFields(p *parser, t reql.Term) (reql.Term, error) {
	args, err := p.parseFieldSelectors()
	if err != nil {
		return reql.Term{}, err
	}
	return t.WithFields(args...), nil
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
	args, opts, err := p.parseArgListNoOptsOnly()
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
	if opts != nil {
		iargs = append(iargs, opts)
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

func chainFold(p *parser, t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	base, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, err
	}
	fn, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, err
	}
	if p.peek().Type == tokenComma {
		p.advance()
		opts, err := p.parseFoldOpts()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return t.Fold(base, fn, opts), nil
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	return t.Fold(base, fn), nil
}

// chainDo parses .do(fn) -- chain form of r.do.
// Equivalent to r.do(t, fn): applies fn to the current term.
func chainDo(p *parser, t reql.Term) (reql.Term, error) {
	fn, err := p.parseOneArg()
	if err != nil {
		return reql.Term{}, err
	}
	return t.Do(fn), nil
}

// parseFoldOpts parses {key: expr, ...} where values are full expressions (for lambdas in emit/finalEmit).
func (p *parser) parseFoldOpts() (reql.OptArgs, error) {
	return p.parseObjectBody(func() (interface{}, error) {
		v, err := p.parseExpr()
		return v, err
	})
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

// noArgChainWithOpts creates a chain builder for zero-argument methods that accept optional OptArgs.
func noArgChainWithOpts(fn func(reql.Term, ...reql.OptArgs) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		if _, err := p.expect(tokenLParen); err != nil {
			return reql.Term{}, err
		}
		if p.peek().Type == tokenRParen {
			p.advance()
			return fn(t), nil
		}
		if p.peek().Type != tokenLBrace {
			return reql.Term{}, fmt.Errorf("expected optargs object at position %d", p.peek().Pos)
		}
		opts, err := p.parseOptArgs()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return fn(t, opts), nil
	}
}

// oneArgChainWithOpts creates a chain builder for single-Term methods that accept optional OptArgs.
func oneArgChainWithOpts(fn func(reql.Term, reql.Term, ...reql.OptArgs) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		if _, err := p.expect(tokenLParen); err != nil {
			return reql.Term{}, err
		}
		arg, err := p.parseExpr()
		if err != nil {
			return reql.Term{}, err
		}
		if p.peek().Type == tokenComma {
			p.advance()
			if p.peek().Type != tokenLBrace {
				return reql.Term{}, fmt.Errorf("expected optargs object at position %d", p.peek().Pos)
			}
			opts, err := p.parseOptArgs()
			if err != nil {
				return reql.Term{}, err
			}
			if _, err := p.expect(tokenRParen); err != nil {
				return reql.Term{}, err
			}
			return fn(t, arg, opts), nil
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return fn(t, arg), nil
	}
}

// strArgChainWithOpts creates a chain builder for single-string methods that accept optional OptArgs.
func strArgChainWithOpts(fn func(reql.Term, string, ...reql.OptArgs) reql.Term) chainFn {
	return func(p *parser, t reql.Term) (reql.Term, error) {
		if _, err := p.expect(tokenLParen); err != nil {
			return reql.Term{}, err
		}
		tok, err := p.expect(tokenString)
		if err != nil {
			return reql.Term{}, err
		}
		if p.peek().Type == tokenComma {
			p.advance()
			if p.peek().Type != tokenLBrace {
				return reql.Term{}, fmt.Errorf("expected optargs object at position %d", p.peek().Pos)
			}
			opts, err := p.parseOptArgs()
			if err != nil {
				return reql.Term{}, err
			}
			if _, err := p.expect(tokenRParen); err != nil {
				return reql.Term{}, err
			}
			return fn(t, tok.Value, opts), nil
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return fn(t, tok.Value), nil
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
		"tableList": parseRTableList,
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
		"do":        parseRDo,
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
	m["branch"] = chainBranch
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
	m["info"] = noArgChain(func(t reql.Term) reql.Term { return t.Info() })
	m["offsetsOf"] = oneArgChain(func(t, pred reql.Term) reql.Term { return t.OffsetsOf(pred) })
	m["fold"] = chainFold
	m["do"] = chainDo
}

func registerFieldChain(m map[string]chainFn) {
	m["pluck"] = chainPluck
	m["without"] = chainWithout
	m["getField"] = oneArgChain(func(t, field reql.Term) reql.Term { return t.GetField(field) })
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
	m["group"] = chainGroup
	m["ungroup"] = noArgChain(func(t reql.Term) reql.Term { return t.Ungroup() })
	m["concatMap"] = oneArgChain(func(t, fn reql.Term) reql.Term { return t.ConcatMap(fn) })
	m["forEach"] = oneArgChain(func(t, fn reql.Term) reql.Term { return t.ForEach(fn) })
	m["sum"] = chainAggregate("sum", reql.Term.Sum)
	m["avg"] = chainAggregate("avg", reql.Term.Avg)
	m["min"] = chainAggregate("min", reql.Term.Min)
	m["max"] = chainAggregate("max", reql.Term.Max)
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
	m["bitAnd"] = oneArgChain(func(t, n reql.Term) reql.Term { return t.BitAnd(n) })
	m["bitOr"] = oneArgChain(func(t, n reql.Term) reql.Term { return t.BitOr(n) })
	m["bitXor"] = oneArgChain(func(t, n reql.Term) reql.Term { return t.BitXor(n) })
	m["bitNot"] = noArgChain(func(t reql.Term) reql.Term { return t.BitNot() })
	m["bitSal"] = oneArgChain(func(t, n reql.Term) reql.Term { return t.BitSal(n) })
	m["bitSar"] = oneArgChain(func(t, n reql.Term) reql.Term { return t.BitSar(n) })
}

func registerStringChain(m map[string]chainFn) {
	m["match"] = oneArgChain(func(t, re reql.Term) reql.Term { return t.Match(re) })
	m["split"] = chainSplit
	m["upcase"] = noArgChain(func(t reql.Term) reql.Term { return t.Upcase() })
	m["downcase"] = noArgChain(func(t reql.Term) reql.Term { return t.Downcase() })
	m["toJSONString"] = noArgChain(func(t reql.Term) reql.Term { return t.ToJSONString() })
	m["toJSON"] = noArgChain(func(t reql.Term) reql.Term { return t.ToJSONString() })
	m["toJsonString"] = noArgChain(func(t reql.Term) reql.Term { return t.ToJSONString() })
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
	m["tableCreate"] = strArgChainWithOpts(func(t reql.Term, s string, opts ...reql.OptArgs) reql.Term { return t.TableCreate(s, opts...) })
	m["tableDrop"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.TableDrop(s) })
	m["tableList"] = noArgChain(func(t reql.Term) reql.Term { return t.TableList() })
	m["indexCreate"] = strArgChainWithOpts(func(t reql.Term, s string, opts ...reql.OptArgs) reql.Term { return t.IndexCreate(s, opts...) })
	m["indexDrop"] = strArgChain(func(t reql.Term, s string) reql.Term { return t.IndexDrop(s) })
	m["indexList"] = noArgChain(func(t reql.Term) reql.Term { return t.IndexList() })
	m["indexWait"] = chainIndexWait
	m["indexStatus"] = chainIndexStatus
	m["indexRename"] = chainIndexRename
	m["changes"] = noArgChainWithOpts(func(t reql.Term, opts ...reql.OptArgs) reql.Term { return t.Changes(opts...) })
	m["config"] = noArgChain(func(t reql.Term) reql.Term { return t.Config() })
	m["status"] = noArgChain(func(t reql.Term) reql.Term { return t.Status() })
	m["sync"] = noArgChain(func(t reql.Term) reql.Term { return t.Sync() })
	m["reconfigure"] = noArgChainWithOpts(func(t reql.Term, opts ...reql.OptArgs) reql.Term { return t.Reconfigure(opts...) })
	m["rebalance"] = noArgChain(func(t reql.Term) reql.Term { return t.Rebalance() })
	m["wait"] = noArgChain(func(t reql.Term) reql.Term { return t.Wait() })
	m["grant"] = chainGrant
	m["toGeoJSON"] = noArgChain(func(t reql.Term) reql.Term { return t.ToGeoJSON() })
	m["distance"] = oneArgChainWithOpts(func(t, o reql.Term, opts ...reql.OptArgs) reql.Term { return t.Distance(o, opts...) })
	m["intersects"] = oneArgChain(func(t, o reql.Term) reql.Term { return t.Intersects(o) })
	m["includes"] = oneArgChain(func(t, pt reql.Term) reql.Term { return t.Includes(pt) })
	m["getIntersecting"] = oneArgChainWithOpts(func(t, geo reql.Term, opts ...reql.OptArgs) reql.Term { return t.GetIntersecting(geo, opts...) })
	m["getNearest"] = oneArgChainWithOpts(func(t, pt reql.Term, opts ...reql.OptArgs) reql.Term { return t.GetNearest(pt, opts...) })
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

// parseBracketArg parses term("field"), term(0) or term(expr) bracket notation.
// String arg -> Bracket(field); integer arg -> Nth(n); float -> error;
// anything else -> Bracket(expr), which covers lambda parameters used as field keys.
func (p *parser) parseBracketArg(t reql.Term) (reql.Term, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return reql.Term{}, err
	}
	tok := p.peek()
	switch tok.Type {
	case tokenString, tokenNumber:
		return p.parseBracketLiteral(t, tok)
	case tokenRParen, tokenBool, tokenNull:
		// a bool or null can name neither a field nor an index
		return reql.Term{}, fmt.Errorf("expected string, integer or expression in bracket notation at position %d", tok.Pos)
	default:
		field, err := p.parseExpr()
		if err != nil {
			return reql.Term{}, err
		}
		if _, err := p.expect(tokenRParen); err != nil {
			return reql.Term{}, err
		}
		return t.Bracket(field), nil
	}
}

// parseBracketLiteral consumes a string or number bracket key already peeked as tok.
func (p *parser) parseBracketLiteral(t reql.Term, tok token) (reql.Term, error) {
	p.advance()
	if _, err := p.expect(tokenRParen); err != nil {
		return reql.Term{}, err
	}
	if tok.Type == tokenString {
		return t.Bracket(tok.Value), nil
	}
	n, err := strconv.Atoi(tok.Value)
	if err != nil {
		return reql.Term{}, fmt.Errorf("bracket index must be an integer, got %q at position %d", tok.Value, tok.Pos)
	}
	return t.Nth(n), nil
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

// tryTrailingOptArgs attempts to parse '{...}' as trailing OptArgs when followed by ')'.
// Returns (opts, true) on success, or (nil, false) with the parser state restored on failure.
// Optarg values are full expressions, so a failed attempt can have parsed a lambda and
// advanced the scope state; pos, scope depth, var counter and nesting depth are all rolled back
// so that the re-parse allocates the same VAR ids.
func (p *parser) tryTrailingOptArgs() (reql.OptArgs, bool) {
	if p.peek().Type != tokenLBrace {
		return nil, false
	}
	savePos, saveScopes, saveVarID, saveDepth := p.pos, len(p.paramsStack), p.nextVarID, p.depth
	o, err := p.parseOptArgs()
	if err == nil && p.peek().Type == tokenRParen {
		return o, true
	}
	p.pos = savePos
	if len(p.paramsStack) > saveScopes {
		p.paramsStack = p.paramsStack[:saveScopes]
	}
	if len(p.localsStack) > saveScopes {
		p.localsStack = p.localsStack[:saveScopes]
	}
	p.nextVarID = saveVarID
	p.depth = saveDepth
	return nil, false
}

// parseArgAndSep parses one expression then the following separator.
// Returns (arg, opts, done, err):
//   - done=true with opts!=nil: trailing OptArgs was found, caller should consume RPAREN
//   - done=true with opts==nil: hit RPAREN/EOF, loop should end
//   - done=false: comma consumed, more args expected
func (p *parser) parseArgAndSep() (reql.Term, reql.OptArgs, bool, error) {
	arg, err := p.parseExpr()
	if err != nil {
		return reql.Term{}, nil, false, err
	}
	next := p.peek().Type
	if next == tokenEOF || next == tokenRParen {
		return arg, nil, true, nil
	}
	if _, err := p.expect(tokenComma); err != nil {
		return reql.Term{}, nil, false, err
	}
	if p.peek().Type == tokenRParen {
		return reql.Term{}, nil, false, fmt.Errorf("trailing comma in argument list at position %d", p.peek().Pos)
	}
	if opts, ok := p.tryTrailingOptArgs(); ok {
		return arg, opts, true, nil
	}
	return arg, nil, false, nil
}

// parseArgListBody parses arg expressions up to and including ')'.
// Called after '(' has already been consumed.
func (p *parser) parseArgListBody() ([]reql.Term, reql.OptArgs, error) {
	var args []reql.Term
	for p.peek().Type != tokenRParen && p.peek().Type != tokenEOF {
		arg, opts, done, err := p.parseArgAndSep()
		if err != nil {
			return nil, nil, err
		}
		args = append(args, arg)
		if done {
			if opts != nil {
				if _, err := p.expect(tokenRParen); err != nil {
					return nil, nil, err
				}
				return args, opts, nil
			}
			break
		}
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, nil, err
	}
	return args, nil, nil
}

// parseArgListWithOpts parses (arg1, ..., {opts}?) returning terms and optional trailing OptArgs.
// After consuming a comma, if '{' follows, attempts parseOptArgs; if succeeded and ')' follows,
// treats it as trailing OptArgs. Otherwise tryTrailingOptArgs rolls the parser state back and
// the object is re-parsed as a positional argument.
// Also handles opts-only case: ({opts}) with no positional args.
func (p *parser) parseArgListWithOpts() ([]reql.Term, reql.OptArgs, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, nil, err
	}
	// opts-only: ({key: val}) with no positional args; tryTrailingOptArgs verified ')' follows
	if opts, ok := p.tryTrailingOptArgs(); ok {
		p.advance()
		return nil, opts, nil
	}
	return p.parseArgListBody()
}

// parseArgListNoOptsOnly parses (arg1, ..., {opts}?) like parseArgListWithOpts but skips
// the opts-only path. Use when at least one positional argument is required (e.g. getAll).
func (p *parser) parseArgListNoOptsOnly() ([]reql.Term, reql.OptArgs, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, nil, err
	}
	return p.parseArgListBody()
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

// parseObjectBody parses {key: val, ...} using valueParser for each value.
func (p *parser) parseObjectBody(valueParser func() (interface{}, error)) (reql.OptArgs, error) {
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
		val, err := valueParser()
		if err != nil {
			return nil, err
		}
		opts[camelToSnake(key)] = val
		if p.peek().Type == tokenComma {
			p.advance()
			if p.peek().Type == tokenRBrace {
				return nil, fmt.Errorf("trailing comma in opts at position %d", p.peek().Pos)
			}
		}
	}
	if _, err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}
	return opts, nil
}

// parseOptArgs parses {key: val, ...} into a reql.OptArgs.
func (p *parser) parseOptArgs() (reql.OptArgs, error) {
	return p.parseObjectBody(p.parseOptArgValue)
}

// parseOptArgValue parses one optarg value: a datum literal on the fast path,
// any expression otherwise (e.g. {index: r.desc("d")}). Terms marshal correctly
// because OptArgs is a map[string]interface{} and Term implements MarshalJSON.
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
	return p.parseExpr()
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

// parseFieldSelectors parses (arg, arg, ...) where each arg is a string literal
// or a {key: val, ...} object. Returns []interface{} for use with Pluck/Without/etc.
func (p *parser) parseFieldSelectors() ([]interface{}, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}
	var args []interface{}
	for p.peek().Type != tokenRParen && p.peek().Type != tokenEOF {
		v, err := p.parseOneFieldSelector()
		if err != nil {
			return nil, err
		}
		args = append(args, v)
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

// parseOneFieldSelector parses one pluck/without/hasFields/withFields argument:
// a string literal, a {...} object, a [...] array, or any expression (a lambda
// parameter holding the field name). Scalar literals other than strings can never
// name a field, so they stay an error.
func (p *parser) parseOneFieldSelector() (interface{}, error) {
	tok := p.peek()
	switch tok.Type {
	case tokenString:
		p.advance()
		return tok.Value, nil
	case tokenLBrace:
		return p.parseDatumObject()
	case tokenLBracket:
		return p.parseDatumArray()
	case tokenNumber, tokenBool, tokenNull:
		return nil, fmt.Errorf("expected string, object, array or expression in field selector at position %d, got %q", tok.Pos, tok.Value)
	default:
		return p.parseExpr()
	}
}

// parseDatumValue parses a JSON-like datum literal into a native Go value.
// Produces string, float64/int, bool, nil, map[string]interface{}, or reql.Term (MAKE_ARRAY for arrays).
// Arrays become reql.Term because RethinkDB interprets bare JSON arrays in term arg positions as terms.
func (p *parser) parseDatumValue() (interface{}, error) {
	p.depth++
	if p.depth > maxDepth {
		return nil, fmt.Errorf("datum too deeply nested (max depth %d)", maxDepth)
	}
	defer func() { p.depth-- }()
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
	case tokenLBracket:
		t, err := p.parseDatumArray()
		if err != nil {
			return nil, err
		}
		return t, nil
	case tokenLBrace:
		return p.parseDatumObject()
	default:
		return nil, fmt.Errorf("expected datum value at position %d, got %q", tok.Pos, tok.Value)
	}
}

// parseDatumArray parses [v, v, ...] into a reql.Array (MAKE_ARRAY) term.
// Arrays inside field selector objects must be MAKE_ARRAY terms, not plain JSON
// arrays, because RethinkDB interprets bare arrays in term arg positions as terms.
func (p *parser) parseDatumArray() (reql.Term, error) {
	if _, err := p.expect(tokenLBracket); err != nil {
		return reql.Term{}, err
	}
	var elems []interface{}
	for p.peek().Type != tokenRBracket && p.peek().Type != tokenEOF {
		v, err := p.parseDatumValue()
		if err != nil {
			return reql.Term{}, err
		}
		elems = append(elems, v)
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
	return reql.Array(elems...), nil
}

// parseDatumObject parses {key: val, ...} into map[string]interface{} with native Go values.
// Keys are preserved as-is (no camelToSnake conversion).
func (p *parser) parseDatumObject() (map[string]interface{}, error) {
	if _, err := p.expect(tokenLBrace); err != nil {
		return nil, err
	}
	obj := map[string]interface{}{}
	for p.peek().Type != tokenRBrace && p.peek().Type != tokenEOF {
		key, err := p.parseObjectKey()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokenColon); err != nil {
			return nil, err
		}
		val, err := p.parseDatumValue()
		if err != nil {
			return nil, err
		}
		obj[key] = val
		if p.peek().Type == tokenComma {
			p.advance()
			if p.peek().Type == tokenRBrace {
				return nil, fmt.Errorf("trailing comma in object at position %d", p.peek().Pos)
			}
		}
	}
	if _, err := p.expect(tokenRBrace); err != nil {
		return nil, err
	}
	return obj, nil
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

// parseIntArgs parses (n1, n2, ...) for methods taking a variable number of integers.
// The caller checks how many values are acceptable.
func (p *parser) parseIntArgs() ([]int, error) {
	if _, err := p.expect(tokenLParen); err != nil {
		return nil, err
	}
	var nums []int
	for p.peek().Type != tokenRParen && p.peek().Type != tokenEOF {
		n, err := p.expectIntArg()
		if err != nil {
			return nil, err
		}
		nums = append(nums, n)
		if p.peek().Type != tokenComma {
			break
		}
		p.advance()
		if p.peek().Type == tokenRParen {
			return nil, fmt.Errorf("trailing comma in argument list at position %d", p.peek().Pos)
		}
	}
	if _, err := p.expect(tokenRParen); err != nil {
		return nil, err
	}
	return nums, nil
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

// camelToSnake converts camelCase to snake_case (e.g. leftBound -> left_bound).
func camelToSnake(s string) string {
	if s == "" {
		return s
	}
	out := make([]rune, 0, len(s)+4)
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
		} else {
			out = append(out, r)
		}
	}
	return string(out)
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
