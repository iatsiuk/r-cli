# Plan: Integration Tests - Basic

## Overview

Integration tests against real RethinkDB (via testcontainers-go). Covers connection, handshake, server info, database/table CRUD, document CRUD, filter, and basic query operations.

Build tag: `//go:build integration`. Package: `internal/integration`.
Container: `rethinkdb:2.4.4`, shared via TestMain.

Depends on: all internal packages (01-11)

## Validation Commands
- `go test -tags integration ./internal/integration/... -race -count=1 -run 'Test(Connection|ServerInfo|Database|Table|Insert|Get|Filter|Update|Replace|Delete)'`
- `make build`

### Task 1: Test infrastructure and connection tests

Set up TestMain with shared container, `setupTestDB`, `createTestTable` helpers.

- [x] Test: connect with default credentials (admin, no password) -> handshake succeeds
- [x] Test: verify server version is returned in handshake response
- [x] Test: connect to non-existent host -> dial error with timeout
- [x] Test: open connection, close it, verify TCP socket released
- [x] Test: concurrent Dial from multiple goroutines -> all succeed
- [x] Implement: TestMain, setupTestDB, createTestTable helpers

### Task 2: Server info and database operations

- [x] Test: SERVER_INFO query returns valid server name and id (non-empty strings)
- [x] Test: server id is a valid UUID format
- [x] Test: DB_LIST returns array containing "rethinkdb" and "test" (default system dbs)
- [x] Test: DB_CREATE creates a new database, DB_LIST now includes it
- [x] Test: DB_CREATE with existing name -> RUNTIME_ERROR (OP_FAILED)
- [x] Test: DB_DROP removes database, DB_LIST no longer includes it
- [x] Test: DB_DROP non-existent database -> RUNTIME_ERROR (OP_FAILED)

### Task 3: Table operations

- [ ] Test: TABLE_CREATE in test db, TABLE_LIST includes new table
- [ ] Test: TABLE_CREATE with primary_key option -> table uses custom primary key
- [ ] Test: TABLE_CREATE duplicate name -> RUNTIME_ERROR
- [ ] Test: TABLE_DROP removes table, TABLE_LIST no longer includes it
- [ ] Test: TABLE_DROP non-existent table -> RUNTIME_ERROR
- [ ] Test: CONFIG on table -> returns object with id, name, db, primary_key, shards
- [ ] Test: STATUS on table -> returns object with status.all_replicas_ready = true

### Task 4: Insert operations

- [ ] Test: insert single document -> response has inserted=1, generated_keys has 1 UUID
- [ ] Test: insert document with explicit id -> no generated_keys, inserted=1
- [ ] Test: insert duplicate id -> RUNTIME_ERROR (OP_FAILED) or conflict response
- [ ] Test: insert with conflict="replace" -> replaced=1
- [ ] Test: insert with conflict="update" -> unchanged=1 or replaced=1
- [ ] Test: bulk insert 100 documents -> inserted=100, generated_keys has 100 UUIDs
- [ ] Test: insert empty object -> inserted=1 (id auto-generated)
- [ ] Test: insert document with nested objects and arrays -> roundtrip preserves structure

### Task 5: Get, GetAll and Filter

- [ ] Test: GET with existing id -> returns the document
- [ ] Test: GET with non-existent id -> returns null
- [ ] Test: GET_ALL with multiple ids -> returns matching documents
- [ ] Test: GET_ALL with secondary index -> returns matching documents
- [ ] Test: GET_ALL with no matches -> empty sequence
- [ ] Test: filter by exact field match -> returns matching docs
- [ ] Test: filter with GT comparison -> correct results
- [ ] Test: filter with compound condition (AND) -> correct results
- [ ] Test: filter returns empty sequence when nothing matches
- [ ] Test: filter with nested field access -> correct results

### Task 6: Update, Replace and Delete

- [ ] Test: update single document by GET -> replaced=1, verify field changed
- [ ] Test: update all documents in table (no filter) -> replaced=N
- [ ] Test: update with merge (add new field) -> field appears in document
- [ ] Test: update non-existent document via GET -> skipped=1
- [ ] Test: update with return_changes=true -> old_val and new_val present
- [ ] Test: replace document by GET -> replaced=1, old fields gone
- [ ] Test: replace must include primary key -> RUNTIME_ERROR if missing
- [ ] Test: delete single document by GET -> deleted=1
- [ ] Test: delete with filter -> deleted=N (matching count)
- [ ] Test: delete all from table -> deleted=total
- [ ] Test: delete non-existent document -> deleted=0
