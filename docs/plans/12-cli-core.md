# Plan: CLI Core Commands

## Overview

Root command with global flags, `run` command (raw ReQL JSON), `db` and `table` subcommands, `status` command. The `query` command (11.2) depends on the parser and is in a separate plan.

Package: `cmd/r-cli`

Depends on: `10-connmgr-query`, `11-output`

## Validation Commands
- `go test ./cmd/... -race -count=1`
- `make build`

### Task 1: Root command and global flags

- [ ] Test: `--host` / `-H` flag defaults to "localhost"
- [ ] Test: `--port` / `-P` flag defaults to 28015
- [ ] Test: `--db` / `-d` flag sets default database
- [ ] Test: `--user` / `-u` flag defaults to "admin"
- [ ] Test: `--password` / `-p` flag (also `RETHINKDB_PASSWORD` env)
- [ ] Test: `--password-file` reads password from file (avoids shell history leaks)
- [ ] Test: `--timeout` / `-t` flag defaults to 30s
- [ ] Test: `--format` / `-f` flag: "json" (default), "jsonl", "raw", "table"
- [ ] Test: `--version` flag
- [ ] Implement: root command with persistent flags

### Task 2: Environment variables and precedence

- [ ] Test: `RETHINKDB_HOST` env var overrides default host
- [ ] Test: `RETHINKDB_PORT` env var overrides default port
- [ ] Test: `RETHINKDB_USER` env var overrides default user
- [ ] Test: `RETHINKDB_PASSWORD` env var overrides default password
- [ ] Test: `RETHINKDB_DATABASE` env var overrides default db
- [ ] Test: CLI flag takes precedence over env var

### Task 3: Additional flags and signal handling

- [ ] Test: `--profile` flag enables query profiling output
- [ ] Test: `--time-format` flag: "native" (default, pseudo-type conversion), "raw" (pass-through)
- [ ] Test: `--binary-format` flag: "native" (default), "raw" (pass-through)
- [ ] Test: `--quiet` suppresses non-data output to stderr
- [ ] Test: `--verbose` shows connection info and query timing to stderr
- [ ] Test: exit code 0 on success
- [ ] Test: exit code 1 on connection error
- [ ] Test: exit code 2 on query/parse error
- [ ] Test: exit code 3 on auth error
- [ ] Test: SIGINT during query -> cancel context, clean exit code 130
- [ ] Test: SIGINT during output streaming -> stop output, clean exit
- [ ] Implement: signal handler (SIGINT/SIGTERM) -> cancel root context

### Task 4: `run` command

Execute a raw ReQL JSON term directly (pre-serialized).

- [ ] Test: `r-cli run '[15,[[14,["test"]],"users"]]'` -> sends term as-is
- [ ] Test: stdin input
- [ ] Implement: run command

### Task 5: `db` subcommands

- [ ] Test: `r-cli db list` -> list databases
- [ ] Test: `r-cli db create <name>` -> create database
- [ ] Test: `r-cli db drop <name>` -> drop database (with confirmation)
- [ ] Implement: db command group

### Task 6: `table` subcommands

- [ ] Test: `r-cli table list` -> list tables in current db
- [ ] Test: `r-cli table create <name>` -> create table
- [ ] Test: `r-cli table drop <name>` -> drop table (with confirmation)
- [ ] Test: `r-cli table info <name>` -> table status/config
- [ ] Implement: table command group

### Task 7: `status` and `completion` commands

- [ ] Test: `r-cli status` -> shows server info, connection status
- [ ] Implement: status command
- [ ] Test: `r-cli completion bash` generates valid bash completion script
- [ ] Test: `r-cli completion zsh` generates valid zsh completion script
- [ ] Test: `r-cli completion fish` generates valid fish completion script
- [ ] Implement: cobra built-in completion generation
