# Parser Error File Log

## Overview

Add persistent file logging for parser errors to `~/.r-cli/parser-errors.log`. Every time `parser.Parse()` fails, the expression and error are appended to the log file as a JSONL entry. This enables tracking parser bugs over time without polluting stderr or requiring env vars.

Log format (JSONL): `{"ts":"...","ver":"...","err":"...","expr":"..."}`

The log file is created lazily on first error. Directory `~/.r-cli/` is created if it doesn't exist (resolved via `os.UserHomeDir()`). Log file is opened in append mode.

## Context

- Parser errors occur in 2 call sites: `cmd/r-cli/query.go:64` and `cmd/r-cli/repl_cmd.go:144`
- Both call `parser.Parse(expr)` and check the returned error
- Version is a package-level `var version = "dev"` in `cmd/r-cli/main.go`, set via ldflags at build time
- A new `internal/parselog` package will provide `Log(expr, err)` and `SetVersion(v)` functions
- The function is fire-and-forget: logging failures are silently ignored (never affect query execution)

## Development Approach
- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task**
- Run tests after each change

## Implementation Steps

### Task 1: Create `internal/parselog` package with `Log` function
- [ ] write tests for `Log(expr, err)`: appends correct JSONL line with all 4 fields
- [ ] write test: creates directory if missing (use temp dir via `SetDir`)
- [ ] write test: silently ignores write errors (read-only dir)
- [ ] write test: concurrent calls produce valid JSONL lines (no corruption)
- [ ] write test: does nothing when err is nil
- [ ] write test: expression longer than 4096 bytes is truncated
- [ ] write test: expression with `\n`, `\t`, `\r` and unicode is properly JSON-escaped
- [ ] write test: `os.UserHomeDir()` failure results in silent no-op
- [ ] write test: `SetDir`/`SetVersion` use `t.Cleanup` to restore previous state
- [ ] implement `Log(expr string, err error)` in `internal/parselog/parselog.go`
- [ ] add `SetDir(path)` for test injection (override default dir)
- [ ] add `SetVersion(v string)` to set version string (called once at startup)
- [ ] run tests - must pass before next task

### Task 2: Integrate logging into CLI call sites
- [ ] call `parselog.SetVersion(version)` in `cmd/r-cli/main.go` before command execution
- [ ] call `parselog.Log(expr, err)` in `cmd/r-cli/query.go` after `parser.Parse` fails
- [ ] call `parselog.Log(expr, err)` in `cmd/r-cli/repl_cmd.go` after `parser.Parse` fails
- [ ] write test: `runQueryExpr` with invalid expression logs to file with version
- [ ] write test: REPL exec with invalid expression logs to file
- [ ] run tests - must pass before next task

### Task 3: Verify and finalize
- [ ] run full test suite (`go test ./...`)
- [ ] run linter (`make build`)
- [ ] verify log file format manually with a broken expression

## Technical Details

**Package**: `internal/parselog`

**Exported API**:
- `Log(expr string, err error)` - append JSONL entry to log file; no-op if err is nil
- `SetDir(path string)` - override log directory (for tests)
- `SetVersion(v string)` - set version string included in each log line

**Log file path**: `$HOME/.r-cli/parser-errors.log` (resolved via `os.UserHomeDir()`)

**JSONL entry**: `{"ts":"2026-03-04T15:04:05+03:00","ver":"v0.1.2","err":"expected ')' at position 42","expr":"r.table(\"t\").between([\"a\"]"}`

- One JSON object per line; `encoding/json.Marshal` handles all escaping (tabs, newlines, unicode)
- Expression truncated to 4096 bytes max (protection against 64MB input from `--file`)
- File opened/closed per write (append mode, no persistent handle)
- Permissions: directory `0700`, file `0600`
- `os.UserHomeDir()` failure (e.g. `$HOME` not set) results in silent no-op

**Concurrency**: `sync.Mutex` protects writes within a single process.

**Test safety**: `SetDir`/`SetVersion` modify package-level state; tests must use `t.Cleanup` to restore previous values. Tests that call these functions should not use `t.Parallel` with each other.
