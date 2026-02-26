# Plan: Fix Remaining Parser Errors

## Overview

Fix 5 remaining parser errors collected from real-world usage (see `parser-errors.md`):

1. **Bracket numeric index** (`term(0)`) -- parser only accepts string literals in bracket notation, but ReQL supports numeric indexing via `Nth`.
2. **Nested functions** -- `function(){}` inside another `function(){}` is rejected. Requires lifting the nesting guard and implementing proper scoping with stacked parameter maps.
3. **`.sample()` method** -- not implemented in either the reql builder or parser.
4. **OptArgs in `insert()`** -- `insert(doc, {return_changes: true})` fails because `chainInsert` only parses one argument.
5. **Arrow `=> ({...})`** -- parenthesized object literal after `=>` is not recognized because `parsePrimary` has no case for `(expr)` grouping.

Package: `internal/reql`, `internal/reql/parser`

Depends on: `22-parser-lambda`, `23-parser-function-syntax`

## Context

- Files involved: `internal/reql/term.go` (builder), `internal/reql/parser/parser.go` (parser core), `internal/reql/parser/chain.go` (chain methods), `internal/reql/parser/parser_test.go`, `internal/reql/parser/fuzz_test.go`, `internal/integration/parser_test.go`
- Related patterns: existing `intArgChain` helper for `skip`/`nth`; `parseOneArg`/`parseArgList` for argument parsing; `parseLambda`/`parseFunctionExpr` for function scoping
- Dependencies: `internal/proto` (TermSample=81, TermNth=45, TermBracket=170), `internal/reql` (Term builder methods)

## Development Approach

- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task** -- no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Testing Strategy

- **Unit tests**: required for every task -- both success and error scenarios
- **Fuzz tests**: dedicated task 8 extends seed corpus for all new features
- **Integration tests**: dedicated task 9 runs against live RethinkDB via testcontainers

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Technical Details

### Bracket numeric index

Current: `parseChain` calls `parseOneStringArg()` for bracket notation -> only `tokenString` accepted.

Fix: replace `parseOneStringArg()` call in the bracket-notation case with a new helper `parseBracketArg()`. When argument is `tokenString`, produce `t.Bracket(field)`. When argument is `tokenNumber` and is an integer, produce `t.Nth(n)`. This correctly maps `term(0)` to NTH (term type 45) and `term("field")` to BRACKET (term type 170).

Wire JSON: `r.table("t").limit(1)(0)` -> `[45,[[71,[[15,["t"]],1]],0]]` (NTH).

### Nested functions

Current: `p.params != nil` guard in `parseIdentPrimary` rejects any function/lambda inside another.

Fix: replace single `params map[string]int` with a stack (`paramsStack []map[string]int`) and add `nextVarID int` field on the parser struct (NOT package-level). On entering a lambda/function, compute new IDs starting from `p.nextVarID + 1`, push a new scope. On exit, pop the scope. Parameter lookup walks the stack from top to bottom. `r.row` check: error if any scope is active (`len(paramsStack) > 0`).

ID allocation strategy: when stack is empty (top-level lambda), IDs start at 1 (preserving backward compat with sibling lambdas -- `map(x => x).filter(y => y)` both use VAR(1)). When stack is non-empty (nested lambda), IDs start at `max(all active scope IDs) + 1` to avoid collisions with outer scopes.

Wire JSON for nested example: `function(doc) { return doc("f").filter(function(item) { return item("type").eq("x") }) }` -> `[69,[[2,[1]],[39,[[170,[[10,[1]],"f"]],[69,[[2,[2]],[17,[[170,[[10,[2]],"type"]],"x"]]]]]]]]` -- outer FUNC with VAR(1), inner FUNC with VAR(2).

Nesting depth limit: reuse existing `maxDepth` (256) -- each lambda/function increments `depth`, so deeply nested functions are bounded.

### `.sample()` method

`TermSample` (81) already exists in `internal/proto/term.go`.

Add `Sample(n int) Term` to `internal/reql/term.go`. Register `"sample"` in parser's `registerCoreChain()` using existing `intArgChain` helper (same pattern as `skip`, `nth`).

Wire JSON: `r.table("t").sample(5)` -> `[81,[[15,["t"]],5]]`.

### OptArgs in `insert()`, `update()`, `delete()`

Current: `chainInsert` calls `parseOneArg()` -- only one argument. `chainUpdate` uses `oneArgChain`. `chainDelete` uses `noArgChain`.

**Cross-package constraint**: all fields of `reql.Term` are unexported (`datum`, `termType`, etc.), so the parser cannot introspect a parsed Term to extract a map. The `termToOptArgs(t reql.Term)` approach is NOT viable from the parser package.

Fix: add a `parseOptArgs()` helper in the parser that directly parses `{key: val, ...}` into `reql.OptArgs` (a `map[string]interface{}`). This avoids cross-package visibility issues. The helper peeks for `tokenLBrace`, then parses key-value pairs where values are restricted to datum literals (strings, numbers, bools) -- optargs don't contain Term expressions.

**insert/update** (two-arg pattern): change `chainInsert`/`chainUpdate` to use `parseArgList()`. If one argument, call without opts. If two arguments, parse second as optargs via `parseOptArgs()`.

**delete** (different pattern -- opts only, no doc): change `chainDelete` from `noArgChain` to a custom function. If `()` -- call `t.Delete()`. If `({...})` -- parse the single argument as optargs via `parseOptArgs()` and call `t.Delete(opts)`.

**Builder change required**: `reql.Term.Delete()` currently takes no arguments. Must update to `Delete(opts ...OptArgs) Term` (variadic, so `Delete()` without args still compiles at all existing call sites).

### Arrow `=> ({...})`

Current: `parsePrimary` only enters lambda path when `tokenLParen && isLambdaAhead()`. When `(` is not a lambda start, there is no fallback to parse `(expr)` as a grouped expression.

Fix: add a new case in `parsePrimary` for `tokenLParen` when `isLambdaAhead()` is false: consume `(`, call `parseExpr()`, expect `)`. This handles `({key: val})` as a parenthesized object literal, and also enables general grouping like `(1 + 2)` if needed in the future.

The arrow body parser (`parseLambda`/`parseFunctionExpr`) calls `parseExpr()` which calls `parsePrimary()`, so `=> ({...})` naturally works: the `(` triggers grouped-expression parsing, inner `{...}` is parsed as an object term, `)` closes the group.

Wire JSON: `row => ({name: row("name")})` -> `[69,[[2,[1]],{"name":[170,[[10,[1]],"name"]]}]]` (FUNC wrapping a datum object). Note: `parseObjectTerm()` returns `reql.Datum(map)` (termType==0), which serializes as a raw JSON object -- NOT as a MAKE_OBJ term `[3, ...]`.

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `go test ./internal/reql/parser/... -race -count=1`
- `go test -tags integration ./internal/integration/... -race -count=1 -run TestParserFixes`
- `make build`

## Implementation Steps

### Task 1: Bracket notation with numeric index

- [x] Test: parse `r.table("t").limit(1)(0)` -> wire JSON with NTH term `[45,[...,0]]`
- [x] Test: parse `r.row("items")(0)` -> NTH on BRACKET(IMPLICIT_VAR, "items")
- [x] Test: parse `r.table("t").insert({a: 1})("changes")(0)("new_val")` -> chained BRACKET, NTH, BRACKET
- [x] Test: parse `r.table("t")(0)("name")` -> NTH then BRACKET
- [x] Test: parse `r.table("t")("field")` -> BRACKET (no regression, still works)
- [x] Test: parse `r.table("t")(0.5)` -> error (float in bracket not valid)
- [x] Test: parse `r.table("t")(-1)` -> NTH with negative index (valid ReQL, last element)
- [x] Implement: add `parseBracketArg()` helper; dispatch to `Nth(n)` for integers, `Bracket(s)` for strings; replace `parseOneStringArg()` call in `parseChain` bracket case
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task

### Task 2: `.sample()` method -- reql builder

- [x] Test: `reql.Table("t").Sample(5)` -> wire JSON `[81,[[15,["t"]],5]]`
- [x] Test: `reql.Table("t").Sample(1)` -> wire JSON `[81,[[15,["t"]],1]]`
- [x] Test: `reql.Table("t").Sample(0)` -> wire JSON `[81,[[15,["t"]],0]]`
- [x] Implement: add `Sample(n int) Term` method to `term.go`
- [x] Run `go test ./internal/reql/... -race -count=1` -- must pass before next task

### Task 3: `.sample()` method -- parser

- [x] Test: parse `r.table("t").sample(5)` -> wire JSON `[81,[[15,["t"]],5]]`
- [x] Test: parse `r.table("t").sample(1).pluck("id","name")` -> SAMPLE chained with PLUCK
- [x] Test: parse `r.table("t").sample(0)` -> wire JSON (edge case, valid ReQL)
- [x] Implement: register `"sample"` in `registerCoreChain()` via `intArgChain`
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task

### Task 4: Parenthesized expressions (`(expr)` grouping)

- [x] Test: parse `r.table("t").map(row => ({name: row("name")}))` -> FUNC wrapping datum object `{"name": BRACKET(VAR(1), "name")}`
- [x] Test: parse `r.table("t").map(row => ({a: row("x"), b: row("y")}))` -> datum object with two fields
- [x] Test: parse `r.table("t").map((x) => ({id: x("id"), n: x("name").upcase()}))` -> datum object with chain in value
- [x] Test: parse `r.table("t").map(row => row("name"))` -> no regression, arrow without parens still works
- [x] Test: parse `r.table("t").filter(row => row("age").gt(21))` -> no regression
- [x] Test: parse `(` -> error (unclosed paren, expected expression)
- [x] Implement: add `tokenLParen && !isLambdaAhead()` case in `parsePrimary`: consume `(`, `parseExpr()`, expect `)`
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task

### Task 5: Nested functions -- scoping infrastructure

- [x] Test: parse `function(a){ return function(b){ return b } }` -> outer FUNC(VAR(1)), inner FUNC(VAR(2))
- [x] Test: parse `(a) => (b) => b` -> outer FUNC(VAR(1)), inner FUNC(VAR(2))
- [x] Test: parse `function(x){ return (y) => y("f") }` -> mixed nesting: outer function, inner arrow
- [x] Test: parse `(x) => function(y){ return y("f") }` -> mixed nesting: outer arrow, inner function
- [x] Test: parameter shadowing `(x) => (x) => x` -> inner `x` is VAR(2), not VAR(1)
- [x] Test: outer param accessible in inner body `(a) => (b) => a.add(b)` -> VAR(1).add(VAR(2))
- [x] Test: `r.row` inside any nesting level -> error "r.row inside arrow function is ambiguous"
- [x] Test: three levels `(a) => (b) => (c) => c` -> 3 nested FUNC terms with VAR(1), VAR(2), VAR(3)
- [x] Implement: replace `params map[string]int` with `paramsStack []map[string]int` and `nextVarID int` on parser struct; push/pop on lambda enter/exit; when stack empty IDs start at 1 (backward compat), when nested IDs start at max(active IDs)+1; update param lookup to walk stack top-to-bottom; update `r.row` guard to check `len(paramsStack) > 0`; update all nesting guards in `parseLambda`, `parseFunctionExpr`, `parseBareArrowLambda`
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task

### Task 6: Nested functions -- integration with chain methods

- [x] Test: parse `r.table("t").map(function(doc){ return doc("items").filter(function(i){ return i("active").eq(true) }) })` -> MAP with outer FUNC wrapping FILTER with inner FUNC
- [x] Test: parse `r.table("t").map((doc) => doc("items").filter((i) => i("active").eq(true)))` -> same structure, arrow syntax
- [x] Test: parse `r.table("t").filter(function(doc){ return doc("tags").contains(function(tag){ return tag.eq("hot") }) })` -> FILTER with nested CONTAINS FUNC
- [x] Test: parse `r.table("t").map(function(doc){ return doc.merge({count: doc("items").count()}) })` -> MAP with FUNC, MERGE with nested expression (no inner function -- verify no regression)
- [x] Implement: verify chain methods work with nested functions via existing infrastructure
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task

### Task 7a: Update `Delete` builder to accept OptArgs

- [ ] Test: `reql.Table("t").Delete()` -> wire JSON `[54,[[15,["t"]]]]` (no regression)
- [ ] Test: `reql.Table("t").Delete(reql.OptArgs{"durability": "soft"})` -> wire JSON with optargs
- [ ] Implement: change `Delete()` to `Delete(opts ...OptArgs) Term` in `term.go`; add optargs handling matching Insert/Update pattern
- [ ] Run `go test ./internal/reql/... -race -count=1` -- must pass before next task

### Task 7b: OptArgs in `insert()`, `update()`, `delete()` -- parser

- [ ] Test: parse `r.table("t").insert({a: 1}, {return_changes: true})` -> INSERT with optargs `{"return_changes": true}`
- [ ] Test: parse `r.table("t").insert({a: 1}, {conflict: "replace"})` -> INSERT with optargs `{"conflict": "replace"}`
- [ ] Test: parse `r.table("t").insert({a: 1}, {durability: "soft", return_changes: true})` -> INSERT with multiple optargs
- [ ] Test: parse `r.table("t").insert({a: 1})` -> INSERT without optargs (no regression)
- [ ] Test: parse `r.table("t").insert({a: 1}, "bad")` -> error (second arg must be object)
- [ ] Test: parse `r.table("t").update({x: 1}, {durability: "soft"})` -> UPDATE with optargs
- [ ] Test: parse `r.table("t").delete({durability: "soft"})` -> DELETE with optargs (1-arg pattern: opts only, no doc)
- [ ] Test: parse `r.table("t").delete()` -> DELETE without optargs (no regression)
- [ ] Test: parse `r.table("t").insert({a: 1}, {return_changes: true})("changes")(0)("new_val")` -> INSERT with optargs chained with BRACKET and NTH (depends on task 1)
- [ ] Implement: add `parseOptArgs()` helper that directly parses `{key: val}` into `reql.OptArgs` (avoids cross-package access to unexported Term fields); modify `chainInsert` and `chainUpdate` to use `parseArgList` with optional second optargs; replace `chainDelete` with custom function: `()` -> `Delete()`, `({...})` -> `Delete(opts)`
- [ ] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task

### Task 8: Fuzz testing

- [ ] Test: fuzz parser does not panic with bracket-numeric seeds
- [ ] Test: fuzz parser does not panic with nested function seeds
- [ ] Test: fuzz parser does not panic with parenthesized expression seeds
- [ ] Test: fuzz parser does not panic with insert optargs seeds
- [ ] Seed corpus: `r.table("t")(0)`, `r.table("t")(0)("f")`, `r.table("t").sample(1)`, `r.table("t").insert({a:1},{return_changes:true})`, `row => ({a: row("b")})`, `(x) => (y) => y`, `function(a){ return function(b){ return b } }`, `r.table("t")(0.5)`, `r.table("t").insert({a:1},)`, `=> ({})`, `(()`, `function(x){ return function(y){ return function(z){ return z } } }`
- [ ] Implement: extend `FuzzParse` seed corpus
- [ ] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task

### Task 9: Integration tests (live RethinkDB)

Build tag: `//go:build integration`. Package: `internal/integration`.

- [ ] Test `TestParserFixesBracketNumericIndex`: seed table with 3 docs; parse and execute `r.db('<db>').table('t').orderBy("id").limit(1)(0)` -> returns single doc (atom, not array)
- [ ] Test `TestParserFixesSample`: seed table with 10 docs; parse and execute `r.db('<db>').table('t').sample(3)` -> returns exactly 3 docs
- [ ] Test `TestParserFixesNestedFunction`: seed table with docs containing nested array field `items: [{type: "a"}, {type: "b"}]`; parse and execute `r.db('<db>').table('t').map(function(doc){ return doc("items").filter(function(i){ return i("type").eq("a") }) })` -> returns filtered inner arrays
- [ ] Test `TestParserFixesNestedArrow`: same data; parse and execute `r.db('<db>').table('t').map((doc) => doc("items").filter((i) => i("type").eq("a")))` -> same result as nested function syntax
- [ ] Test `TestParserFixesInsertOptArgs`: parse and execute `r.db('<db>').table('t').insert({id: "new", val: 1}, {return_changes: true})` -> result contains `changes` array with `new_val`
- [ ] Test `TestParserFixesArrowParenObject`: seed table with `{id: "1", first: "Alice", last: "Smith"}`; parse and execute `r.db('<db>').table('t').map(row => ({full: row("first").add(" ").add(row("last"))}))` -> returns `[{full: "Alice Smith"}]`
- [ ] Test `TestParserFixesCLI`: run CLI binary with `r.db('<db>').table('t').sample(3)` -> valid JSON output, exit code 0
- [ ] Implement: integration test functions using existing test helpers (setupTestDB, createTestTable, seedTable, newExecutor, cliRun)
- [ ] Run `go test -tags integration ./internal/integration/... -race -count=1 -run TestParserFixes` -- must pass before next task

### Task 10: Verify acceptance criteria

- [ ] Verify all 5 parser errors from Overview are fixed
- [ ] Verify edge cases are handled (float in bracket, empty optargs, deeply nested functions)
- [ ] Run full test suite: `go test ./internal/reql/... -race -count=1`
- [ ] Run full parser tests: `go test ./internal/reql/parser/... -race -count=1`
- [ ] Run linter: `make build` -- all issues must be fixed
