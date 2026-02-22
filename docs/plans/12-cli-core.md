# Plan: CLI Core Commands

## Overview

Root command with global flags, `run` command (raw ReQL JSON), `db` and `table` subcommands, `status` command. The `query` command (11.2) depends on the parser and is in a separate plan.

Package: `cmd/r-cli`

Depends on: `10-connmgr-query`, `11-output`

## Validation Commands
- `go test ./cmd/... -race -count=1`
- `make build`

### Task 1: Root command and global flags

- [x] Test: `--host` / `-H` flag defaults to "localhost"
- [x] Test: `--port` / `-P` flag defaults to 28015
- [x] Test: `--db` / `-d` flag sets default database
- [x] Test: `--user` / `-u` flag defaults to "admin"
- [x] Test: `--password` / `-p` flag (also `RETHINKDB_PASSWORD` env)
- [x] Test: `--password-file` reads password from file (avoids shell history leaks)
- [x] Test: `--timeout` / `-t` flag defaults to 30s
- [x] Test: `--format` / `-f` flag: "json" (default), "jsonl", "raw", "table"
- [x] Test: `--version` flag
- [x] Implement: root command with persistent flags

### Task 2: Environment variables and precedence

- [x] Test: `RETHINKDB_HOST` env var overrides default host
- [x] Test: `RETHINKDB_PORT` env var overrides default port
- [x] Test: `RETHINKDB_USER` env var overrides default user
- [x] Test: `RETHINKDB_PASSWORD` env var overrides default password
- [x] Test: `RETHINKDB_DATABASE` env var overrides default db
- [x] Test: CLI flag takes precedence over env var

### Task 3: Additional flags and signal handling

- [x] Test: `--profile` flag enables query profiling output
- [x] Test: `--time-format` flag: "native" (default, pseudo-type conversion), "raw" (pass-through)
- [x] Test: `--binary-format` flag: "native" (default), "raw" (pass-through)
- [x] Test: `--quiet` suppresses non-data output to stderr
- [x] Test: `--verbose` shows connection info and query timing to stderr
- [x] Test: exit code 0 on success
- [x] Test: exit code 1 on connection error
- [x] Test: exit code 2 on query/parse error
- [x] Test: exit code 3 on auth error
- [x] Test: SIGINT during query -> cancel context, clean exit code 130
- [x] Test: SIGINT during output streaming -> stop output, clean exit
- [x] Implement: signal handler (SIGINT/SIGTERM) -> cancel root context

### Task 4: `run` command

Execute a raw ReQL JSON term directly (pre-serialized).

- [x] Test: `r-cli run '[15,[[14,["test"]],"users"]]'` -> sends term as-is
- [x] Test: stdin input
- [x] Implement: run command

### Task 5: `db` subcommands

- [x] Test: `r-cli db list` -> list databases
- [x] Test: `r-cli db create <name>` -> create database
- [x] Test: `r-cli db drop <name>` -> drop database (with confirmation)
- [x] Implement: db command group

### Task 6: `table` subcommands

- [x] Test: `r-cli table list` -> list tables in current db
- [x] Test: `r-cli table create <name>` -> create table
- [x] Test: `r-cli table drop <name>` -> drop table (with confirmation)
- [x] Test: `r-cli table info <name>` -> table status/config
- [x] Implement: table command group

### Task 7: `status` and `completion` commands

- [ ] Test: `r-cli status` -> shows server info, connection status
- [ ] Implement: status command
- [ ] Test: `r-cli completion bash` generates valid bash completion script
- [ ] Test: `r-cli completion zsh` generates valid zsh completion script
- [ ] Test: `r-cli completion fish` generates valid fish completion script
- [ ] Implement: cobra built-in completion generation
