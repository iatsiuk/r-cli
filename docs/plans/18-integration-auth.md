# Plan: Integration Tests - Authentication and Permissions

## Overview

Integration tests for SCRAM-SHA-256 authentication, user management, and permission system (global, database-level, table-level, inheritance).

Uses `startRethinkDBWithPassword(t, "testpass")` -- separate container with admin password.

Build tag: `//go:build integration`. Package: `internal/integration`.

Depends on: `16-integration-basic`

## Validation Commands
- `go test -tags integration ./internal/integration/... -race -count=1 -run 'Test(Auth|Permission|User)'`
- `make build`

### Task 1: SCRAM-SHA-256 handshake

- [x] Test: connect as admin with correct password ("testpass") -> handshake succeeds
- [x] Test: connect with wrong password -> ReqlAuthError (error_code 10-20)
- [x] Test: connect with non-existent username -> ReqlAuthError
- [x] Test: create user with password, connect with correct credentials -> handshake succeeds
- [x] Test: create user, change password, old password fails, new password works
- [x] Test: user with special characters in password (unicode, quotes, commas) -> handshake succeeds
- [x] Test: user with empty password -> handshake succeeds (if server allows)

### Task 2: Global permissions

- [x] Test: create user with no permissions -> any query returns PERMISSION_ERROR
- [x] Test: grant global read -> user can r.dbList(), r.table().count()
- [x] Test: global read without write -> insert returns PERMISSION_ERROR
- [x] Test: grant global read+write -> insert succeeds
- [x] Test: global write without read -> select returns PERMISSION_ERROR
- [x] Test: revoke permissions (grant read: false) -> previously working query fails

### Task 3: Database-level permissions

- [x] Test: grant read on specific db only -> can query tables in that db
- [x] Test: query table in different db -> PERMISSION_ERROR
- [x] Test: grant write on specific db -> insert in that db succeeds
- [x] Test: insert in other db -> PERMISSION_ERROR
- [x] Test: config permission on db -> can create/drop tables in that db
- [x] Test: config=false -> TABLE_CREATE returns PERMISSION_ERROR

### Task 4: Table-level permissions and inheritance

- [x] Test: grant read on specific table -> can query that table
- [x] Test: query different table in same db -> PERMISSION_ERROR
- [x] Test: grant write on specific table -> insert into that table succeeds
- [x] Test: insert into different table -> PERMISSION_ERROR
- [x] Test: global read + db-level write override -> user can read globally but write only in specific db
- [x] Test: db-level read=false overrides global read=true -> PERMISSION_ERROR on that db
- [x] Test: table-level grant overrides db-level -> more specific wins

### Task 5: Cleanup

- [ ] Test: delete user -> connection with that user fails on next query or reconnect
- [ ] Test: t.Cleanup removes all test users (no leftover state between test runs)
