# Plan: Function Syntax and `r` Parameter Name in Parser

## Overview

Two changes to the ReQL string parser:

1. **Allow `r` as a lambda parameter name.** Currently `r` is rejected as reserved because it conflicts with the global `r.*` namespace. Inside a lambda body, when `r` is a declared parameter, it should resolve to `Var(id)` instead of triggering `parseRExpr`. This enables idiomatic JS-style expressions like `filter((r) => r('enabled').eq(false))`.

2. **Add `function(params){ return expr }` syntax.** This is the traditional JavaScript function expression syntax used by the official RethinkDB Data Explorer. It desugars to the same `FUNC` term as arrow syntax.

Both syntaxes must produce identical wire JSON for the same logic.

Package: `internal/reql/parser`

Depends on: `22-parser-lambda`

## Design Notes

### Allowing `r` as parameter name

Current flow in `parseIdentPrimary`:
```
if tok.Value == "r" -> parseRExpr()
if p.params[tok.Value] -> Var(id)
```

Fix: when `p.params` is non-nil and contains `r`, check params first. Reorder the two branches so that param lookup takes priority over `r.*` dispatch when inside a lambda. Outside a lambda (`p.params == nil`), behavior is unchanged.

Remove the `r` check from `validateLambdaParam`. The only truly reserved names are `true`, `false`, `null` (which are tokenized as `tokenBool`/`tokenNull`, not `tokenIdent`, so they already fail the `tokenIdent` check).

`r.row` conflict check in `parseRRow` stays: if someone writes `(r) => r.row('f')`, the `r` resolves as Var, then `.row` is an unknown chain method -- natural error, no special handling needed.

### `function` keyword syntax

Syntax: `function(params){ return expr }`

- `function` is parsed as `tokenIdent` by the existing lexer (no new token type needed).
- Detection: in `parsePrimary`/`parseIdentPrimary`, when tok is `function` and next token is `(`, parse as function expression.
- Parameter list: reuse `parseLambdaParams` (same validation rules).
- Body: expect `{`, then optional `return` keyword (consumed if present, not required), then `parseExpr()`, then optional `;`, then `}`.
- `return` is also a regular `tokenIdent` -- no new token type needed.
- Scoping: identical to arrow lambda -- same `p.params` mechanism, same nesting guard, same `r.row` conflict.
- Wire output: identical to arrow syntax -- `reql.Func(body, ids...)`.

Body boundaries: `parseExpr` consumes greedily up to `}` (which is not a valid expression continuation), so the `}` naturally terminates the body.

### Nesting rules

- `function` inside arrow or arrow inside `function` both rejected by existing `p.params != nil` nesting guard.
- `function` inside `function` also rejected by the same guard.

## Validation Commands
- `go test ./internal/reql/parser/... -race -count=1`
- `go test -tags integration ./internal/integration/... -race -count=1 -run TestFunctionSyntax`
- `make build`

### Task 1: Allow `r` as lambda parameter name

- [x] Test: parse `(r) => r('enabled').eq(false)` -> wire JSON with FUNC wrapping EQ(BRACKET(VAR(1), "enabled"), false)
- [x] Test: parse `r.table('t').filter((r) => r('age').gt(21))` -> FILTER with FUNC, VAR(1) in body
- [x] Test: parse `(r) => r('a').add(r('b'))` -> multiple VAR(1) refs
- [x] Test: parse `r => r('field')` -> bare arrow with `r` param works
- [x] Test: parse `r.db('test')` -> still works (no regression outside lambda)
- [x] Test: parse `r.table('t').filter((r) => r.row('f'))` -> error (`r` resolves as Var, `.row` is unknown chain method -- natural error)
- [x] Implement: reorder `parseIdentPrimary` to check `p.params` before `r` dispatch; remove `r` from `validateLambdaParam` reserved check

### Task 2: Lexer -- `function` and `return` as identifiers (no changes needed, verify)

- [x] Test: tokenize `function(x){ return x }` -> [IDENT:function, LPAREN, IDENT:x, RPAREN, LBRACE, IDENT:return, IDENT:x, RBRACE]
- [x] Test: tokenize `function(){}` -> [IDENT:function, LPAREN, RPAREN, LBRACE, RBRACE]
- [x] Implement: no lexer changes; `function` and `return` are already valid identifiers; verify with tests

### Task 3: Parser -- `function(params){ return expr }` single param

- [x] Test: parse `function(x){ return x('age').gt(21) }` -> same wire JSON as `(x) => x('age').gt(21)`
- [x] Test: parse `function(x){ x('age').gt(21) }` -> same wire JSON (return keyword optional)
- [x] Test: parse `function(x){ return x('age').gt(21); }` -> same wire JSON (trailing semicolon tolerated)
- [x] Test: parse `r.table('t').filter(function(x){ return x('age').gt(21) })` -> FILTER with FUNC
- [x] Implement: detect `function` ident followed by `(` in `parseIdentPrimary`; add `parseFunctionExpr` method

### Task 4: Parser -- `function(params){ return expr }` with `r` param

- [x] Test: parse `function(r){ return r('enabled').eq(false) }` -> wire JSON with FUNC wrapping EQ(BRACKET(VAR(1), "enabled"), false)
- [x] Test: parse `r.db('restored').table('routes').filter(function(r){ return r('enabled').eq(false) })` -> full chain with FUNC
- [x] Test: parse `r.table('t').filter((r) => r('enabled').eq(false))` -> same wire JSON as function syntax equivalent
- [x] Implement: reuse task 1 param scoping; `r` inside function body resolves to Var when in params

### Task 5: Parser -- `function` multi-param and error cases

- [x] Test: parse `function(a, b){ return a.add(b) }` -> same wire JSON as `(a, b) => a.add(b)`
- [x] Test: parse `function(a, b, c){ return a.add(b).add(c) }` -> 3-param FUNC
- [x] Test: parse `function(){ return 1 }` -> error "lambda requires at least one parameter"
- [x] Test: parse `function(x, x){ return x }` -> error "duplicate parameter name"
- [x] Test: parse `function(x){ }` -> error (empty body, expected expression)
- [x] Test: parse `function(x){ return }` -> error (missing expression after return)
- [x] Test: parse `function(x) x` -> error (missing `{`)
- [x] Test: parse `function(x){ return x('a')` -> error (missing `}`)
- [x] Test: nested `function(x){ return function(y){ return y } }` -> error "nested arrow functions are not supported"
- [x] Test: `function(x){ return r.row('f') }` -> error "r.row inside arrow function is ambiguous"
- [x] Implement: error handling in `parseFunctionExpr`

### Task 6: Parser -- semicolon handling in lexer

- [x] Test: tokenize `function(x){ return x; }` -> tokens include IDENT:x followed by some representation of `;`
- [x] Implement: add `;` as a recognized punctuation token (`tokenSemicolon`) so the parser can optionally consume it before `}`; alternatively, if `;` is not in the lexer, it will error -- must handle it

### Task 7: Fuzz testing

- [x] Test: fuzz parser with function-syntax seeds does not panic
- [x] Seed corpus: `function(x){ return x }`, `function(a,b){ return a.add(b) }`, `function(r){ return r('f') }`, `function(){ return 1 }`, `function(x){}`, `function(x){ return }`, `function x`, `function`, `(r) => r('f')`
- [x] Implement: extend `FuzzParse` seed corpus with function-syntax patterns

### Task 8: Integration tests (live RethinkDB)

Build tag: `//go:build integration`. Package: `internal/integration`.

These tests use the exact expressions from the user's requirements.

- [x] Test `TestFunctionSyntaxFilter`: seed table `routes` with `{id: "1", enabled: true}, {id: "2", enabled: false}, {id: "3", enabled: false}`; parse and execute `r.db('<db>').table('routes').filter(function(r){ return r('enabled').eq(false) })` -> returns exactly 2 docs with `enabled: false`
- [x] Test `TestFunctionSyntaxArrowWithR`: same table; parse and execute `r.db('<db>').table('routes').filter((r) => r('enabled').eq(false))` -> returns same 2 docs
- [x] Test `TestFunctionSyntaxEquivalence`: verify both syntaxes produce identical result sets on the same data
- [x] Test `TestFunctionSyntaxCLI`: run CLI binary with `r.db('<db>').table('routes').filter(function(r){ return r('enabled').eq(false) })` -> valid JSON output with expected rows
- [x] Test `TestFunctionSyntaxCLIArrowR`: run CLI binary with `r.db('<db>').table('routes').filter((r) => r('enabled').eq(false))` -> same result
- [x] Implement: integration test functions using existing test helpers (setupTestDB, createTestTable, seedTable, newExecutor, cliRun)
