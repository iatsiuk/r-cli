# Plan: Expression Arguments For desc/asc And indexCreate

## Overview

Fix the parser gaps recorded in `~/.r-cli/parser-errors.log` since version 1.3.0 (3 entries, 2 distinct root causes) so that real-world ReQL expressions pasted from the RethinkDB Data Explorer parse correctly.

1. **`r.desc(...)` / `r.asc(...)` accept only a string literal** (2 entries, 2026-08-14, `expected string literal, got "acc" at position 114`). Official drivers apply `funcWrap` to the argument, so `r.desc("a")`, `r.desc(r.row("a"))` and `r.desc(d => d("a"))` are all valid. Only the first form parses today.
2. **`.abs()`** (1 entry, 2026-08-17, `unknown method .abs at position 1467`). There is no `ABS` term in the RethinkDB wire protocol -- 183 to 185 are `FLOOR`/`CEIL`/`ROUND`, 186 to 190 are `VALUES`/`FOLD`/`GRANT`/`SET_WRITE_HOOK`/`GET_WRITE_HOOK`, 191 to 196 are the bitwise operators. This is not a gap in r-cli, so it is handled by an actionable error message pointing at `r.branch(x.lt(0), x.mul(-1), x)`, not by adding a term.

One related gap is folded in because it is the same code path and the same class of defect:

- **`indexCreate` accepts only a string index name**, so functional secondary indexes are unreachable: `indexCreate("full", function(d){ return d("a").add(d("b")) })` and `indexCreate("m", r.row("a"), {multi: true})` both fail. Not present in the log; found while auditing the other `parseOneStringArg` call sites.

The remainder of the third log entry -- a 1400-character `getAll`/`filter`/`group`/`map`/`reduce`/`ungroup` chain with nested `r.branch` and `function(t){...}` bodies -- already parses; `.abs()` was its only blocker.

Package: `internal/reql`, `internal/reql/parser`

## Context

- Files involved: `internal/reql/term.go` (builders), `internal/reql/term_test.go`, `internal/reql/parser/parser.go`, `internal/reql/parser/parser_test.go`, `internal/reql/parser/production_log_test.go`, `internal/reql/parser/fuzz_test.go`, `internal/integration/parser_test.go`
- Current state: `parseRDesc` and `parseRAsc` (`parser.go:595-609`) call `parseOneStringArg`; `Asc(field string)` and `Desc(field string)` (`term.go:331-338`) wrap the argument in `Datum` with no `funcWrap`
- `indexCreate` is registered through `strArgChainWithOpts` (`parser.go:1885`); `IndexCreate(name string, opts ...OptArgs)` lives at `term.go:589`
- Term types already present in `internal/proto/term.go`: ASC 73, DESC 74, INDEX_CREATE 75, ORDERBY 41, FUNC 69, VAR 10, IMPLICIT_VAR 13, MAKE_ARRAY 2, BRACKET 170, ADD 24
- Existing patterns to reuse: `funcWrap` / `splitOptArgs` / `errTerm` in `term.go`; `parseOneArg`, `parseArgListWithOpts` and the `noArgChain` / `oneArgChain` / `strArgChainWithOpts` registration helpers in `parser.go`; the `new Date(...)` and missing-`r.`-prefix hints (`parser.go:283-288`) as the model for the `.abs()` message
- `Asc` and `Desc` have exactly two non-test callers, both in `parser.go`; `IndexCreate` has two non-test callers in `cmd/r-cli/index.go:70-72`. Widening to variadic `...interface{}` keeps every existing call site source-compatible
- `funcWrap` is a no-op when the argument contains no `IMPLICIT_VAR`, so `r.desc("a")` and `indexCreate("i")` must keep producing byte-identical wire JSON
- `r.desc` and `r.asc` must not join the `errBareRowInStaticValue` set: their argument is funcWrap-ped, so a bare `r.row` there is legal and must be accepted
- Adopted from the analysis of `~/.r-cli/parser-errors.log`, entries dated 2026-08-14 and 2026-08-17, version 1.3.0

## Development Approach

- **Testing approach**: strict TDD -- write the test first, run it, confirm it fails for the expected reason, only then write the implementation
- **CRITICAL: never write implementation code before the failing test exists** -- a task whose tests were written after the code must be redone
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: every parser-level task MUST also ship integration coverage** in `internal/integration`, following the existing style of that directory, and that integration test is written **before** the parser change like every other test -- it must be seen failing on the parse error first
- **CRITICAL: all tests must pass before starting next task** -- no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- Builder change first, parser change second, so each layer is verified independently
- Maintain backward compatibility: existing expressions must produce byte-identical wire JSON

## Testing Strategy

- **Unit tests**: required for every task -- success and error scenarios, asserted on exact wire JSON
- **Regression tests**: each task that widens a signature must keep a test for the old narrow form
- **Integration tests**: required in every parser-level task, written **before** the parser change like any other test, added under the `integration` build tag and run against a live `rethinkdb:2.4` container via testcontainers. Wire JSON proves the query shape; the integration test proves the server accepts it and returns the expected rows
  - Query-shape tasks go into `internal/integration/parser_test.go`, reusing the existing `newExecutor`, `setupTestDB`, `createTestTable`, `seedTable`, `waitForIndex`, `atomRows` and `closeCursor` helpers
  - Error-message tasks go into `internal/integration/cli_test.go`, reusing `cliRun` and `cliArgs` and asserting stderr plus the exit code, in the style of `TestCLIBadHost`. A parse error never reaches the server, so the CLI boundary is the only meaningful end-to-end assertion for it
- **Builder-only tasks**: unit tests are sufficient, because the parser task that consumes the builder carries the integration coverage
- **Regression corpus**: the log expressions are added to `internal/reql/parser/production_log_test.go` alongside the existing corpus, accepted ones in the accepted table and `.abs()` in the rejected table with its pinned hint substring
- **Fuzz tests**: dedicated task extends the seed corpus with the new syntax and its error forms

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Technical Details

### desc / asc

Current: `Asc(field string)` and `Desc(field string)` at `term.go:331-338`; `parseRDesc` / `parseRAsc` at `parser.go:595-609` restrict the argument to `tokenString` via `parseOneStringArg`.

Builder: `Asc(field interface{}) Term` and `Desc(field interface{}) Term`, each routing the argument through `funcWrap` and returning `errTerm(err)` on a wrap error, mirroring `OrderBy` (`term.go:346-352`).

Parser: replace `parseOneStringArg` with the existing `parseOneArg`, which parses `( expr )` and returns a Term. A string literal comes back as a datum Term (termType 0) that marshals to the bare string, so the narrow form is unchanged.

Wire JSON:

- `r.table("t").orderBy(r.desc("a"))` -> `[41,[[15,["t"]],[74,["a"]]]]` (unchanged)
- `r.table("t").orderBy(r.desc(d => d("a")))` -> `[41,[[15,["t"]],[74,[[69,[[2,[1]],[170,[[10,[1]],"a"]]]]]]]]`
- `r.table("t").orderBy(r.desc(r.row("a")))` -> identical to the lambda form, because `funcWrap` replaces `IMPLICIT_VAR` with `VAR 1`
- `r.table("t").orderBy(r.asc(function(d){ return d("a") }))` -> same shape with `73` in place of `74`
- `r.table("t").orderBy({index: r.desc("d")})` -> `[41,[[15,["t"]]],{"index":[74,["d"]]}]` (unchanged)

Because `funcWrap` leaves no `IMPLICIT_VAR` behind, `{index: r.desc(r.row("a"))}` does not trip the `errBareRowInStaticValue` check in `parseOptArgValue`. That is the correct parser behavior even though the server will reject a non-name index; pin it with a test so the interaction is not rediscovered later.

The log expression that motivates this task, reduced to its failing part:

```
r.db("restored").table("tmp_edge_check")
  .filter(acc => acc("balance")("express").default(0).gt(0))
  .orderBy(r.desc(acc => acc("balance")("express").sub(acc("balance")("amount"))))
  .pluck("id")
```

### abs

`ABS` does not exist in `ql2.proto` and is absent from `docs/protocol-spec.md`. Absolute value is expressed as `r.branch(x.lt(0), x.mul(-1), x)`.

Parser: in the unknown-method branch of `parseChain` (`parser.go:459-462`), look the method name up in a new package-level `unsupportedChainHints map[string]string` before falling back to the generic message. Seed it with `abs`. Message style follows the existing hints at `parser.go:283-288`, one sentence plus `at position %d`.

- `r.expr(-5).abs()` -> `.abs() is not a ReQL term, use r.branch(x.lt(0), x.mul(-1), x) at position 11`

The generic `unknown method .%s at position %d` message stays for every name not in the map.

### indexCreate

Current: `IndexCreate(name string, opts ...OptArgs)` at `term.go:589`; registered via `strArgChainWithOpts` at `parser.go:1885`, which accepts a string then optionally one `{...}` object.

Builder: `IndexCreate(name string, args ...interface{}) Term`. Peel a trailing `OptArgs` with `splitOptArgs`; at most one remaining argument is the index function, appended after `funcWrap`; more than one returns `errTerm`. The name stays a typed `string` -- it is always a literal identifier, never a per-document expression -- so `cmd/r-cli/index.go:70-72` continues to compile, `IndexCreate(args[1], opts)` binding `opts` to the `...interface{}` variadic.

Parser: replace the `strArgChainWithOpts` registration with a dedicated `chainIndexCreate` that expects a string literal name, then optionally an index function followed by a trailing OptArgs, in that order, built on `parseArgListWithOpts`. OptArgs before the index function is not supported and fails with "takes at most one index function".

Wire JSON:

- `r.table("t").indexCreate("i")` -> `[75,[[15,["t"]],"i"]]` (unchanged)
- `r.table("t").indexCreate("i",{multi:true})` -> `[75,[[15,["t"]],"i"],{"multi":true}]` (unchanged)
- `r.table("t").indexCreate("full", function(d){ return d("a").add(d("b")) })` -> `[75,[[15,["t"]],"full",[69,[[2,[1]],[24,[[170,[[10,[1]],"a"]],[170,[[10,[1]],"b"]]]]]]]]`
- `r.table("t").indexCreate("m", r.row("a"), {multi:true})` -> `[75,[[15,["t"]],"m",[69,[[2,[1]],[170,[[10,[1]],"a"]]]]],{"multi":true}]`

`camelToSnake` already runs over optarg keys, and `multi` / `geo` are unaffected by it.

## Implementation Steps

### Task 1: Widen Asc and Desc builders to any expression

- [x] write failing tests in `internal/reql/term_test.go` asserting exact wire JSON for `Desc("a")`, `Desc(Row("a"))`, `Desc(Func(...))`, and the same three for `Asc`
- [x] add a test that a wrap error inside `Desc` propagates as a deferred error through `MarshalJSON`
- [x] change `Asc` and `Desc` in `internal/reql/term.go` to take `interface{}` and route the argument through `funcWrap`, returning `errTerm(err)` on failure
- [x] update the two call sites in `internal/reql/parser/parser.go` so the package still builds (no edit needed: both pass a `string`, which satisfies `interface{}`)
- [x] verify `Desc("a")` still marshals to `[74,["a"]]` byte for byte
- [x] run `go test ./internal/reql/... -race -count=1` - must pass before next task

### Task 2: Parse r.desc and r.asc with an expression argument

- [x] write failing parser tests for `orderBy(r.desc(d => d("a")))`, `orderBy(r.desc(r.row("a")))`, `orderBy(r.asc(function(d){ return d("a") }))` and the nested-field form from the log entry
- [x] write a failing test that `orderBy({index: r.desc(r.row("a"))})` parses and lands in the optargs slot rather than being rejected as a bare `r.row`
- [x] keep regression tests for `r.desc("a")`, `r.asc("a")` and `orderBy({index: r.desc("d")})` asserting unchanged wire JSON
- [x] add error-case tests: `r.desc()` and `r.desc("a","b")` must still fail with a positioned message
- [x] write failing integration coverage in `internal/integration/parser_test.go`: seed a table, order by a lambda-derived key with `r.desc`, assert the returned row order
- [x] run the new integration test and confirm it fails on the parse error, not on the assertion
- [x] replace `parseOneStringArg` with `parseOneArg` in `parseRDesc` and `parseRAsc`
- [x] run `go test ./internal/reql/... -race -count=1` - must pass before next task
- [x] run `go test -tags integration ./internal/integration/... -race -count=1` - must pass before next task

### Task 3: Actionable error message for .abs()

- [x] write a failing test asserting the `.abs()` hint text and its byte position
- [x] write a test that an unrelated unknown method still produces the generic `unknown method .%s at position %d` message
- [x] write failing integration coverage in `internal/integration/cli_test.go` using `cliRun` and `cliArgs`: run an expression ending in `.abs()`, assert the hint on stderr and exit code 2, in the style of `TestCLIBadHost`
- [x] run the new integration test and confirm it fails on the old generic message
- [x] add the `unsupportedChainHints` lookup to the unknown-method branch of `parseChain` and seed it with `abs`
- [x] run `go test ./internal/reql/parser/... -race -count=1` - must pass before next task
- [x] run `go test -tags integration ./internal/integration/... -race -count=1` - must pass before next task

### Task 4: IndexCreate builder accepts an index function

- [x] write failing tests in `internal/reql/term_test.go` for `IndexCreate("i")`, `IndexCreate("i", OptArgs{...})`, `IndexCreate("i", Func(...))` and `IndexCreate("i", Row("a"), OptArgs{...})`, asserting exact wire JSON
- [x] write a test that more than one non-optargs argument yields a deferred error
- [x] change the signature to `IndexCreate(name string, args ...interface{})`, using `splitOptArgs` and `funcWrap`
- [x] verify `cmd/r-cli/index.go` still compiles without edits
- [x] run `go test ./internal/reql/... -race -count=1` - must pass before next task
- + the `strArgChainWithOpts` registration in `parser.go` could no longer spread `opts...` into the `...interface{}` variadic; wrapped in an explicit len check (Task 5 replaces this registration anyway)

### Task 5: Parse indexCreate with an index function

- [x] write failing parser tests for `indexCreate("full", function(d){ return d("a").add(d("b")) })`, `indexCreate("m", r.row("a"), {multi:true})` and `indexCreate("g", d => d("loc"), {geo:true})`
- [x] keep regression tests for `indexCreate("i")` and `indexCreate("i",{multi:true})` asserting unchanged wire JSON
- [x] add error-case tests: a non-string first argument and a trailing comma must fail with a positioned message
- [x] write failing integration coverage in `internal/integration/parser_test.go`: create a functional index, wait for it with `waitForIndex`, then query it through `getAll` and assert the rows
- [x] run the new integration test and confirm it fails on the parse error, not on the assertion
- [x] replace the `strArgChainWithOpts` registration with a dedicated `chainIndexCreate`
- [x] run `go test ./internal/reql/... -race -count=1` - must pass before next task
- [x] run `go test -tags integration ./internal/integration/... -race -count=1` - must pass before next task
- + `make build` lint fails on a pre-existing toolchain mismatch (golangci-lint cannot read Go 1.27 export data); the same failure reproduces on a clean tree

### Task 6: Regression corpus and fuzz seeds

- [x] add the two accepted log expressions verbatim to the accepted table in `internal/reql/parser/production_log_test.go`, labelled with their log dates
- [x] add the third log expression to the rejected table, pinning the `.abs()` hint substring, and add a note that the rest of that chain parses
- [x] extend the seed corpus in `internal/reql/parser/fuzz_test.go` with `r.desc(d => d("a"))`, `r.asc(r.row("a"))`, `indexCreate("i", d => d("a"), {multi:true})`, `.abs()`, `r.desc()`, `r.desc("a","b")` and `indexCreate(1)`
- [x] run `go test ./internal/reql/parser/ -run FuzzParse -race -count=1` over the seeds
- [x] run `go test ./internal/reql/parser/ -fuzz=FuzzParse -fuzztime=30s` and confirm no crash
- [x] run `go test ./internal/reql/parser/... -race -count=1` - must pass before next task

### Task 7: Verify acceptance criteria

- [x] verify each of the three original log expressions from `~/.r-cli/parser-errors.log` now parses, or fails only with the new `.abs()` hint (replayed every log line through `Parse`: entries 1 and 2 build wire JSON, entry 3 fails only with `.abs() is not a ReQL term, use r.branch(x.lt(0), x.mul(-1), x) at position 1467`)
- [x] verify `r.desc("a")`, `r.asc("a")`, `indexCreate("i")` and `indexCreate("i",{multi:true})` produce byte-identical wire JSON to before the change (also `orderBy({index: r.desc("d")})`; pinned in `parser_test.go:1705,2754-2764,2887-2892` and `term_test.go:1599-1629`)
- [x] verify no unrelated method lost its generic unknown-method message (`unsupportedChainHints` holds `abs` only; generic message pinned in `parser_test.go:216,802,2869`)
- [x] verify tasks 2, 3 and 5 each left integration coverage in `internal/integration` (`parser_test.go:1078,1088` for desc expressions, `cli_test.go:301` `TestCLIAbsHint`, `parser_test.go:1110` `TestParserIndexCreateFunction`)
- [x] verify Docker is available for testcontainers; the suite starts its own `rethinkdb:2.4` container and does not reuse a locally running server (Docker 29.7.2; shared `TestMain` container)
- [x] update the `internal/reql` and `internal/reql/parser` entries in `CLAUDE.md` to match the new signatures and syntax
- [x] run `go test ./... -race -count=1`
- [x] run `go test -tags integration ./internal/integration/... -race -count=1`
- [x] run `make build` - linter issues must all be fixed (fixed pre-existing gofmt drift in `internal/integration/string_time_join_test.go` and `type_ops_test.go`; `gofmt -l`, `go vet ./...` and `go vet -tags integration` are clean)
- ! `golangci-lint` still fails with 3 `typecheck` issues in `internal/wire` only: its x/tools cannot read Go 1.27 export data (`export data version 4 is greater than maximum supported version 2`). Reproduces on a clean tree and with `go run golangci-lint@v2.10.1` compiled by the local toolchain, so it is an environment limitation, not a code defect. Needs a golangci-lint release built against Go 1.27.

## Post-Completion

*Items requiring manual intervention - no checkboxes, informational only*

- Re-run the three log expressions against the real `restored` database and confirm the results, not just that they parse. Parsing is necessary but not sufficient: `orderBy(r.desc(<function>))` on a table without a matching index is a server-side sorting limit once the sequence exceeds 100k documents.
- Rewrite the `.abs()` query by hand as `r.branch(diff.lt(0), diff.mul(-1), diff).lt(0.005)` and confirm the VAT reconciliation classification it was computing.
- Consider rotating `~/.r-cli/parser-errors.log` after the release so the next analysis starts from a clean window.
