# Plan: CLI Query Command

## Overview

The `query` command -- primary command for executing ReQL query strings. Supports argument, stdin, and file input. Depends on the parser (plan 14).

Package: `cmd/r-cli`

Depends on: `14-parser`, `12-cli-core`

## Validation Commands
- `go test ./cmd/... ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Query command implementation

- [x] Test: `r-cli query 'r.db("test").table("users")'` -> executes and prints result
- [x] Test: `r-cli 'r.db("test").table("users")'` -> query as default command
- [x] Test: pipe query from stdin: `echo '...' | r-cli query`
- [x] Test: `--file` / `-F` flag reads query from file
- [x] Test: `--file` with multiple queries separated by `---` -> execute sequentially, output each
- [x] Test: `--file` with multiple queries, `--stop-on-error` -> stop on first failure
- [x] Test: invalid query string -> parse error
- [x] Test: connection failure -> descriptive error
- [x] Implement: query command with input modes (arg, stdin, file)
