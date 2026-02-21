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

- [ ] Test: create geo index, insert points, getNearest returns sorted by distance
- [ ] Test: getIntersecting with polygon -> correct results
- [ ] Test: distance between two points -> correct meters

### Task 2: String/time operations and joins

- [ ] Test: filter with match regex -> correct results
- [ ] Test: insert with r.now(), read back -> recent timestamp
- [ ] Test: group by .year() -> correct grouping
- [ ] Test: epochTime roundtrip -> correct value
- [ ] Test: eqJoin between two tables on secondary index -> correct joined docs
- [ ] Test: eqJoin + zip -> flattened result

### Task 3: Control flow

- [ ] Test: update with branch -> conditional field update
- [ ] Test: forEach: select from table A, insert into table B
- [ ] Test: default on missing field -> fallback value

### Task 4: CLI end-to-end

Tests execute compiled `r-cli` binary via `os/exec`. Host and port from `testAddr`.

- [ ] Test: `r-cli -H <host> -P <port> 'r.dbList()'` -> output contains "test"
- [ ] Test: `r-cli -H <host> -P <port> db list` -> output contains "test"
- [ ] Test: `r-cli -H <host> -P <port> -d <testdb> table list` -> output is valid JSON array
- [ ] Test: `r-cli -H <host> -P <port> status` -> output contains server name
- [ ] Test: `r-cli -H <host> -P <port> -f json 'r.dbList()'` -> valid JSON output
- [ ] Test: `r-cli -H <host> -P <port> -f table 'r.db("<testdb>").table("<t>").limit(5)'` -> ASCII table output
- [ ] Test: `r-cli -H <host> -P <port> -f raw 'r.dbList()'` -> plain text, one item per line
- [ ] Test: `r-cli -H badhost -P <port> 'r.dbList()'` -> exit code 1, stderr contains error
- [ ] Test: `r-cli -H <host> -P <port> run '[59,[]]'` -> same result as r.dbList()
- [ ] Test: `echo 'r.dbList()' | r-cli -H <host> -P <port>` -> works via stdin
- [ ] Test: `r-cli -H <host> -P <port> query -F /tmp/test.reql` -> reads query from file
- [ ] Test: `r-cli -H <host> -P <port> db create <name>` + `r-cli db drop <name>` -> roundtrip
- [ ] Test: `r-cli -H <host> -P <port> table create <name> -d <testdb>` + `table drop` -> roundtrip

### Task 5: REPL e2e

- [ ] Test: echo query via pipe to r-cli binary (REPL stdin mode)
- [ ] Test: multiple queries via pipe separated by newlines

### Task 6: User/permission and index e2e

Uses password container.

- [ ] Test: `r-cli user create` + `r-cli user list` -> user appears
- [ ] Test: `r-cli grant <user> --read --db <testdb>` -> user can query that db
- [ ] Test: `r-cli user delete` -> user removed
- [ ] Test: `r-cli index create` + `r-cli index list` -> index appears
- [ ] Test: `r-cli index wait` -> returns after index ready
- [ ] Test: `r-cli index drop` -> index removed

### Task 7: Bulk insert e2e

- [ ] Test: generate JSONL file, pipe to `r-cli insert <db.table>` -> documents in table
- [ ] Test: bulk insert with --conflict replace -> existing docs replaced
