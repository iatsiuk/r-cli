# Plan: Parser Errors From Production Log

## Overview

Fix the parser gaps recorded in `~/.r-cli/parser-errors.log` (81 entries, 15 root causes) so that real-world ReQL expressions pasted from the RethinkDB Data Explorer parse correctly.

Ranked by frequency in the log:

1. **`group()` with anything but one string literal** (44 of 81 entries) -- multiple group keys, lambdas, `r.row(...)`, array of keys, `{index}`/`{multi}` optargs.
2. **`min`/`max`/`sum`/`avg` beyond one string literal** (15 entries) -- zero arguments, `{index: "..."}` optargs, expressions and lambdas.
3. **Infix arithmetic** (4 entries) -- `60*60*24*30`, `1779222884700+1`.
4. **`r.tableList()`** (3 entries) -- top-level form missing.
5. **`var x = ...;` in `function` bodies** (2 entries).
6. **Field selectors with an expression or an array** (2 entries) -- `t.hasFields(f)`, `t.hasFields(["a","b"])`.
7. **`.branch()` chain form** (2 entries).
8. **Bracket notation and `getField` with an expression** (2 entries) -- `p(f)` where `f` is a lambda parameter.
9. **Term-valued optargs** (1 entry) -- `{index: r.desc("...")}`.
10. **Arrow lambda with a block body** (1 entry) -- `g => { return {...} }`.
11. **`slice()` with a single argument** (1 entry) -- `slice(-2)`.

Two related defects are folded in because they are the same code paths:

- **Silent optargs downgrade**: `orderBy({index: r.desc("d")})` parses today without error but produces the wrong query. `tryTrailingOptArgs` rejects the term-valued optarg, backtracks, and the object is re-parsed as a positional MAKE_OBJ argument to ORDER_BY instead of an optarg. Nothing in the log shows this because it is not a parse error.
- **Missing funcWrap**: `wrapImplicitVar` is applied only in `Filter`. `map(r.row("a"))`, `sum(r.row("x"))` and similar parse fine but send a bare IMPLICIT_VAR that the server rejects. Official drivers funcWrap every function-position argument.

Out of scope, handled only by better error messages: `new Date("...").getTime()` (2 entries), multiple statements separated by `;` (1 entry), missing `r.` prefix (1 entry).

Package: `internal/reql`, `internal/reql/parser`

## Context

- Files involved: `internal/reql/term.go` (builder), `internal/reql/term_test.go`, `internal/reql/parser/lexer.go`, `internal/reql/parser/lexer_test.go`, `internal/reql/parser/parser.go`, `internal/reql/parser/parser_test.go`, `internal/reql/parser/fuzz_test.go`, `internal/integration/parser_test.go`
- Term types already present in `internal/proto/term.go`: GROUP 144, SUM 145, AVG 146, MIN 147, MAX 148, SLICE 30, BRANCH 65, TABLE_LIST 62, GET_FIELD 31, BRACKET 170, HAS_FIELDS 32, MATCH 97, ADD 24, SUB 25, MUL 26, DIV 27, MOD 28, FUNC 69, VAR 10, IMPLICIT_VAR 13, MAKE_ARRAY 2, DESC 74
- Existing patterns to reuse: `wrapImplicitVar` / `errTerm` in `term.go`; `parseArgListWithOpts` / `tryTrailingOptArgs` / `parseFoldOpts` in `parser.go`; `noArgChainWithOpts` and friends for chain registration
- `reql.Term` fields are unexported, so the parser cannot introspect a parsed Term. Optargs must be produced by the parser directly, as `parseOptArgs` already does.
- Every affected builder method is called only from the parser and from tests, so widening signatures to variadic `...interface{}` is source-compatible at all existing call sites.
- Adopted from the analysis of `~/.r-cli/parser-errors.log` (entries dated 2026-05-01 to 2026-07-30, versions 1.1.1 through 1.2.0).

## Development Approach

- **Testing approach**: strict TDD -- write the test first, run it, confirm it fails for the expected reason, only then write the implementation
- **CRITICAL: never write implementation code before the failing test exists** -- a task whose tests were written after the code must be redone
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: every parser-level task MUST also ship integration coverage** in `internal/integration`, following the existing style of that directory
- **CRITICAL: all tests must pass before starting next task** -- no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- Builder change first, parser change second, so each layer is verified independently
- Maintain backward compatibility: existing expressions must produce byte-identical wire JSON

## Testing Strategy

- **Unit tests**: required for every task -- success and error scenarios, asserted on wire JSON
- **Regression tests**: each task that widens a signature must keep a test for the old narrow form
- **Integration tests**: required in every parser-level task, not deferred to the end. Added to `internal/integration/parser_test.go` under the `integration` build tag, run against a live `rethinkdb:2.4.4` container via testcontainers, reusing the existing `newExecutor`, `setupTestDB`, `createTestTable`, `seedTable`, `waitForIndex`, `atomRows` and `closeCursor` helpers. Wire JSON proves the query shape; the integration test proves the server accepts it and returns the expected rows.
- **Builder-only tasks**: unit tests are sufficient, because the parser task that consumes the builder carries the integration coverage
- **Fuzz tests**: dedicated task extends the seed corpus with the new syntax

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Technical Details

### group

Current: `Group(field string)` in `term.go:354`; registered via `strArgChain` at `parser.go:1531`.

Builder: `Group(fields ...interface{}) Term`. If the last element is an `OptArgs`, it becomes `term.opts`. Every remaining element goes through `toTerm` and then `wrapImplicitVar`; a wrap error returns `errTerm(err)`.

Parser: replace the `strArgChain` registration with a `chainGroup` built on the existing `parseArgListWithOpts`, mirroring `chainOrderBy` (`parser.go:919`).

Wire JSON:

- `r.table("t").group("a","b")` -> `[144,[[15,["t"]],"a","b"]]`
- `r.table("t").group(r.row("a"))` -> `[144,[[15,["t"]],[69,[[2,[1]],[170,[[10,[1]],"a"]]]]]]`
- `r.table("t").group(t => t("a"))` -> same shape as above
- `r.table("t").group("a",{index:"i"})` -> `[144,[[15,["t"]],"a"],{"index":"i"}]`

Note on array group keys: `group([r.row("a"), r.row("b")])` (log entry 23) is a MAKE_ARRAY argument, not multiple positional keys. It parses through the same path once the argument is no longer restricted to a string.

### min / max / sum / avg

Current: `strArgChain` at `parser.go:1535-1538`; builders at `term.go:363-381` take one string.

Builder: `Min(args ...interface{}) Term` and the same for `Max`, `Sum`, `Avg`. Zero args -> `[147,[seq]]`. One field arg -> appended after `toTerm` + `wrapImplicitVar`. Trailing `OptArgs` -> `term.opts`. More than one non-optargs argument -> `errTerm`.

Parser: one shared `chainAggregate` helper over `parseArgListWithOpts`, registered for all four names.

Wire JSON:

- `r.table("t").min()` -> `[147,[[15,["t"]]]]`
- `r.table("t").max({index:"d"})` -> `[148,[[15,["t"]]],{"index":"d"}]`
- `r.table("t").min(r.row("prices")("USD"))` -> `[147,[[15,["t"]],[69,[[2,[1]],[170,[[170,[[10,[1]],"prices"]],"USD"]]]]]]`
- `r.table("t").sum(x => x("v"))` -> `[145,[[15,["t"]],[69,[[2,[1]],[170,[[10,[1]],"v"]]]]]]`

### funcWrap coverage

`wrapImplicitVar` (`term.go:35`) is called only from `Filter` (`term.go:160`). Extend to every function-position argument, matching the official drivers: `Map`, `ConcatMap`, `ForEach`, `Reduce`, `Contains`, `OffsetsOf`, `OrderBy` field arguments, `Update`, `Replace`, plus `Group` and the four aggregations from the tasks above.

The wrap is a no-op when no IMPLICIT_VAR is present, so terms that already are FUNC or plain data are untouched and existing wire JSON does not change.

### Term-valued arguments where a string literal is required today

`parseBracketArg` (`parser.go:1660`), `getField` and `match` (`parser.go:1520`, `parser.go:1571`) accept only `tokenString`. Inside a lambda the natural argument is a parameter reference.

Builder: `GetField(field interface{})`, `Bracket(field interface{})`, `Match(re interface{})`, each using `toTerm`.

Parser: in `parseBracketArg` keep the `tokenString` -> BRACKET and integer -> NTH fast paths, then fall back to `parseExpr` -> `Bracket(term)`. Float literals stay an error. `getField` and `match` move from `strArgChain` to `oneArgChain`.

Wire JSON: `r.table("t").map(function(f){ return f(f) })` -> `[38,[[15,["t"]],[69,[[2,[1]],[170,[[10,[1]],[10,[1]]]]]]]]`

### Field selectors

`parseOneFieldSelector` (`parser.go:1987`) accepts a string literal or `{...}` object. Add two cases: `[` -> `parseDatumArray()` (already returns a MAKE_ARRAY term), and any other token -> `parseExpr()`. The string fast path stays so that existing wire JSON is unchanged.

Applies to `pluck`, `without`, `hasFields`, `withFields`.

Wire JSON: `r.table("t").filter(f => f.hasFields(["a","b"]))` -> `[39,[[15,["t"]],[69,[[2,[1]],[32,[[10,[1]],[2,["a","b"]]]]]]]]`

### Term-valued optargs and safe backtracking

`parseOptArgValue` (`parser.go:1907`) accepts only datum literals. `parseFoldOpts` already demonstrates that Term values inside `reql.OptArgs` marshal correctly, because `OptArgs` is a `map[string]interface{}` and `Term` implements `MarshalJSON`.

Fix: `parseOptArgValue` keeps the datum fast paths and falls back to `parseExpr()`.

This makes the `tryTrailingOptArgs` backtrack (`parser.go:1768`) no longer trivially safe -- its comment states that `parseOptArgs` only accepts datum literals and never touches `paramsStack`. With expression values a failed attempt can parse a lambda and advance `nextVarID`. Snapshot `pos`, `len(paramsStack)`, `nextVarID` and `depth` before the attempt and restore all four on failure; update the comment.

Wire JSON:

- `r.table("t").orderBy({index: r.desc("d")})` -> `[41,[[15,["t"]]],{"index":[74,["d"]]}]` (today: a positional MAKE_OBJ argument, which is wrong)
- `r.table("t").between(1, 2, {index: r.desc("d")})` -> BETWEEN with `{"index":[74,["d"]]}` in the optargs slot

### Small additions

- `r.tableList()`: add top-level `TableList() Term` mirroring `DBList()` (`term.go:127`) and register it in `buildRBuilders` (`parser.go:1438`). Wire: `[62,[]]`; the database comes from the query-level `db` optarg, same as top-level `r.table`.
- `.branch()`: add `(t Term) Branch(branches ...interface{}) Term` that prepends the receiver, reusing the existing top-level `Branch` validation (odd total count, at least 3). Register as a chain. Wire: `r.table("t").count().gt(0).branch([1],[])` -> `[65,[[21,[[43,[[15,["t"]]]],0]],[2,[1]],[2,[]]]]`
- `slice()` with one argument: `Slice(bounds ...int) Term` accepting 1 or 2 values, `errTerm` otherwise; `chainSlice` (`parser.go:1012`) parses one or two integers instead of requiring `parseTwoInts`. Wire: `r.table("t").slice(-2)` -> `[30,[[15,["t"]],-2]]`
- Zero-parameter functions: `parseLambdaParams` rejects `function(){...}` with "lambda requires at least one parameter". FUNC with an empty parameter list is valid ReQL and appears in the log as a group-everything idiom. Allow it: `[69,[[2,[]],body]]`

### Infix arithmetic

Lexer: add `tokenPlus`, `tokenMinus`, `tokenStar`, `tokenSlash`, `tokenPercent`. `-` is ambiguous: `readNumber` (`lexer.go:127`) currently swallows it as a sign. Resolve by previous-token context -- `-` starts a negative literal only when the previously emitted token is not one of ident, number, string, bool, null, `)`, `]`, `}`. This keeps `nth(-1)`, `slice(0,-1)` and `[1,-2]` lexing exactly as today while making `x - 2` a binary operator.

Parser: split `parseExpr` into precedence levels while keeping the depth guard at the top:

- `parseExpr` -> `parseAdditive`
- `parseAdditive` -> `parseMultiplicative { (+|-) parseMultiplicative }` mapping to `Add` / `Sub`
- `parseMultiplicative` -> `parsePostfix { (*|/|%) parsePostfix }` mapping to `Mul` / `Div` / `Mod`
- `parsePostfix` -> `parsePrimary` followed by the existing `parseChain`

Wire JSON: `r.expr(60*60*24)` -> `[26,[[26,[60,60]],24]]`

### Local variables in function bodies

`parseFunctionExpr` (`parser.go:209`) supports only `return expr`. Log entries 1 and 79 declare locals first.

Lexer: `readArrow` (`lexer.go:87`) errors on a lone `=`. Emit `tokenAssign` instead. `==` then lexes as two assign tokens and fails at the parser with a clear message, which is acceptable because ReQL has no `==`.

Parser: after `{`, loop over `var|let|const <ident> = <expr> ;` bindings into a locals scope stack aligned with the function scope stack, then parse the optional `return` and the body. `parseIdentPrimary` (`parser.go:165`) resolves locals before lambda parameters and before `r.*` dispatch.

Binding semantics: the bound Term is inlined at every use site. Using a local twice duplicates the subtree, which is what a user writing the expression by hand would produce. Document this in the helper comment.

Wire JSON: `r.table("t").filter(function(p){ var re = "x"; return p("id").match(re) })` -> `[39,[[15,["t"]],[69,[[2,[1]],[97,[[170,[[10,[1]],"id"]],"x"]]]]]]`

### Arrow lambda block bodies

`g => {return {...}}` currently fails with "expected ':'" because `{` after `=>` is parsed as an object literal. Adopting full JS semantics would be a breaking change: `=> {a: 1}` is valid today and means an object.

Heuristic: after `=>`, treat `{` as a block only when the following token is the identifier `return`, `var`, `let` or `const`. Everything else stays an object literal. The block body then reuses the same statement parser as `parseFunctionExpr`.

Wire JSON: `r.table("t").map(g => { return {a: g("b")} })` -> `[38,[[15,["t"]],[69,[[2,[1]],{"a":[170,[[10,[1]],"b"]]}]]]]`

### Error message hints

Three log cases stay unsupported but deserve actionable messages:

- `new Date("...")` -> "new Date() is not supported, use r.iso8601(\"...\") or r.epochTime(n)"
- a known builder name without the `r.` prefix, e.g. `table("x").count()` -> "unknown identifier \"table\", did you mean r.table(...)?"
- a top-level `;` followed by more input -> "multiple statements are not supported, run one query at a time or use --file with '---' separators"

## Validation Commands

- `go test ./internal/reql/... -race -count=1`
- `go test ./internal/reql/parser/... -race -count=1`
- `go test ./... -race -count=1`
- `go test -tags integration ./internal/integration/... -race -count=1`
- `make build`

## Implementation Steps

### Task 1: Builder -- variadic `Group` with optargs and funcWrap

- [x] Test: `reql.Table("t").Group("a")` -> `[144,[[15,["t"]],"a"]]` (no regression)
- [x] Test: `reql.Table("t").Group("a","b")` -> `[144,[[15,["t"]],"a","b"]]`
- [x] Test: `reql.Table("t").Group(reql.Row().Bracket("a"))` -> GROUP with FUNC-wrapped BRACKET on VAR(1)
- [x] Test: `reql.Table("t").Group(reql.Func(reql.Var(1).Bracket("a"), 1))` -> explicit FUNC passed through unwrapped
- [x] Test: `reql.Table("t").Group("a", reql.OptArgs{"index":"i"})` -> `[144,[[15,["t"]],"a"],{"index":"i"}]`
- [x] Test: `reql.Table("t").Group(reql.OptArgs{"multi":true})` -> GROUP with optargs and no positional key
- [x] Test: `reql.Table("t").Group(reql.Func(reql.Row(), 1))` -> error term (IMPLICIT_VAR inside nested FUNC)
- [x] Implement: change `Group(field string)` to `Group(fields ...interface{}) Term` in `term.go`; trailing `OptArgs` becomes `opts`; each field goes through `toTerm` + `wrapImplicitVar`; wrap error returns `errTerm`
- [x] Run `go test ./internal/reql/... -race -count=1` -- must pass before next task
- + Added `splitOptArgs` helper in `term.go` to share trailing-OptArgs handling with the aggregation builders of task 2

### Task 2: Builder -- variadic `Min`/`Max`/`Sum`/`Avg` with optargs and funcWrap

- [x] Test: `reql.Table("t").Min("f")` -> `[147,[[15,["t"]],"f"]]` (no regression, same for Max/Sum/Avg)
- [x] Test: `reql.Table("t").Min()` -> `[147,[[15,["t"]]]]`
- [x] Test: `reql.Table("t").Max(reql.OptArgs{"index":"d"})` -> `[148,[[15,["t"]]],{"index":"d"}]`
- [x] Test: `reql.Table("t").Min(reql.Row().Bracket("p"))` -> MIN with FUNC-wrapped BRACKET
- [x] Test: `reql.Table("t").Sum(reql.Func(reql.Var(1).Bracket("v"), 1))` -> SUM with explicit FUNC
- [x] Test: `reql.Table("t").Avg("a","b")` -> error term (at most one field argument)
- [x] Implement: change all four builders to `...interface{}` in `term.go` with shared argument handling; trailing `OptArgs` -> `opts`; single field -> `toTerm` + `wrapImplicitVar`; more than one field -> `errTerm`
- [x] Run `go test ./internal/reql/... -race -count=1` -- must pass before next task
- + Added unexported `(t Term) aggregate(tt, args)` helper in `term.go` shared by all four builders
- + Extra coverage beyond the listed cases: `Min("f", OptArgs{"index":"d"})` (field plus optargs) and `Sum(reql.Func(reql.Row(), 1))` (IMPLICIT_VAR inside nested FUNC -> error term)

### Task 3: Parser -- `group` with multiple keys, expressions and optargs

- [x] Test: parse `r.table("t").group("a")` -> `[144,[[15,["t"]],"a"]]` (no regression)
- [x] Test: parse `r.table("t").group("a","b","c")` -> GROUP with three string keys
- [x] Test: parse `r.table("t").group(r.row("a"))` -> GROUP with FUNC-wrapped BRACKET on VAR(1)
- [x] Test: parse `r.table("t").group(t => t("a"))` -> GROUP with FUNC
- [x] Test: parse `r.table("t").group(function(t){ return [t("a"), t("b")] })` -> GROUP with FUNC returning MAKE_ARRAY
- [x] Test: parse `r.table("t").group([r.row("a"), r.row("b")])` -> GROUP with a single MAKE_ARRAY key
- [x] Test: parse `r.table("t").group(t => t("a"), "b")` -> GROUP mixing a lambda and a string key
- [x] Test: parse `r.table("t").group("a",{index:"i"})` -> GROUP with optargs
- [x] Test: parse `r.table("t").group("a").count().ungroup().orderBy(r.desc("reduction"))` -> full pipeline from the log
- [x] Test: parse `r.table("t").group()` -> error (at least one key or optargs required)
- [x] Integration test: `group("a","b").count().ungroup()` over a seeded table returns the expected per-pair counts
- [x] Integration test: `group(t => t("a"))` and `group(r.row("a"))` return identical results
- [x] Integration test: `group("a",{index:"idx"})` on a table with a seeded secondary index
- [x] Implement: replace the `strArgChain` registration of `group` with `chainGroup` using `parseArgListWithOpts`, mirroring `chainOrderBy`
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + Added `argsWithOpts` helper in `parser.go` shared by `chainOrderBy` and `chainGroup`
- + Extra coverage beyond the listed cases: `group({multi:true})` (opts-only form) and `group("a",)` (trailing comma error)
- + Confirmed on the server: `group("city",{index:"dept"})` is accepted and yields two-element group keys `[city, dept]`

### Task 4: Parser -- `min`/`max`/`sum`/`avg` with no args, optargs and expressions

- [x] Test: parse `r.table("t").sum("f")` -> `[145,[[15,["t"]],"f"]]` (no regression, same for min/max/avg)
- [x] Test: parse `r.table("t").min()` -> `[147,[[15,["t"]]]]`
- [x] Test: parse `r.table("t").max({index:"createdAt"})` -> MAX with optargs
- [x] Test: parse `r.table("t").max({index:"createdAt"})("createdAt")` -> MAX with optargs chained with BRACKET
- [x] Test: parse `r.table("t").min(r.row("prices")("USD"))` -> MIN with FUNC-wrapped nested BRACKET
- [x] Test: parse `r.table("t").group("c").sum(x => x("balance")("amount"))` -> SUM with FUNC inside a grouped stream
- [x] Test: parse `r.table("t").map(function(t){ return t("date") }).min()` -> MAP then zero-argument MIN
- [x] Test: parse `r.table("t").avg("a","b")` -> error (at most one field argument)
- [x] Integration test: `min()` and `max()` with no arguments over a seeded numeric column return the extreme values
- [x] Integration test: `min({index:"idx"})` and `max({index:"idx"})` on a seeded secondary index return whole documents
- [x] Integration test: `sum(x => x("balance")("amount"))` over nested fields returns the expected total
- [x] Implement: add a shared `chainAggregate` over `parseArgListWithOpts`; register it for `min`, `max`, `sum`, `avg`
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + `chainAggregate(name, build)` rejects more than one positional argument at parse time (with a byte position) instead of letting the builder's deferred `errTerm` surface later, matching how `r.branch` and `r.object` validate
- + Extra coverage beyond the listed cases: zero-argument `sum()`/`avg()`, `min("f",{index:"i"})` (field plus optargs), `sum("a","b")` and `min("a",)` errors
- + Added `parseRunAtom` helper in `internal/integration/parser_test.go` for single-atom results

### Task 5: Builder -- funcWrap for all function-position arguments

- [x] Test: `reql.Table("t").Map(reql.Row().Bracket("a"))` -> MAP with FUNC-wrapped BRACKET
- [x] Test: `reql.Table("t").ConcatMap(reql.Row().Bracket("a"))` -> CONCAT_MAP with FUNC
- [x] Test: `reql.Table("t").ForEach(reql.Row().Bracket("a"))` -> FOR_EACH with FUNC
- [x] Test: `reql.Table("t").Reduce(reql.Row())` -> REDUCE with FUNC
- [x] Test: `reql.Table("t").Contains(reql.Row().Bracket("a"))` -> CONTAINS with FUNC
- [x] Test: `reql.Table("t").OffsetsOf(reql.Row().Bracket("a"))` -> OFFSETS_OF with FUNC
- [x] Test: `reql.Table("t").OrderBy(reql.Row().Bracket("a"))` -> ORDER_BY with FUNC
- [x] Test: `reql.Table("t").Update(reql.Row().Bracket("a"))` -> UPDATE with FUNC
- [x] Test: `reql.Table("t").Replace(reql.Row().Bracket("a"))` -> REPLACE with FUNC
- [x] Test: `reql.Table("t").Map(reql.Func(reql.Var(1), 1))` -> explicit FUNC passed through unchanged
- [x] Test: `reql.Table("t").Map(reql.Datum("a"))` -> plain datum passed through unchanged (no regression)
- [x] Test: `reql.Table("t").Map(reql.Func(reql.Row(), 1))` -> error term (IMPLICIT_VAR inside nested FUNC)
- [x] Implement: apply `wrapImplicitVar` in `Map`, `ConcatMap`, `ForEach`, `Reduce`, `Contains`, `OffsetsOf`, `OrderBy` field arguments, `Update`, `Replace`; return `errTerm` on wrap error, matching `Filter`
- [x] Run `go test ./internal/reql/... -race -count=1` -- must pass before next task
- + Added `funcWrap(v interface{}) (Term, error)` helper in `term.go` (`wrapImplicitVar` composed with `toTerm`) and routed `Filter`, `Group` and the aggregations through it
- + `OrderBy` now reuses `splitOptArgs` instead of its own trailing-OptArgs branch
- + Extra coverage beyond the listed cases: the nested-FUNC error path asserted for all nine methods, not only `Map`
- + Ran `go test -tags integration ./internal/integration/... -race -count=1` as a regression check even though the task is builder-only; passes

### Task 6: Builder -- term-valued arguments, variadic `Slice`, top-level `TableList`, chain `Branch`

- [x] Test: `reql.Table("t").GetField("a")` -> `[31,[[15,["t"]],"a"]]` (no regression)
- [x] Test: `reql.Table("t").GetField(reql.Var(1))` -> GET_FIELD with a VAR argument
- [x] Test: `reql.Table("t").Bracket(reql.Var(1))` -> `[170,[[15,["t"]],[10,[1]]]]`
- [x] Test: `reql.Datum("x").Match(reql.Var(1))` -> MATCH with a VAR argument
- [x] Test: `reql.Table("t").Slice(0,2)` -> `[30,[[15,["t"]],0,2]]` (no regression)
- [x] Test: `reql.Table("t").Slice(-2)` -> `[30,[[15,["t"]],-2]]`
- [x] Test: `reql.Table("t").Slice()` and `reql.Table("t").Slice(1,2,3)` -> error terms
- [x] Test: `reql.TableList()` -> `[62,[]]`
- [x] Test: `reql.DB("d").TableList()` -> `[62,[[14,["d"]]]]` (no regression)
- [x] Test: `reql.Datum(true).Branch(1, 2)` -> `[65,[true,1,2]]`
- [x] Test: `reql.Datum(true).Branch(1)` -> error term (even number of branches required)
- [x] Implement: widen `GetField`, `Bracket`, `Match` to `interface{}` with `toTerm`; change `Slice` to `Slice(bounds ...int)` accepting 1 or 2 values; add top-level `TableList()`; add `(t Term) Branch(branches ...interface{}) Term` delegating to the existing top-level `Branch` validation
- [x] Run `go test ./internal/reql/... -race -count=1` -- must pass before next task
- + Added `TestTermValuedArguments` and `TestSliceTableListBranchChain` in `term_test.go`
- + Extra coverage beyond the listed cases: `Bracket("a")` and `Match("\w+")` string regressions, and `Table("t").Count().Gt(0).Branch(Array(1), Array())` matching the chain-branch wire JSON from Technical Details
- + `(t Term) Branch` reports the delegated "odd number of arguments" message from the top-level `Branch`; the receiver counts as the first argument

### Task 7: Parser -- expressions where a string literal was required

- [x] Test: parse `r.table("t")("field")` -> BRACKET (no regression)
- [x] Test: parse `r.table("t")(0)` -> NTH (no regression)
- [x] Test: parse `r.table("t")(0.5)` -> error (float index still rejected)
- [x] Test: parse `r.table("t").map(function(f){ return f(f) })` -> BRACKET with VAR as the field argument
- [x] Test: parse `r.expr(["a","b"]).map(function(f){ return r.table("t").filter(function(o){ return o.hasFields(f) }).count() })` -> HAS_FIELDS with VAR
- [x] Test: parse `r.table("t").map(function(f){ return r.table("u").get(f).getField(f) })` -> GET_FIELD with VAR
- [x] Test: parse `r.table("t").map(function(f){ return f.match(f) })` -> MATCH with VAR
- [x] Test: parse `r.table("t").filter(f => f.hasFields(["a","b"]))` -> HAS_FIELDS with MAKE_ARRAY
- [x] Test: parse `r.table("t").pluck("a",{"p":["b","c"]})` -> PLUCK with nested selectors (no regression)
- [x] Test: parse `r.table("t").pluck("a")` -> PLUCK with a plain string (no regression)
- [x] Test: parse `r.table("t").map(function(f){ return r.table("u").without(f) })` -> WITHOUT with a lambda-parameter selector (see review correction below)
- [x] Integration test: the field-probing idiom from the log -- `r.expr([...field names...]).map(function(f){ return [f, r.table("t").filter(function(o){ return o.hasFields(f) }).count()] })` -- returns per-field counts
- [x] Integration test: `hasFields(["a","b"])` with an array selector matches only documents having both fields
- [x] Integration test: a lambda parameter used as a bracket key returns the referenced field value
- [x] Implement: extend `parseBracketArg` with a `parseExpr` fallback after the string and integer fast paths; move `getField` and `match` from `strArgChain` to `oneArgChain`; extend `parseOneFieldSelector` with `[` -> `parseDatumArray` and a `parseExpr` fallback
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + Deviation from Technical Details: the `parseExpr` fallback is not applied to every remaining token. `parseOneFieldSelector` still rejects number, bool and null literals, and `parseBracketArg` still rejects bool and null, because none of them can name a field or an index; this keeps the existing `pluck(123)` / `without(true)` / `t(true)` error tests meaningful
- + `parseBracketArg` error message widened to "expected string, integer or expression in bracket notation"; `TestParse_BracketNumericIndex_Errors` updated and a `t(null)` case added
- + Split the string and integer fast paths into `parseBracketLiteral` to stay under the cyclop limit of 10
- + Extra coverage beyond the listed cases: `getField("a")` and `match("^a")` string regressions, `withFields(["a","b"])`, and error cases `t()`, `getField()`, `hasFields("a",)`
- ! Second code review pass, verified against a live `rethinkdb:2.4.4` server: the originally listed `without(r.row("a"))` test asserted a broken query as correct. `pluck`/`without`/`hasFields`/`withFields` field selectors and bracket-notation keys are evaluated once per call, not once per document, so there is no implicit row for `r.row` to bind to; the server rejects the resulting query with "r.row is not defined in this context" regardless of what `Without`'s builder does with the term. `parseOneFieldSelector` and `parseBracketArg`'s `parseExpr` fallback now reject a parsed expression containing a bare `r.row` (`reql.ContainsImplicitVar`) instead of accepting it; the `without(r.row("a"))` test was replaced with an error-case assertion and the coverage for "field selector holding an expression" now uses a lambda parameter, which is the actual motivating pattern from the log (`hasFields(f)`, not `hasFields(r.row(...))`)
- ! Third code review pass: `getField` and `match` are the same class of position as bracket notation (a static field name / pattern, never funcWrap-ped), but were left on the plain `oneArgChain` helper so `getField(r.row("a"))` and `match(r.row("a"))` parsed a bare, unwrapped `IMPLICIT_VAR` that the server would reject at runtime. Fixed with a new `oneArgChainNoRow` helper (mirrors `parseBracketArg`'s `reql.ContainsImplicitVar` check, tagged with the existing `errBareRowInStaticValue` sentinel) used for both `getField` and `match`; regression tests confirm plain strings, lambda-parameter arguments and chained `getField(...).match(...)` still parse unchanged

### Task 8: Parser -- term-valued optargs and safe backtracking

- [x] Test: parse `r.table("t").orderBy({index: r.desc("d")})` -> `[41,[[15,["t"]]],{"index":[74,["d"]]}]`, an optarg and not a positional MAKE_OBJ argument
- [x] Test: parse `r.table("t").orderBy({index:"d"})` -> ORDER_BY with a string optarg (no regression)
- [x] Test: parse `r.table("t").between(1, 2, {index: r.desc("d")})` -> BETWEEN with a DESC optarg
- [x] Test: parse `r.table("t").between(r.minval, r.maxval, {index:"d", rightBound:"open"})` -> camelCase key still converted to `right_bound` (no regression)
- [x] Test: parse `r.table("t").changes({includeInitial: true})` -> CHANGES with a datum optarg (no regression)
- [x] Test: parse `r.table("t").getAll("a",{index: r.row("i")})` -> rejected (see review correction below); `orderBy`/`between` with `{index: r.desc("d")}` remain the covered "optarg holding an expression" case
- [x] Test: parse `r.table("t").map(x => x("a")).orderBy({index:"d"})` -> a lambda before an optargs backtrack; VAR ids stay 1-based and deterministic
- [x] Test: parse `r.table("t").filter(x => x("a")).orderBy(y => y("b"))` -> two sibling lambdas both using VAR(1) (no regression after the backtrack change)
- [x] Integration test: `orderBy({index: r.desc("idx")})` on a seeded secondary index returns rows in descending order, which fails today because the optarg is sent as a positional argument
- [x] Integration test: `orderBy({index: "idx"})` returns ascending order (no regression)
- [x] Integration test: `getAll` with an index optarg still resolves against the seeded index
- [x] Implement: add a `parseExpr` fallback to `parseOptArgValue`; in `tryTrailingOptArgs` snapshot and restore `pos`, `len(paramsStack)`, `nextVarID` and `depth`; update the stale safety comment
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + Added `assertWireJSON` helper in `parser_test.go` (exact wire-JSON string comparison) so the optarg slot is asserted directly, not through a builder-constructed term
- + Added `parseRunRows` (ordered `cur.All()`) and `rowIDs` helpers in `internal/integration/parser_test.go`
- + The `paramsStack` restore is guarded by a length check so a shorter stack cannot resurrect stale scopes through the shared backing array
- + Extra coverage beyond the listed cases: `orderBy({index: })` errors with a byte position, and `r.table("t").map(x => x("a").orderBy({index: y => y("b")}, "c"))` pins the inner lambda to VAR(2); without the `nextVarID` restore the re-parse allocated VAR(3)
- ! `parseFoldOpts` is now semantically a duplicate of `parseOptArgs` (both accept expression values). Left in place to keep this task's diff to the optargs path; a later cleanup can drop it (resolved in the first review pass: `parseFoldOpts` was removed and `chainFold` calls `parseOptArgs` directly)
- ! Second code review pass, verified against a live `rethinkdb:2.4.4` server: `getAll("a",{index: r.row("i")})` and `orderBy({index: r.row("i")})` parsed successfully but the server always rejects them ("r.row is not defined in this context") -- an optarg value has no per-row context, same reasoning as task 7's field selectors. `parseOptArgValue`'s `parseExpr` fallback now rejects a bare `r.row` via `reql.ContainsImplicitVar`. This could not be a plain returned error: `tryTrailingOptArgs` backtracks on any `parseOptArgs` failure and re-parses the object as a positional datum argument instead, which still parses (object/array literals accept arbitrary expression values) and silently reproduces the exact "silent optargs downgrade" bug this plan set out to fix, just with an unwrapped `r.row` baked into a datum instead of a misplaced positional argument. Fixed by tagging the rejection with a sentinel error (`errBareRowInStaticValue`) that `tryTrailingOptArgs` checks for and re-raises instead of backtracking
- ! Discovered but not fixed (out of scope for this pass): `wrapImplicitVar`/`replaceImplicit` do not recurse into native Go datum values (`map[string]interface{}`, `[]interface{}`, embedded `Term`), unlike `ContainsWrite`'s `datumContainsWrite`, which already handles exactly this shape. Confirmed against a live server that `filter({a: r.row("b")})` parses to `[39,[[15,["t"]],{"a":[170,[[13,[]],"b"]]}]]` -- the object's embedded `r.row` is never wrapped in FUNC even though `Filter` is funcWrap-covered, and the query fails the same way. No existing test asserts this as working and no production log entry uses this pattern, so it was left as a follow-up rather than risk a broad change to the shared `replaceImplicit` walk used by all 12 funcWrap call sites
- ! Post-merge review: the literal fast paths in `parseOptArgValue` (this task), `parseBracketArg`/`parseBracketLiteral` (task 7) and `parseOneFieldSelector` (task 7) returned as soon as they matched a string/number/bool/null token, without checking whether a chain or infix operator followed. A literal-led expression like `{multi: true.eq(true)}` stopped at `true`, failed to close the optargs object, and `tryTrailingOptArgs` silently backtracked it to a positional MAKE_OBJ argument -- the exact "silent optargs downgrade" this task set out to fix, just triggered by a different input shape. The bracket-notation and field-selector equivalents (`t("a".add("b"))`, `hasFields("a".add("b"))`) hit a hard parse error instead, since there is no positional fallback in those call sites. Fixed by only taking the fast path when the literal is immediately followed by the argument's delimiter (`)` for bracket notation, `,`/`)` for field selectors) and always parsing optarg values through `parseExpr()` (a plain literal marshals identically whether it comes back as a native Go value or as a termType-0 Term)
- ! Post-merge review: the `parseExpr` fallback added above (line 353) let `parseOneFieldSelector`/`parseOptArgValue` reach `parseObjectTerm` for a `{...}` value whose contents fail `parseDatumObject` (any non-datum expression, e.g. `r.row(...)`). `parseObjectTerm` stores such a value as a native `Term` inside a `map[string]interface{}`, then wraps the whole map in `reql.Datum` (termType 0). The `reql.ContainsImplicitVar` guard these two call sites rely on is built on `replaceImplicit`, which returns immediately for a termType-0 term without inspecting `t.datum` -- the same gap already noted at line 352 for `wrapImplicitVar`, but here it defeats the rejection check itself instead of just skipping a FUNC-wrap. `hasFields({a: r.row("x")})` and `orderBy({index: {a: r.row("x")}})` parsed successfully and failed only server-side with "r.row is not defined in this context", the exact class of bug this plan set out to eliminate. Fixed by rewriting `ContainsImplicitVar` to walk `t.datum` recursively for `Term`/`map[string]interface{}`/`[]interface{}` values, mirroring `ContainsWrite`'s `datumContainsWrite`; `replaceImplicit`/`wrapImplicitVar` themselves are untouched, so the line-352 deferral for the actual FUNC-wrap positions still stands

### Task 9: Parser -- `r.tableList`, `.branch`, single-argument `slice`, zero-parameter functions

- [x] Test: parse `r.tableList()` -> `[62,[]]`
- [x] Test: parse `r.db("d").tableList()` -> `[62,[[14,["d"]]]]` (no regression)
- [x] Test: parse `r.tableList("x")` -> error (no arguments accepted)
- [x] Test: parse `r.table("t").count().gt(0).branch([1],[])` -> `[65,[[21,[[43,[[15,["t"]]]],0]],[2,[1]],[2,[]]]]`
- [x] Test: parse `r.row("a").branch("yes","no")` -> BRANCH with the receiver as the test
- [x] Test: parse `r.row("a").branch("only")` -> error (even number of branches required)
- [x] Test: parse `r.table("t").slice(-2)` -> `[30,[[15,["t"]],-2]]`
- [x] Test: parse `r.table("t").slice(0,2)` -> `[30,[[15,["t"]],0,2]]` (no regression)
- [x] Test: parse `r.table("t").slice()` -> error
- [x] Test: parse `r.table("t").group(function(){ return true })` -> GROUP with `[69,[[2,[]],true]]`
- [x] Test: parse `r.table("t").map(() => 1)` -> FUNC with an empty parameter list
- [x] Test: parse `r.table("t").filter(x => x("a"))` -> single-parameter lambda unchanged (no regression)
- [x] Integration test: `r.tableList()` lists the tables of the default database selected via `--db`
- [x] Integration test: `.branch()` chain form returns each side depending on the condition
- [x] Integration test: `slice(-2)` over a seeded array field returns its last two elements
- [x] Integration test: `group(function(){ return true })` groups the whole table into one bucket
- [x] Implement: register `tableList` in `buildRBuilders` and `branch` in the chain builders; rewrite `chainSlice` to accept one or two integers; allow an empty parameter list in `parseLambdaParams`
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + Replaced the now-unused `parseTwoInts` with `parseIntArgs` (variadic integer argument list) used by `chainSlice`; a trailing comma is rejected with the standard "trailing comma in argument list" message
- + `chainBranch` validates an even argument count (at least 2) at parse time with a byte position, instead of letting the builder's deferred `errTerm` surface later, matching `chainAggregate` and `r.branch`
- + Dropped the two obsolete error tests asserting "lambda requires at least one parameter" (`() => 1`, `function(){ return 1 }`); both forms are valid now
- + The `slice(-2)` integration test seeds its array field with `reql.Array(...)`: a native `[]interface{}` value inside a seeded document is sent as a bare JSON array and rejected by the server as a raw term
- + Extra coverage beyond the listed cases: multi-condition chain `branch`, and the `slice(0,)`, `slice(0.5)` and `slice(0,1,2)` error paths

### Task 10: Lexer -- arithmetic operator tokens and assignment token

- [x] Test: tokenize `1+2` -> number, plus, number
- [x] Test: tokenize `a-2` -> ident, minus, number
- [x] Test: tokenize `f(-2)` -> ident, lparen, number(-2), rparen
- [x] Test: tokenize `[1,-2]` -> lbracket, number, comma, number(-2), rbracket
- [x] Test: tokenize `x(0)-1` -> minus after rparen is an operator
- [x] Test: tokenize `x[0]-1` and `x}-1` -> minus after rbracket and rbrace is an operator
- [x] Test: tokenize `60*60*24/2%7` -> star, star, slash, percent tokens
- [x] Test: tokenize `var x = 1` -> ident, ident, assign, number
- [x] Test: tokenize `x => y` -> arrow (no regression)
- [x] Test: tokenize `-5` at input start -> negative number literal
- [x] Implement: add `tokenPlus`, `tokenMinus`, `tokenStar`, `tokenSlash`, `tokenPercent`, `tokenAssign` with names in `tokenNames`; make `-` context-sensitive on the previously emitted token; emit `tokenAssign` from `readArrow` when `>` does not follow
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task
- + Deviation from Technical Details: the statement keywords `return`, `var`, `let` and `const` are excluded from the "previous token ends an expression" set, so `return -1` and `var x = -1` keep lexing the `-` as a number sign. Without the exception `function(p){ return -1 }` would break
- + `next` split into `next` (records the previous token) and `scan` (produces it); `readArrow` no longer returns an error, so its signature dropped the error result
- + `TestLexer_EqualAloneError` and `TestLexer_DoubleEqualError` replaced by token assertions: `=` is one assign token and `==` is two. The rejection moved to the parser, covered by the new `TestParse_AssignToken_Errors` (`r.expr(1==2)` -> `expected ')', got "="`, with a byte position)
- + Extra coverage beyond the listed cases: minus after a string literal, and negative literals after `=`, `=>` and another `-` (`1--1`)

### Task 11: Parser -- infix arithmetic with precedence

- [x] Test: parse `r.expr(1+2)` -> `[24,[1,2]]`
- [x] Test: parse `r.expr(60*60*24)` -> `[26,[[26,[60,60]],24]]`
- [x] Test: parse `r.expr(1+2*3)` -> ADD of 1 and MUL of 2 and 3 (precedence)
- [x] Test: parse `r.expr((1+2)*3)` -> MUL of ADD and 3 (grouping wins)
- [x] Test: parse `r.expr(10-2-3)` -> left-associative SUB
- [x] Test: parse `r.expr(10/2%3)` -> left-associative DIV then MOD
- [x] Test: parse `r.now().sub(60*60*24*30)` -> SUB with a folded MUL chain
- [x] Test: parse `r.table("t").between(1779222884700, 1779222884700+1, {index:"d"})` -> BETWEEN with an ADD upper bound
- [x] Test: parse `r.table("t").filter(x => x("a").add(1))` -> method form still works (no regression)
- [x] Test: parse `r.expr(1+)` -> error with a byte position
- [x] Integration test: `r.expr(60*60*24*30)` evaluates to 2592000 on the server
- [x] Integration test: `between` with an arithmetic upper bound over a seeded timestamp index returns the same rows as the pre-computed literal
- [x] Integration test: a filter using `r.now().sub(60*60*24)` selects only recent seeded documents
- [x] Implement: split `parseExpr` into `parseAdditive` / `parseMultiplicative` / `parsePostfix`; keep the depth guard in `parseExpr`; map operators to `Add`, `Sub`, `Mul`, `Div`, `Mod`
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + The two precedence levels share one `parseBinary(ops, next)` helper driven by an `infixOps` map of method expressions (`reql.Term.Add` and friends), instead of a duplicated loop per level
- + `between(...).orderBy("id")` over a selection returns an atom array, not a stream, so the integration test maps the ids server-side and decodes them through a new `atomStrings` helper in `internal/integration/parser_test.go`
- + Extra coverage beyond the listed cases: `r.expr(1-2)`, `r.expr(1 - -2)` (operator then negative literal), `r.table("t").count()+1` (chained term as an operand), and the error paths `r.expr(60*)`, `r.expr(*2)` and `r.expr(1) +`

### Task 12: Parser -- local variable statements in function bodies

- [x] Test: parse `r.table("t").filter(function(p){ var re = "x"; return p("id").match(re) })` -> MATCH with the inlined string
- [x] Test: parse `function(a){ var b = a("x"); return b.add(1) }` -> the bound term inlined into ADD
- [x] Test: parse `function(a){ var b = a("x"); return b.add(b) }` -> the subtree duplicated at both use sites
- [x] Test: parse `function(a){ var b = 1; var c = 2; return a("x").add(b).add(c) }` -> multiple bindings
- [x] Test: parse the `let` and `const` forms -> same result as `var`
- [x] Test: parse `function(a){ var b = function(c){ return c }; return a.map(b) }` -> a lambda bound to a local
- [x] Test: parse `function(a){ var a = 1; return a }` -> local shadows the parameter
- [x] Test: parse `function(a){ var b = 1 return b }` -> error (missing semicolon)
- [x] Test: parse `function(a){ var true = 1; return a }` -> error (reserved name)
- [x] Test: parse `function(a){ return a("x") }` -> body without locals unchanged (no regression)
- [x] Test: parse the full log entry 1 expression (`concatMap` with a `var sw = ...` binding) -> parses without error
- [x] Integration test: `filter(function(p){ var re = "^a"; return p("name").match(re) })` selects only the seeded documents whose name matches
- [x] Integration test: a local bound to a subquery and used twice in the same body returns the same result as the manually inlined form
- [x] Implement: add a locals scope stack to the parser struct pushed and popped with function scopes; parse `var|let|const <ident> = <expr> ;` before the optional `return`; resolve locals in `parseIdentPrimary` ahead of parameters and `r.*` dispatch
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + `parseFunctionExpr` now pushes the parameter scope before `{` and delegates the body to a new `parseBlockBody` helper, so parameters are visible inside the binding expressions and task 13 can reuse the same statement parser for arrow block bodies
- + `localsStack []map[string]reql.Term` is pushed and popped inside `pushScope`/`popScope`, so every lambda form keeps it aligned with `paramsStack`; `tryTrailingOptArgs` truncates it on backtrack alongside `paramsStack`
- + Locals resolve through the whole scope stack, so a binding is also visible inside nested lambdas of the same body; covered by `local_visible_in_nested_lambda`
- + Extra coverage beyond the listed cases: `local_out_of_scope_after_function` (a sibling lambda reusing the local's name sees the parameter, not the binding), and the error paths `var b;` (missing `=`), `var = 1;` (missing name), `var b = ;` (missing value) and `var const = 1;` (keyword as a name)
- ! Follow-up code review: `reservedLocalNames` (added by this task for `var|let|const` bindings) was never applied to `validateLambdaParam`, which still only rejected `return`/`function` as a lambda/function parameter name. `function(var){ return var }` and `(let) => let` parsed successfully with `var`/`let`/`const` bound as an ordinary parameter name. Fixed by having `validateLambdaParam` reuse `reservedLocalNames` instead of its own ad hoc two-word check. Note: this does not change the lexer's `stmtKeywords`/`prevEndsExpr` heuristic (task 10), so a reserved word immediately followed by a spaced arithmetic operator (e.g. `(var) => var - 1`) still errors, just with the lexer's generic "unexpected character" message rather than a "reserved word" one -- the same ambiguity already existed for `return - 1` (vs. the supported `return -1` literal) before this task and is unrelated to parameter validation; left unchanged since fixing it would require lexing to be context-aware of grammar position, not just the previous token's text

### Task 13: Parser -- arrow lambda block bodies

- [x] Test: parse `r.table("t").map(g => { return {a: g("b")} })` -> FUNC returning a datum object
- [x] Test: parse `r.table("t").map(g => { var x = g("b"); return {a: x} })` -> block body with a local
- [x] Test: parse `r.table("t").map(g => {a: g("b")})` -> object literal, unchanged behaviour (no regression)
- [x] Test: parse `r.table("t").map(g => ({a: g("b")}))` -> parenthesized object literal (no regression)
- [x] Test: parse `r.table("t").map((x, y) => { return x.add(y) })` -> multi-parameter arrow with a block body
- [x] Test: parse `r.table("t").map(g => { return })` -> error
- [x] Test: parse the full log entry 54 expression (`group`/`ungroup`/`map(g => {return {...}})`) -> parses without error
- [x] Integration test: `map(g => { return {id: g("id"), n: g("name").upcase()} })` reshapes seeded documents as expected
- [x] Integration test: the log entry 54 pipeline (`group`, `ungroup`, `map` with a block body, `orderBy`) runs against seeded data
- [x] Implement: in the arrow lambda parsers, treat `{` as a block only when the next token is the identifier `return`, `var`, `let` or `const`; reuse the statement parser from task 12
- [x] Run `go test ./internal/reql/parser/... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1` -- must pass before next task
- + Both arrow forms route through a new `parseArrowBody` helper (`parseLambda` for the parenthesized form, `parseBareArrowLambda` for the bare one), so the block heuristic lives in one place; the keyword set reuses `localKeywords` from task 12 plus `return`
- + The block body reuses `parseBlockBody` unchanged, so `return` stays optional inside a block: `g => { var x = g("b"); x }` is valid and covered
- + Extra coverage beyond the listed cases: `(g) => { return g("b") }` (single parenthesized parameter), the no-`return` block form, and the error paths `g => { return g("b") )` (unterminated block) and `g => { var x = 1; }` (binding without a body)

### Task 14: Parser -- actionable messages for unsupported input

- [x] Test: parse `r.table("t").between(["A", new Date("2026-06-19T07:40:13.981Z").getTime()], ["A", r.maxval], {index:"i"})` -> error mentioning `r.iso8601` and `r.epochTime`
- [x] Test: parse `table("x").count()` -> error suggesting `r.table(...)`
- [x] Test: parse `db("x").tableList()` -> error suggesting `r.db(...)`
- [x] Test: parse `r.table("x").count(); r.table("y").count()` -> error mentioning one query at a time and `--file` with `---` separators
- [x] Test: parse `notAKnownName("x")` -> the generic unexpected-token error, unchanged (no regression)
- [x] Test: every new error message includes a byte position
- [x] Implement: special-case `new` before `Date`, a known builder name used without the `r.` prefix, and a top-level `;` followed by more input
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task
- + The identifier fallback of `parseIdentPrimary` now goes through a new `parseUnknownIdent(tok)` helper instead of calling `parseDatumTerm` directly, so the hints do not push `parseIdentPrimary` over the cyclop limit of 10; the fallback is reachable only for `tokenIdent`, where `parseDatumTerm` always errored anyway
- + Added a `peekAt(offset)` helper (EOF-safe lookahead) used by the `new Date` and multi-statement checks
- + The missing-prefix hint keys off `rBuilders`, so every top-level builder name is covered, not only `table` and `db`; the suggestion is uniformly `r.<name>(...)` even for the paren-less `r.minval` / `r.maxval`
- + A lone trailing `;` (`r.table("x").count();`) keeps the generic unexpected-token error: the multi-statement hint fires only when a non-EOF token follows the `;`
- + No integration coverage for this task: it only produces parse-time error messages, so nothing reaches the server. `go test ./... -race -count=1` and `make build` were run as regression checks

### Task 15: Fuzz corpus

- [x] Test: fuzz parser does not panic on group and aggregation seeds
- [x] Test: fuzz parser does not panic on arithmetic seeds
- [x] Test: fuzz parser does not panic on local variable and block body seeds
- [x] Test: fuzz parser does not panic on term-valued optargs seeds
- [x] Seed corpus: `r.table("t").group("a","b")`, `r.table("t").group(x => x("a"))`, `r.table("t").group()`, `r.table("t").min()`, `r.table("t").max({index:"d"})`, `r.table("t").sum(x => x("v"))`, `r.tableList()`, `r.table("t").slice(-2)`, `r.expr(1+2*3)`, `r.expr(1+)`, `r.expr(-)`, `1--1`, `function(a){ var b = 1; return b }`, `function(a){ var b = ; return b }`, `g => { return {a: 1} }`, `g => { return }`, `r.table("t").orderBy({index: r.desc("d")})`, `r.table("t").filter(f => f.hasFields(["a"]))`, `r.table("t")(0)(0)`, `r.row("a").branch(1,2)`
- [x] Implement: extend the `FuzzParse` seed corpus in `fuzz_test.go`
- [x] Run `go test ./internal/reql/parser/... -race -count=1` -- must pass before next task
- + Extra seeds beyond the listed corpus: the error forms of each new syntax (`group("a",{index:"i"})` opts, `avg("a","b")`, `tableList("x")`, `slice()`, `branch("only")`, `r.expr(*2)`, `r.expr(1==2)`, `orderBy({index: })`, `var b = 1 return b`, `var true = 1`, `g => { var x = 1; }`), plus the task 14 hint inputs (`new Date(...)`, `table("x").count()`, a `;`-separated pair)
- + Ran `go test ./internal/reql/parser/ -fuzz=FuzzParse -fuzztime=30s` beyond the seed run: 15.8M executions, 315 new interesting inputs, no crash

### Task 16: Verify acceptance criteria

- [x] Verify every root cause listed in Overview items 1 through 11 parses, using the original expressions from `~/.r-cli/parser-errors.log`
- [x] Verify `orderBy({index: r.desc(...)})` produces an optarg rather than a positional MAKE_OBJ argument
- [x] Verify funcWrap is applied in every function-position method listed in Technical Details
- [x] Verify the three unsupported cases produce the new actionable messages
- [x] Verify every parser-level task (3, 4, 7, 8, 9, 11, 12, 13) left integration coverage in `internal/integration`
- [x] Update `CLAUDE.md` package descriptions for `internal/reql` and `internal/reql/parser` to match the new signatures and syntax
- [x] Run `go test ./... -race -count=1`
- [x] Run `go test -tags integration ./internal/integration/... -race -count=1`
- [x] Run `make build` -- linter issues must all be fixed
- + The verification is a committed regression corpus, not a one-off check: `internal/reql/parser/production_log_test.go` replays all 80 distinct log expressions. `TestParse_ProductionLog_Accepted` requires the 72 in-scope ones to parse, grouped by root cause with the log entry numbers; `TestParse_ProductionLog_Rejected` pins the hint substrings and byte position for the 7 out-of-scope ones. Tables are package-level vars so the 80-expression corpus does not trip `funlen`
- + Log entry 34 is the one recorded expression that still fails, and by design: it uses `r.row("date")` inside a `do(function(uid){...})` body, which the documented scope rule rejects. Entry 35 is the same query rewritten with the parameter form and parses. Covered by `TestParse_ProductionLog_RowInsideLambda`
- ! The scope error for entry 34 reads "r.row inside arrow function is ambiguous; use the arrow parameter instead" even for the `function(p){...}` form. Wording only, left unchanged to keep this task read-only over the parser
- + funcWrap coverage confirmed by call site, not only by test: `funcWrap` is called from exactly 12 places in `term.go` -- `Filter`, `Update`, `Replace`, `OrderBy`, `Map`, `Reduce`, `Group`, `aggregate` (the shared `Min`/`Max`/`Sum`/`Avg` body), `OffsetsOf`, `ForEach`, `ConcatMap`, `Contains` -- which is every method listed in Technical Details
- + Integration coverage confirmed per commit: each of the eight parser-level commits (tasks 3, 4, 7, 8, 9, 11, 12, 13) touches `internal/integration/parser_test.go`, which now holds 29 test functions
- + `orderBy({index: r.desc("d")})` asserted as an exact wire string `[41,[[15,["t"]]],{"index":[74,["d"]]}]` in `TestParse_OptArgs_TermValued`, and end to end by `TestParserOrderByIndexOptArgs`
- + `CLAUDE.md` corrections beyond the new syntax: the `internal/reql` entry claimed funcWrap applied only to `Filter`, and the `internal/reql/parser` entry claimed OptArgs values were restricted to datum literals and that `parseFoldOpts` differed from `parseOptArgs`. All three were stale after tasks 5 and 8
- ! Post-merge review: task 7's and 8's fixes for literal-led expressions (line 353) only covered the string case end to end. `parseBracketArg` still rejected a bare `bool`/`null` unconditionally instead of only when it was the whole argument, so `t(true.branch("a","b"))` and `t(null.default("a"))` hit a hard parse error while the parenthesized-grouping equivalent `t((true.branch("a","b")))` parsed fine. `parseOneFieldSelector` had the same gap for `number`/`bool`/`null`, and additionally locked in `{...}`/`[...]` as a data selector as soon as they parsed as a literal, without checking for a following chain/operator, so `hasFields(["a"].nth(0))` and `hasFields(null.default("a"))` failed the same way. Verified against a live `rethinkdb:2.4.4` server that the parenthesized forms return the expected results. Fixed by extending the "immediately followed by the argument delimiter" check to bool/null in both functions, and adding the same continuation check (via a new `tryDatumSelectorLiteral` helper, restoring parser position on non-commit) to the object/array fast paths in `parseOneFieldSelector`

## Post-Completion

*Items requiring manual intervention - no checkboxes, informational only*

- Re-run the failing expressions from `~/.r-cli/parser-errors.log` against a real database and confirm the results, not just that they parse. Parsing is necessary but not sufficient: `between(..., {index: r.desc(...)})` is accepted by the parser after task 8 but may still be rejected by the server, since `between` expects a string index name.
- Consider rotating or archiving `~/.r-cli/parser-errors.log` after the release so that the next analysis starts from a clean window.
