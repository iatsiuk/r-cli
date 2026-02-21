# Plan: CLI Query Command

## Overview

The `query` command -- primary command for executing ReQL query strings. Supports argument, stdin, and file input. Depends on the parser (plan 14).

Package: `cmd/r-cli`

Depends on: `14-parser`, `12-cli-core`

## Validation Commands
- `go test ./cmd/... ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Query command implementation

- [ ] Test: `r-cli query 'r.db("test").table("users")'` -> executes and prints result
- [ ] Test: `r-cli 'r.db("test").table("users")'` -> query as default command
- [ ] Test: pipe query from stdin: `echo '...' | r-cli query`
- [ ] Test: `--file` / `-F` flag reads query from file
- [ ] Test: `--file` with multiple queries separated by `---` -> execute sequentially, output each
- [ ] Test: `--file` with multiple queries, `--stop-on-error` -> stop on first failure
- [ ] Test: invalid query string -> parse error
- [ ] Test: connection failure -> descriptive error
- [ ] Implement: query command with input modes (arg, stdin, file)
