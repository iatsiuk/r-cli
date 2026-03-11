# REPL startup hint

## Overview
Show available dot-commands when user enters interactive REPL, printed to stderr so it does not interfere with piped output. Respect `--quiet` flag to suppress the hint.

## Context
- `internal/repl/repl.go` -- REPL loop, `Run()` method, `dotCommand()` with `.help` output
- `cmd/r-cli/repl_cmd.go` -- CLI entry point, constructs `repl.Config`
- `cmd/r-cli/root.go` -- `rootConfig.quiet` flag
- hint text matches existing `.help` output format, written to `errOut`

## Development Approach
- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- **CRITICAL: every task MUST include new/updated tests**
- **CRITICAL: all tests must pass before starting next task**

## Implementation Steps

### Task 1: Add `ShowHint` field to `repl.Config` and print hint on REPL start
- [ ] write test: `Run` prints help lines to `errOut` when `ShowHint` is true
- [ ] write test: `Run` prints nothing to `errOut` when `ShowHint` is false (default, backward compat)
- [ ] add `ShowHint bool` field to `repl.Config`
- [ ] in `Repl.Run`, print the same help text as `.help` to `r.errOut` when `showHint` is true, before entering the main loop
- [ ] extract help text into a helper (e.g. `printHelp(w io.Writer)`) shared by `dotCommand(".help")` and startup hint to avoid duplication
- [ ] run tests (`go test ./internal/repl/... -race -count=1`) -- must pass

### Task 2: Wire `ShowHint` from CLI with `--quiet` awareness
- [ ] write test: `repl.Config.ShowHint` is true when `cfg.quiet` is false
- [ ] write test: `repl.Config.ShowHint` is false when `cfg.quiet` is true
- [ ] set `ShowHint: !cfg.quiet` in `repl_cmd.go` when constructing `repl.Config`
- [ ] run tests (`go test ./cmd/r-cli/... -race -count=1`) -- must pass

### Task 3: Verify acceptance criteria
- [ ] verify hint appears on REPL start (manual or integration check)
- [ ] verify `--quiet` suppresses hint
- [ ] verify `.help` still works and shows the same text
- [ ] run full test suite (`make test`)
- [ ] run linter (`make build`)

## Technical Details
- `ShowHint` defaults to `false` (zero value) so existing callers/tests are unaffected
- help text written to `errOut` (stderr), not `out` (stdout)
- shared `printHelp` avoids duplicating the 4 help lines in two places
