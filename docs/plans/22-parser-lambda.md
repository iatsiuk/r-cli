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

## Validation Commands
- `go test ./internal/reql/parser/... -race -count=1`
- `go test -tags integration ./internal/integration/... -race -count=1 -run TestLambda`
- `make build`

## Design

### Lexer changes

Add one new token type: `tokenArrow` for `=>`. The lexer must handle `=` carefully:
- `=>` produces `tokenArrow`
- `=` alone is an error (not used in ReQL syntax)

This avoids conflicts with existing tokens since `=` is not currently recognized.

### Parser changes

Arrow expressions are parsed as primary expressions (in `parsePrimary`) when the parser sees a pattern that starts a lambda:
1. `(ident, ident, ...) =>` -- parenthesized parameter list followed by arrow
2. `ident =>` -- single bare identifier followed by arrow

#### Lambda detection strategy

The parser cannot use general backtracking because `(expr)` grouping is not supported -- there is nothing to backtrack to. Instead, use a lightweight lookahead scan that only inspects token types without consuming them:

- **For `(`**: scan forward from current position: if the tokens match the pattern `LPAREN (token COMMA)* token RPAREN ARROW`, this is a lambda. The scan checks token types in the existing token slice (O(n) where n = number of params), does not modify parser state, and does not call `parseExpr`. The scan accepts any token between LPAREN and RPAREN (not just IDENT) so that `parseLambda` is entered and can produce specific error messages (e.g., `(false) => 1` -> "expected identifier, got `false`"). If the pattern does not match (no RPAREN ARROW sequence), fall through to existing behavior (`parseDatumTerm` error for `(` in primary position).
- **For `ident` (not `r`)**: peek at the next token. If it is `ARROW`, parse as single-param lambda. Otherwise fall through to existing behavior.

This avoids the complexity of save/restore state and does not require adding `(expr)` grouping support.

#### Parameter scope

Inside the lambda body, parameter names are resolved to `reql.Var(id)`. A parameter scope map (`name -> id`) is stored as a field on the parser struct and maintained during body parsing. When `parsePrimary` encounters a bare identifier that matches a parameter name, it produces `Var(id)` instead of an error. Chaining on `Var` terms works naturally (e.g., `x('field')` -> `Var(1).Bracket("field")`).

Parameter scoping rules:
- Parameters shadow `r.row` -- using `r.row` inside an arrow function is an error (ambiguous)
- Nested arrows are not supported (max lambda depth = 1, matches RethinkDB limitation)
- Parameter names must be valid identifiers; reserved words (`r`, `true`, `false`, `null`) are rejected at the identifier check stage

#### Arrow body boundaries

The `=>` operator has the lowest precedence. The body is parsed by calling `parseExpr` which greedily consumes everything including chains. Inside an argument list (e.g., `branch(...)`), the body naturally terminates at `,` or `)` because `parseExpr` does not consume these tokens. Example:
- `r.branch((x) => x('a').gt(1), "yes", "no")` -- body ends at comma inside branch

Note: `chainFilter` uses `parseOneArg` (single expression), so `filter` with optargs like `filter((x) => x('a'), {default: true})` is not supported. This is a pre-existing parser limitation unrelated to arrow syntax.

#### Interaction with `wrapImplicitVar` in `Filter`

`term.Filter()` calls `wrapImplicitVar` on its predicate. When the parser produces a `FUNC` term from arrow syntax, `wrapImplicitVar` finds no `IMPLICIT_VAR` inside and returns the term unchanged -- no double wrapping occurs. This must be verified by a wire JSON roundtrip test.

Note: `Map`, `Reduce`, `ConcatMap`, `ForEach` do NOT call `wrapImplicitVar`. This means `r.row` does not work in these methods (existing limitation). Arrow syntax solves this because it produces an explicit `FUNC` term that these methods pass through directly.

### Wire format

`(x) => x('age').gt(21)` serializes as:
```json
[69,[[2,[1]],[21,[[170,[[10,[1]],"age"]],21]]]]
```
Which is `FUNC([MAKE_ARRAY([1])], GT(BRACKET(VAR(1), "age"), 21))`.

All parser tests verify wire JSON via `MarshalJSON` roundtrip, not just Term structure.

### Task 1: Lexer -- arrow token

- [ ] Test: tokenize `=>` -> [ARROW] token
- [ ] Test: tokenize `(x) => x` -> [LPAREN, IDENT:x, RPAREN, ARROW, IDENT:x]
- [ ] Test: tokenize `=` alone -> error "unexpected character"
- [ ] Test: tokenize `==` -> error (not supported)
- [ ] Test: fuzz -- lexer does not panic with `=` in arbitrary positions
- [ ] Implement: add `tokenArrow` type, handle `=` in lexer

### Task 2: Parser -- single-param arrow in parentheses

- [ ] Test: parse `(x) => x('age').gt(21)` -> wire JSON `[69,[[2,[1]],[21,[[170,[[10,[1]],"age"]],21]]]]`
- [ ] Test: parse `(x) => x.eq(5)` -> wire JSON with FUNC wrapping EQ(VAR(1), 5)
- [ ] Test: parse `(x) => true` -> wire JSON with FUNC wrapping datum true
- [ ] Test: parse `(r) => r('f')` -> error (reserved parameter name "r")
- [ ] Test: parse `(false) => 1` -> error (expected identifier, got "false")
- [ ] Test: parse `(null) => 1` -> error (expected identifier, got "null")
- [ ] Implement: lookahead scan for `(params) =>` pattern, arrow parsing with param scope

### Task 3: Parser -- multi-param arrow

- [ ] Test: parse `(a, b) => a.add(b)` -> wire JSON `[69,[[2,[1,2]],[24,[[10,[1]],[10,[2]]]]]]`
- [ ] Test: parse `(a, b, c) => a.add(b).add(c)` -> wire JSON with 3-param FUNC
- [ ] Test: parse `(x, x) => x` -> error "duplicate parameter name"
- [ ] Test: parse `() => 1` -> error "at least one parameter required"
- [ ] Test: parse `(a,) => a` -> error (trailing comma)
- [ ] Implement: multi-param arrow parsing

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
- [ ] Implement: parser `params` field (map[string]int), nesting guard, r.row conflict check

### Task 6: Arrow body boundaries and precedence

The `=>` body is greedy (lowest precedence) but terminates at `,` and `)` inside argument lists.

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
