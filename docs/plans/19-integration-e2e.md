# Plan: Integration Tests - E2E and Extended Operations

## Overview

End-to-end CLI binary tests, geospatial, string/time operations, joins, control flow, REPL, user/permission e2e, index e2e, and bulk insert e2e.

Tests execute compiled `r-cli` binary via `os/exec`.

Build tag: `//go:build integration`. Package: `internal/integration`.

Depends on: `16-integration-basic`, `15-cli-query`, `13-cli-extended`

## Validation Commands
- `go test -tags integration ./internal/integration/... -race -count=1 -run 'Test(CLI|Geo|String|Time|Join|Control|REPL|UserE2E|IndexE2E|BulkInsert)'`
- `make build`

### Task 1: Geospatial integration

- [x] Test: create geo index, insert points, getNearest returns sorted by distance
- [x] Test: getIntersecting with polygon -> correct results
- [x] Test: distance between two points -> correct meters

### Task 2: String/time operations and joins

- [x] Test: filter with match regex -> correct results
- [x] Test: insert with r.now(), read back -> recent timestamp
- [x] Test: group by .year() -> correct grouping
- [x] Test: epochTime roundtrip -> correct value
- [x] Test: eqJoin between two tables on secondary index -> correct joined docs
- [x] Test: eqJoin + zip -> flattened result

### Task 3: Control flow

- [x] Test: update with branch -> conditional field update
- [x] Test: forEach: select from table A, insert into table B
- [x] Test: default on missing field -> fallback value

### Task 4: CLI end-to-end

Tests execute compiled `r-cli` binary via `os/exec`. Host and port from `testAddr`.

- [x] Test: `r-cli -H <host> -P <port> 'r.dbList()'` -> output contains "test"
- [x] Test: `r-cli -H <host> -P <port> db list` -> output contains "test"
- [x] Test: `r-cli -H <host> -P <port> -d <testdb> table list` -> output is valid JSON array
- [x] Test: `r-cli -H <host> -P <port> status` -> output contains server name
- [x] Test: `r-cli -H <host> -P <port> -f json 'r.dbList()'` -> valid JSON output
- [x] Test: `r-cli -H <host> -P <port> -f table 'r.db("<testdb>").table("<t>").limit(5)'` -> ASCII table output
- [x] Test: `r-cli -H <host> -P <port> -f raw 'r.dbList()'` -> plain text, one item per line
- [x] Test: `r-cli -H badhost -P <port> 'r.dbList()'` -> exit code 1, stderr contains error
- [x] Test: `r-cli -H <host> -P <port> run '[59,[]]'` -> same result as r.dbList()
- [x] Test: `echo 'r.dbList()' | r-cli -H <host> -P <port>` -> works via stdin
- [x] Test: `r-cli -H <host> -P <port> query -F /tmp/test.reql` -> reads query from file
- [x] Test: `r-cli -H <host> -P <port> db create <name>` + `r-cli db drop <name>` -> roundtrip
- [x] Test: `r-cli -H <host> -P <port> table create <name> -d <testdb>` + `table drop` -> roundtrip

### Task 5: REPL e2e

- [x] Test: echo query via pipe to r-cli binary (REPL stdin mode)
- [x] Test: multiple queries via pipe separated by newlines

### Task 6: User/permission and index e2e

Uses shared container (admin has no password; avoids local --password flag conflict on user create).

- [x] Test: `r-cli user create` + `r-cli user list` -> user appears
- [x] Test: `r-cli grant <user> --read --db <testdb>` -> user can query that db
- [x] Test: `r-cli user delete` -> user removed
- [x] Test: `r-cli index create` + `r-cli index list` -> index appears
- [x] Test: `r-cli index wait` -> returns after index ready
- [x] Test: `r-cli index drop` -> index removed

### Task 7: Bulk insert e2e

- [ ] Test: generate JSONL file, pipe to `r-cli insert <db.table>` -> documents in table
- [ ] Test: bulk insert with --conflict replace -> existing docs replaced
