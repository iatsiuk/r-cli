# Read-only mode flag

## Overview
Add a global `--read-only` flag (and matching `RETHINKDB_READ_ONLY` env var) that rejects every write operation before it reaches the server. When enabled, any query that contains a write term (insert/update/delete/replace, DDL, index DDL, reconfigure/rebalance/sync, grant, set-write-hook) fails with a client-side error `read-only mode: write operations are not permitted` and exit code 2. Read queries and read-only admin commands continue to work unchanged.

## Context
- `internal/proto/term.go` -- ReQL term type constants; add a pure `IsWriteTerm` classifier (no I/O)
- `internal/reql/term.go` -- `Term` struct with unexported fields; write detection must live in this package
- `internal/query/executor.go` -- `Executor.Run` is the single chokepoint every write path passes through (query, run, insert, db/table/index/user/grant subcommands, REPL); `ServerInfo` bypasses it and stays read-only
- `cmd/r-cli/root.go` -- `rootConfig`, persistent flags, `resolveEnvVars`, `envVarsSection`, `exitCode`/`isQueryError`
- `cmd/r-cli/run.go` -- `newExecutor` builds the shared `*query.Executor`
- `internal/integration` -- live-RethinkDB tests via testcontainers (`newExecutor(t)`, `defaultCfg()`, `setupTestDB`, `seedTable`); read-only must be validated here against a real server
- Confirmed decisions: block all mutating terms including `sync`, `reconfigure` (even `--dry-run`) and `rebalance`; support the `RETHINKDB_READ_ONLY` env var; error message `read-only mode: write operations are not permitted`, exit code 2
- Write-term set to block: Insert(56), Update(53), Delete(54), Replace(55), DBCreate(57), DBDrop(58), TableCreate(60), TableDrop(61), IndexCreate(75), IndexDrop(76), IndexRename(156), Reconfigure(176), Rebalance(179), Sync(138), Grant(188), SetWriteHook(189)

## Development Approach
- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests written before implementation**
- **CRITICAL: all tests must pass before starting the next task**
- Enforce read-only at one place (`Executor.Run`, before dialing) rather than per-subcommand (DRY)

## Testing Strategy
- Unit tests for term classification (`proto`, `reql`) and the executor gate
- CLI wiring tests for the flag, env var parsing, precedence, and exit-code mapping
- Integration tests against a live RethinkDB container for write rejection and read allowance
- Run `go test ./... -race -count=1` after each unit task; run `make test-integration` for the integration task; run `make test-all` and `make build` at the end

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Update this plan if implementation deviates from the original scope

## Technical Details
- `proto.IsWriteTerm(tt TermType) bool` -- flat switch over the 16 write term types above; pure function, keeps `proto` I/O-free
- `reql.(Term).ContainsWrite() bool` -- walks the term tree:
  - deferred-error term (`t.err != nil`) -> `false`
  - compound term (`t.termType != 0`): `true` if `proto.IsWriteTerm(t.termType)`, else recurse into `t.args` (this catches writes nested in `forEach`, `do`, function bodies, etc.)
  - raw-datum term (`t.termType == 0`): only scan `json.RawMessage`/`[]byte` datums (the `run` raw-JSON path) via `rawJSONContainsWrite`; native Go datums (builder-path documents) are data, not terms, and are ignored to avoid false positives
- `rawJSONContainsWrite` -- decodes JSON and looks for wire-format term arrays `[writeType, [args...], opts?]`; reliable because ReQL data arrays are always encoded as `MAKE_ARRAY [2,[...]]`, so a bare `[56,[...]]` is always a term
- `query.ErrReadOnly = errors.New("read-only mode: write operations are not permitted")`; `Run` returns `fmt.Errorf("query: %w", ErrReadOnly)` and the guard runs before `e.mgr.Get` so no connection is opened
- `query.New(mgr, opts ...Option)` with `WithReadOnly(bool)` Option -- variadic keeps existing `New(mgr)` callers compatible
- `RETHINKDB_READ_ONLY` parsed with `strconv.ParseBool`; explicit `--read-only` flag wins over env var (same pattern as other settings)
- `exitCode` maps `errors.Is(err, query.ErrReadOnly)` to exit code 2 (query error)

## Implementation Steps

### Task 1: Term write-classification in proto
- [x] write test: `IsWriteTerm` returns true for all 16 write term types (Insert, Update, Delete, Replace, DBCreate, DBDrop, TableCreate, TableDrop, IndexCreate, IndexDrop, IndexRename, Reconfigure, Rebalance, Sync, Grant, SetWriteHook)
- [x] write test: `IsWriteTerm` returns false for representative read terms (Get, GetAll, Filter, Table, DBList, TableList, IndexList, IndexWait, Config, Status, Wait)
- [x] implement `IsWriteTerm(tt TermType) bool` in `internal/proto/term.go`
- [x] run tests (`go test ./internal/proto/... -race -count=1`) -- must pass

### Task 2: Write detection over the term tree in reql
- [x] write test: `ContainsWrite` is true for builder write terms (`DB(x).Table(y).Insert(...)`, `.Update`, `.Delete`, `.Replace`, `DBCreate`, `TableCreate`, `IndexCreate`, `.Reconfigure`, `.Rebalance`, `.Sync`, `Grant`)
- [x] write test: `ContainsWrite` is false for read terms (`Table(x).Filter(...)`, `.Get`, `DBList`, `TableList`, `Table(x).Wait()`)
- [x] write test: `ContainsWrite` is true for a write nested in another term (`Table(x).ForEach(Func(Table(y).Insert(...), 1))`)
- [x] write test: `ContainsWrite` is true for a raw-JSON `run` term wrapping a write (`Datum(json.RawMessage("[56,[[15,[\"t\"]],{...}]]"))`) and false for a raw-JSON read term
- [x] write test: `ContainsWrite` is false for a native datum document whose field value is an array (no false positive) and false for a deferred-error term
- [x] implement `(Term).ContainsWrite()` plus `datumContainsWrite` and `rawJSONContainsWrite` in a new `internal/reql/readonly.go`
- [x] run tests (`go test ./internal/reql/... -race -count=1`) -- must pass

### Task 3: Read-only gate in the query executor
- [x] write test: `New(nil, WithReadOnly(true))` + `Run` with a write term returns `ErrReadOnly` without touching the network (nil manager is safe because the guard precedes `mgr.Get`)
- [x] write test: default `New(mgr)` (no option) executes a write term normally against the mock server (backward compat)
- [x] write test: `New(mgr, WithReadOnly(true))` allows a read term to execute against the mock server
- [x] add `ErrReadOnly` sentinel, `readOnly` field, `Option` type, and `WithReadOnly` to `internal/query/executor.go`
- [x] change `New` to `New(mgr *connmgr.ConnManager, opts ...Option) *Executor`
- [x] add the guard at the top of `Run` (before `e.mgr.Get`): return `fmt.Errorf("query: %w", ErrReadOnly)` when `e.readOnly && term.ContainsWrite()`
- [x] run tests (`go test ./internal/query/... -race -count=1`) -- must pass

### Task 4: CLI flag, env var, and exit-code wiring
- [x] write test: `--read-only` sets `rootConfig.readOnly` true
- [x] write test: `resolveEnvVars` reads `RETHINKDB_READ_ONLY` (true and false), rejects an invalid boolean with an error, and the explicit flag takes precedence over the env var
- [x] write test: `exitCode` maps `query.ErrReadOnly` to 2
- [x] add `readOnly bool` to `rootConfig` and register the `--read-only` persistent flag in `buildRootCmd`
- [x] parse `RETHINKDB_READ_ONLY` in `resolveEnvVars` (via `strconv.ParseBool`, only when the flag was not set) and add a line to `envVarsSection`
- [x] map `errors.Is(err, query.ErrReadOnly)` to exit code 2 in `exitCode`/`isQueryError`
- [x] pass `query.WithReadOnly(cfg.readOnly)` in `newExecutor` (`cmd/r-cli/run.go`)
- [x] run tests (`go test ./cmd/r-cli/... -race -count=1`) -- must pass

### Task 5: Integration tests against a live RethinkDB
- [x] add a read-only helper in `internal/integration` that builds an executor with `query.WithReadOnly(true)` (mirroring `newExecutor(t)`)
- [x] write test: read-only executor rejects document writes (insert, update, delete, replace) with `query.ErrReadOnly` and leaves seeded data unchanged when re-read by a normal executor
- [x] write test: read-only executor rejects DDL (dbCreate, tableCreate, tableDrop, indexCreate) with `query.ErrReadOnly`
- [x] write test: read-only executor rejects admin writes (reconfigure, rebalance, sync, grant) with `query.ErrReadOnly`
- [x] write test: read-only executor allows reads (table list, get, filter over a seeded table)
- [x] run integration tests (`make test-integration`) -- must pass
- [x] run project tests (`go test ./... -race -count=1`) -- must pass

### Task 6: Documentation
- [x] add `--read-only` and `RETHINKDB_READ_ONLY` to the flag/env-var lists in `README.md`
- [x] update `CLAUDE.md` package descriptions: `internal/proto` (`IsWriteTerm`), `internal/reql` (`ContainsWrite`), `internal/query` (`ErrReadOnly`, `WithReadOnly`, new `New` signature), `cmd/r-cli` (`--read-only` flag and env var)
- [x] run project linter (`make build`) -- must pass

### Task 7: Verify acceptance criteria
- [x] verify every requirement from Overview is implemented (write terms blocked in read-only, reads allowed, env var honored, error message and exit code correct)
- [x] run full project test suite (`make test-all`)
- [x] run project linter (`make build`) -- all issues must be fixed
