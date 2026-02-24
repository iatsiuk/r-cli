# Plan: Lambda Expressions in Parser

## Overview

Add arrow function syntax (`(x) => expr` and `(x, y) => expr`) to the ReQL string parser. This enables writing lambda predicates in `filter`, `map`, `reduce`, `forEach`, `concatMap`, `innerJoin`, `outerJoin`, and other methods that accept function arguments.

Currently the parser only supports `r.row` for implicit single-argument functions. Arrow syntax adds explicit parameter binding, which enables multi-argument functions (e.g., `reduce`, `innerJoin`) and clearer variable naming.

Syntax to support:
- `(x) => x('field').gt(10)` -- single-param arrow
- `(left, right) => left('a').add(right('a'))` -- multi-param arrow
- `x => x('field').gt(10)` -- single-param without parens

Arrow functions are parsed into `reql.Func(body, paramIDs...)` with `reql.Var(id)` references. Parameter IDs are assigned sequentially starting from 1.

Package: `internal/reql/parser`

Depends on: `14-parser`

Out of scope: `r.do` is not yet implemented in the parser; adding it is a separate task.

## Design Notes

Lexer: add `tokenArrow` for `=>`. Handle `=` in `next()` before `punctToken`/`readValue`: if followed by `>` produce `tokenArrow`, otherwise error.

Parser -- lambda detection in `parsePrimary`:
- For `(`: lightweight lookahead scan matching `LPAREN (token COMMA)* token? RPAREN ARROW`. The `token?` means zero or more tokens are accepted, so `() => 1` also matches (the error "at least one parameter required" is produced by `parseLambda`, not the lookahead). The scan accepts any token between parens (not just IDENT) so that `parseLambda` produces specific errors (e.g., `(false) => 1` -> "expected identifier"). If no match, fall through to existing behavior.
- For `ident` (not `r`): peek next token; if `ARROW`, parse as bare single-param lambda.

Parser -- parameter scope: `params map[string]int` field on parser struct. `parsePrimary` checks params before falling through to `parseDatumTerm`. Chaining on `Var` works naturally (`x('field')` -> `Var(1).Bracket("field")`). Cleanup via `defer` after body parsing.

Scoping rules:
- `r.row` inside arrow is an error. Check in `parseRRow`: if `p.params != nil`, error immediately. This is the only IMPLICIT_VAR creation point, so it covers all methods. Critical because Map/Reduce/ConcatMap/ForEach do NOT call `wrapImplicitVar`.
- Nested arrows not supported (max depth 1).
- Reserved parameter names (`r`, `true`, `false`, `null`) rejected.

Arrow body boundaries: `parseExpr` greedily consumes body including chains. Terminates at `,`/`)` inside argument lists. `chainFilter` uses `parseOneArg` so `filter` with optargs is not supported (pre-existing limitation).

Filter interaction: `wrapImplicitVar` finds no `IMPLICIT_VAR` in FUNC from arrow syntax -> returns unchanged. No double wrapping.

Wire format: `(x) => x('age').gt(21)` -> `[69,[[2,[1]],[21,[[170,[[10,[1]],"age"]],21]]]]` = `FUNC([MAKE_ARRAY([1])], GT(BRACKET(VAR(1), "age"), 21))`. All tests verify wire JSON via `MarshalJSON` roundtrip.

## Validation Commands
- `go test ./internal/reql/parser/... -race -count=1`
- `go test -tags integration ./internal/integration/... -race -count=1 -run TestLambda`
- `make build`

### Task 1: Lexer -- arrow token

- [x] Test: tokenize `=>` -> [ARROW] token
- [x] Test: tokenize `(x) => x` -> [LPAREN, IDENT:x, RPAREN, ARROW, IDENT:x]
- [x] Test: tokenize `=` alone -> error "unexpected character"
- [x] Test: tokenize `==` -> error (not supported)
- [x] Test: fuzz -- lexer does not panic with `=` in arbitrary positions
- [x] Implement: add `tokenArrow` type, handle `=` in lexer

### Task 2: Parser -- single-param arrow in parentheses

- [x] Test: parse `(x) => x('age').gt(21)` -> wire JSON `[69,[[2,[1]],[21,[[170,[[10,[1]],"age"]],21]]]]`
- [x] Test: parse `(x) => x.eq(5)` -> wire JSON with FUNC wrapping EQ(VAR(1), 5)
- [x] Test: parse `(x) => true` -> wire JSON with FUNC wrapping datum true
- [x] Test: parse `(r) => r('f')` -> error (reserved parameter name "r")
- [x] Test: parse `(false) => 1` -> error (expected identifier, got "false")
- [x] Test: parse `(null) => 1` -> error (expected identifier, got "null")
- [x] Test: parse `(x) =>` -> error (expected expression after `=>`, got EOF)
- [x] Implement: lookahead scan for `(params) =>` pattern, arrow parsing with param scope

### Task 3: Parser -- multi-param arrow

- [x] Test: parse `(a, b) => a.add(b)` -> wire JSON `[69,[[2,[1,2]],[24,[[10,[1]],[10,[2]]]]]]`
- [x] Test: parse `(a, b, c) => a.add(b).add(c)` -> wire JSON with 3-param FUNC
- [x] Test: parse `(x, x) => x` -> error "duplicate parameter name"
- [x] Test: parse `() => 1` -> error "at least one parameter required"
- [x] Test: parse `(a,) => a` -> error (trailing comma)
- [x] Implement: multi-param arrow parsing

### Task 4: Parser -- bare single-param arrow (no parens)

- [ ] Test: parse `x => x('field').gt(0)` -> same wire JSON as `(x) => x('field').gt(0)`
- [ ] Test: parse `r.table('t').filter(x => x('age').gt(21))` -> FILTER with FUNC
- [ ] Test: bare ident without `=>` after it -> falls through to existing behavior (error for unknown ident)
- [ ] Implement: bare identifier arrow detection via peek at next token

### Task 5: Scoping and nesting rules

- [ ] Test: nested arrow `(x) => (y) => y` -> error "nested arrow functions not supported"
- [ ] Test: `r.row` inside arrow `(x) => r.row('f')` -> error "r.row inside arrow function is ambiguous"
- [ ] Test: parameter name used consistently in body -- `(x) => x('a').add(x('b')).mul(2)` -> multiple VAR(1) refs in wire JSON
- [ ] Test: body with chain methods on param -- `(doc) => doc('name').upcase().match('^A')` -> chained terms on VAR(1)
- [ ] Test: unknown identifier in body -- `(x) => y` -> error (verifies scope isolation: only declared params resolve to Var)
- [ ] Implement: parser `params` field (map[string]int), nesting guard, r.row conflict check in `parseRRow`

### Task 6: Arrow body boundaries and precedence

- [ ] Test: `r.table('t').filter((x) => x('a').gt(1))` -> body is entire `x('a').gt(1)`, single FUNC in FILTER
- [ ] Test: `r.branch((x) => x('ok'), "yes", "no")` -> arrow body is `x('ok')`, remaining args are branch args
- [ ] Test: filter with arrow does not double-wrap -- parse + MarshalJSON -> exactly one FUNC(69) in output
- [ ] Test: `r.table('t').map((x) => x('price').mul(x('qty')))` -> MAP with FUNC (no wrapImplicitVar needed)
- [ ] Implement: verify parseExpr terminates arrow body correctly at delimiters

### Task 7: Integration with chain methods

- [ ] Test: `.filter((doc) => doc('status').eq('active').and(doc('age').gt(18)))` -> wire JSON with compound predicate FUNC
- [ ] Test: `.reduce((a, b) => a.add(b))` -> wire JSON with 2-param FUNC in REDUCE
- [ ] Test: `.concatMap((x) => x('items'))` -> wire JSON with FUNC in CONCAT_MAP
- [ ] Test: `.forEach((x) => x('src').add('_copy'))` -> wire JSON with FUNC in FOR_EACH
- [ ] Test: `.innerJoin(r.table('b'), (left, right) => left('id').eq(right('id')))` -> wire JSON with 2-param FUNC in INNER_JOIN
- [ ] Test: `.outerJoin(r.table('b'), (a, b) => a('k').eq(b('k')))` -> wire JSON with 2-param FUNC in OUTER_JOIN
- [ ] Implement: no parser changes expected; arrow is parsed by parseExpr which chain methods already call

### Task 8: Fuzz testing

- [ ] Test: fuzz parser with arrow-containing seeds does not panic
- [ ] Seed corpus: `(x) => x`, `(a,b) => a.add(b)`, `x => x`, `() => 1`, `(x) => (y) => y`, `=> x`, `(x) =>`
- [ ] Implement: extend `FuzzParse` seed corpus with lambda patterns

### Task 9: Integration tests (live RethinkDB)

Build tag: `//go:build integration`. Package: `internal/integration`.

- [ ] Test: `filter((x) => x('age').gt(N))` on table with known data -> correct filtered results
- [ ] Test: `reduce((a, b) => a('val').add(b('val')))` -> correct sum
- [ ] Test: `map((x) => x('name').upcase())` -> correct transformed results
- [ ] Test: `innerJoin` with 2-param arrow on two tables -> correct joined docs
- [ ] Test: CLI binary: `r-cli query 'r.table("t").filter((x) => x("age").gt(21))'` -> valid JSON output with expected rows
- [ ] Implement: integration test functions using existing test helpers (setupTestDB, createTestTable, newExecutor)
