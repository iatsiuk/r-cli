# Plan: Connection Manager and Query Executor

## Overview

Single multiplexed connection with lazy connect and automatic reconnect. High-level query executor combining connmgr + reql + cursor.

Packages: `internal/connmgr`, `internal/query`

Depends on: `03-conn`, `04-reql-core`, `09-response-cursor`

## Validation Commands
- `go test ./internal/connmgr/... ./internal/query/... -race -count=1`
- `make build`

### Task 1: Lazy connect

- [x] Test: `Get()` on fresh manager creates connection on first call
- [x] Test: subsequent `Get()` returns the same connection (no reconnect)
- [x] Test: `Close()` closes the underlying connection
- [x] Implement: `ConnManager` struct with `Get(ctx) (*Conn, error)`, `Close()`

### Task 2: Reconnect on failure

- [x] Test: `Get()` after connection drop -> reconnects automatically
- [x] Test: `Get()` during server downtime -> returns dial error
- [x] Test: reconnect preserves config (host, port, user, password, tls)
- [x] Implement: detect closed/errored connection in `Get()`, re-dial

### Task 3: Query executor

- [x] Test: execute `r.db("test").table("users")` against mock server, get cursor
- [x] Test: execute with `db` option
- [x] Test: execute with timeout
- [x] Test: execute with noreply
- [x] Implement: `Executor` struct with `Run(ctx, term, opts) (*Cursor, error)`

### Task 4: Server info

- [ ] Test: `ServerInfo()` returns server name and ID
- [ ] Implement: `ServerInfo(ctx) (*ServerInfo, error)`
