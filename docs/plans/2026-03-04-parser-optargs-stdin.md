# Fix parser OptArgs handling and -F stdin support

## Overview

Three categories of bugs discovered from Claude Code session transcripts:

1. **Parser chains using `parseArgList` + variadic Go API fail to pass OptArgs** -- `getAll`, `orderBy` (and `union`) parse `{key: val}` as `reql.Term` (datum wrapping `map[string]interface{}`, termType=0), but the Go API checks `.(OptArgs)` type assertion which fails on `Term`
2. **Parser chains using fixed-arity parsers ignore OptArgs entirely** -- `between`, `eqJoin`, `distance`, `getIntersecting`, `getNearest`, `tableCreate`, `indexCreate`, `changes`, `reconfigure` never parse optional `{...}` argument
3. **`-F -` doesn't read from stdin** -- `runQueryFile` calls `os.Open("-")` literally

Root cause for issues 1+2: the parser has no unified mechanism for "parse your required args, then optionally parse trailing OptArgs". Each chain handler either has ad-hoc OptArgs support or none at all.

## Context

- Parser chains: `internal/reql/parser/parser.go` (chain handlers + helper builders)
- Term API: `internal/reql/term.go` (method signatures with `OptArgs`)
- Query CLI: `cmd/r-cli/query.go` (`runQueryFile`)
- Parser tests: `internal/reql/parser/parser_test.go`
- Integration tests: `internal/integration/`

### Broken chains (full audit)

**Category A -- variadic `[]interface{}` with runtime OptArgs detection:**
| Chain | Parser handler | Problem |
|-------|---------------|---------|
| `getAll` | `chainGetAll` -- `parseArgList()` returns `[]reql.Term` | `.(OptArgs)` assertion fails on `Term` |
| `orderBy` | `chainOrderBy` -- `parseArgList()` returns `[]reql.Term` | same |

**Category B -- fixed-arity parsers, no OptArgs at all:**
| Chain | Parser handler | Go API |
|-------|---------------|--------|
| `between` | `parseTwoArgs()` | `Between(lo, hi, opts ...OptArgs)` |
| `eqJoin` | `parseStringThenArg()` | `EqJoin(field, table, opts ...OptArgs)` |
| `distance` | `oneArgChain` | `Distance(other, opts ...OptArgs)` |
| `getIntersecting` | `oneArgChain` | `GetIntersecting(geo, opts ...OptArgs)` |
| `getNearest` | `oneArgChain` | `GetNearest(point, opts ...OptArgs)` |
| `tableCreate` | `strArgChain` | `TableCreate(name, opts ...OptArgs)` |
| `indexCreate` | `strArgChain` | `IndexCreate(name, opts ...OptArgs)` |
| `changes` | `noArgChain` | `Changes(opts ...OptArgs)` |
| `reconfigure` | `noArgChain` | `Reconfigure(opts ...OptArgs)` |

**Already working (for reference):**
- `insert`, `update`, `delete` -- custom handlers with explicit `parseOptArgs()`
- `fold` -- custom handler with `parseFoldOpts()`
- `r.circle`, `r.random` -- custom top-level parsers

## Development Approach
- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**
- Run `make build` after each change (includes linter)

## Testing Strategy
- **Unit tests**: parser tests in `internal/reql/parser/parser_test.go` for every fixed chain
- **Integration tests**: `internal/integration/` for `getAll` with index, `between` with opts, `eqJoin` with opts
- **CLI tests**: `cmd/r-cli/query_test.go` for `-F -` stdin support

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix

## Implementation Steps

### Task 1: Add parser unit tests for broken OptArgs chains (TDD red phase)

Add test cases that currently fail, covering all broken chains:

- [x] add test: `getAll` with OptArgs -- `r.db("d").table("t").getAll("a", "b", {index: "idx"})`
- [x] add test: `orderBy` with OptArgs -- `r.db("d").table("t").orderBy("name", {index: "idx"})`
- [x] add test: `between` with OptArgs -- `r.db("d").table("t").between(1, 10, {index: "score", left_bound: "closed"})`
- [x] add test: `eqJoin` with OptArgs -- `r.table("t").eqJoin("field", r.table("t2"), {index: "idx"})`
- [x] add test: `distance` with OptArgs -- `term.distance(other, {unit: "km"})`
- [x] add test: `getIntersecting` with OptArgs -- `r.table("t").getIntersecting(point, {index: "geo"})`
- [x] add test: `getNearest` with OptArgs -- `r.table("t").getNearest(point, {index: "geo", max_dist: 1000})`
- [x] add test: `tableCreate` with OptArgs -- `r.db("d").tableCreate("t", {primary_key: "uid"})`
- [x] add test: `indexCreate` with OptArgs -- `r.db("d").table("t").indexCreate("idx", {multi: true})`
- [x] add test: `changes` with OptArgs -- `r.db("d").table("t").changes({include_initial: true})`
- [x] add test: `reconfigure` with OptArgs -- `r.db("d").table("t").reconfigure({shards: 2, replicas: 1})`
- [x] verify tests fail (red phase) -- run `go test ./internal/reql/parser/ -run <new_tests>`

### Task 2: Add parser unit test for `-F -` stdin (TDD red phase)

- [x] add test in `cmd/r-cli/query_test.go`: `runQueryFile` with path `"-"` reads from stdin
- [x] verify test fails

### Task 3: Add integration tests (TDD red phase)

Integration tests parse string expressions via `parser.Parse` and execute against a live RethinkDB. They will fail because the parser cannot parse OptArgs yet.

- [x] add integration test: `getAll` with secondary index + OptArgs (parse from string expression)
- [x] add integration test: `between` with index + OptArgs (parse from string expression)
- [x] add integration test: `eqJoin` with index + OptArgs (parse from string expression)
- [x] verify tests fail -- run `make test-integration`

### Task 4: Implement OptArgs-aware helper builders in parser (green phase)

Replace ad-hoc chain helpers with OptArgs-aware versions. The fix should be systematic -- modify or create new helper builder functions that all chains can use:

- [ ] create `oneArgChainWithOpts` helper: parses one arg + optional trailing OptArgs
- [ ] create `strArgChainWithOpts` helper: parses string arg + optional trailing OptArgs
- [ ] create `noArgChainWithOpts` helper: parses optional OptArgs only
- [ ] create `parseArgListWithOpts` helper for variadic chains (`getAll`, `orderBy`): parses `(expr, ..., {opts})` at token level -- when a comma is followed by `{` and the parsed OptArgs is followed by `)`, treat it as trailing OptArgs; otherwise parse as a normal expression. Note: backtracking via `p.pos = save` is safe here because `parseOptArgs` only parses datum literals (no lambdas that mutate `paramsStack`/`nextVarID`)
- [ ] fix `chainGetAll`: use `parseArgListWithOpts`
- [ ] fix `chainOrderBy`: use `parseArgListWithOpts`
- [ ] fix `chainBetween`: parse two args + optional OptArgs
- [ ] fix `chainEqJoin`: parse string + arg + optional OptArgs
- [ ] re-register `distance` with `oneArgChainWithOpts`
- [ ] re-register `getIntersecting` with `oneArgChainWithOpts`
- [ ] re-register `getNearest` with `oneArgChainWithOpts`
- [ ] re-register `tableCreate` with `strArgChainWithOpts`
- [ ] re-register `indexCreate` with `strArgChainWithOpts`
- [ ] re-register `changes` with `noArgChainWithOpts`
- [ ] re-register `reconfigure` with `noArgChainWithOpts`
- [ ] run `make build` -- must pass (linter + compile)
- [ ] run `go test ./internal/reql/parser/ -race` -- all new tests from task 1 must pass
- [ ] run `make test-integration` -- all new tests from task 3 must pass

### Task 5: Implement `-F -` stdin support (green phase)

- [ ] in `runQueryFile`: if `path == "-"`, use `cmd.InOrStdin()` instead of `os.Open`
- [ ] run `make build`
- [ ] run `go test ./cmd/r-cli/ -race` -- test from task 2 must pass

### Task 6: Verify acceptance criteria
- [ ] verify all 3 original error scenarios from transcripts work correctly
- [ ] run full test suite: `make test-all`
- [ ] run linter: `golangci-lint run`

## Technical Details

### OptArgs extraction for variadic chains (getAll, orderBy)

`parseArgList()` returns `[]reql.Term`. Object literals `{key: val}` become datum Terms (termType=0, wrapping `map[string]interface{}`). The parser (package `parser`) cannot inspect `reql.Term` internals (unexported fields `termType`, `datum`) across the package boundary, so post-parse Term-to-OptArgs conversion is not possible without adding exported helpers to `reql`.

**Chosen approach: token-level detection with safe backtracking.** Create `parseArgListWithOpts` that parses expressions normally; when a comma is followed by `{`, attempt `parseOptArgs()`. If the result is followed by `)`, accept it as trailing OptArgs. Otherwise, backtrack (`p.pos = save`) and parse as a normal expression.

Backtracking is safe here: `parseOptArgs` calls `parseOptArgValue` which only accepts datum literals (string/number/bool/null) -- it never enters lambda parsing, so `paramsStack` and `nextVarID` are not modified.

This is consistent with how `chainInsert`/`chainUpdate`/`chainDelete` handle OptArgs (check `, {` then `parseOptArgs`).

### Helper builder pattern

```go
// example: oneArgChainWithOpts creates a chain that parses (arg) or (arg, {opts})
func oneArgChainWithOpts(method func(reql.Term, reql.Term, ...reql.OptArgs) reql.Term) chainFunc {
    return func(p *parser, t reql.Term) (reql.Term, error) {
        // parse "(", arg, optional ", {opts}", ")"
    }
}
```

### stdin support

```go
func runQueryFile(cmd *cobra.Command, cfg *rootConfig, path string, stopOnError bool) error {
    var r io.Reader
    if path == "-" {
        r = cmd.InOrStdin()
    } else {
        f, err := os.Open(path)
        // ...
        defer f.Close()
        r = f
    }
    queries, err := splitQueries(r)
    // ...
}
```

## Post-Completion

**Manual verification:**
- test with real RethinkDB: `r-cli query 'r.table("t").getAll("key", {index: "idx"})'`
- test with real RethinkDB: `r-cli query 'r.table("t").between(1, 10, {index: "score"})'`
- test: `echo 'r.dbList()' | r-cli query -F -`
