# Plan: CLI Extended Commands

## Overview

Extended CLI subcommands: index management, user management, grant permissions, table admin operations, and bulk insert.

Package: `cmd/r-cli`

Depends on: `12-cli-core`

## Validation Commands
- `go test ./cmd/... -race -count=1`
- `make build`

### Task 1: `index` subcommands

- [x] Test: `r-cli index list <table>` -> list secondary indexes
- [x] Test: `r-cli index create <table> <name>` -> create secondary index
- [x] Test: `r-cli index create <table> <name> --geo` -> create geo index
- [x] Test: `r-cli index create <table> <name> --multi` -> create multi index
- [x] Test: `r-cli index drop <table> <name>` -> drop index
- [x] Test: `r-cli index rename <table> <old> <new>` -> rename index
- [x] Test: `r-cli index status <table> [name]` -> show index status
- [x] Test: `r-cli index wait <table> [name]` -> wait for index readiness
- [x] Implement: index command group

### Task 2: `user` subcommands

- [ ] Test: `r-cli user list` -> list users from rethinkdb.users table
- [ ] Test: `r-cli user create <name> --password <pwd>` -> insert user
- [ ] Test: `r-cli user create <name>` (no password flag) -> prompt for password (no echo)
- [ ] Test: `r-cli user delete <name>` -> delete user (with confirmation)
- [ ] Test: `r-cli user set-password <name>` -> prompt and update password (no echo)
- [ ] Implement: user command group (uses `golang.org/x/term` for password prompt)

### Task 3: `grant` command

- [ ] Test: `r-cli grant <user> --read --write` -> global permissions
- [ ] Test: `r-cli grant <user> --read --db test` -> database permissions
- [ ] Test: `r-cli grant <user> --read --db test --table users` -> table permissions
- [ ] Test: `r-cli grant <user> --read=false` -> revoke permission
- [ ] Implement: grant command with scope flags

### Task 4: `table reconfigure`, `rebalance`, `wait`, `sync`

- [ ] Test: `r-cli table reconfigure <name> --shards 4 --replicas 2`
- [ ] Test: `r-cli table reconfigure <name> --dry-run` -> preview without applying
- [ ] Test: `r-cli table rebalance <name>`
- [ ] Test: `r-cli table wait <name>`
- [ ] Test: `r-cli table sync <name>`
- [ ] Implement: extend table command group

### Task 5: `insert` command (bulk)

- [ ] Test: `cat data.jsonl | r-cli insert <db.table>` -> bulk insert from stdin
- [ ] Test: `r-cli insert <db.table> -F data.json` -> bulk insert from JSON file
- [ ] Test: `r-cli insert <db.table> -F data.jsonl --format jsonl` -> JSONL file
- [ ] Test: `--batch-size N` controls documents per insert (default 200)
- [ ] Test: `--conflict replace|update|error` conflict strategy
- [ ] Test: reports total inserted/errors on completion
- [ ] Implement: insert command with streaming stdin reader
