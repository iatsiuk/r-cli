# Plan: Integration Tests - Advanced

## Overview

Advanced integration tests: ordering/aggregation, pluck/merge, map/reduce/group, secondary indexes, streaming/cursors, changefeeds, pseudo-types, error handling, connection manager, noreply.

Build tag: `//go:build integration`. Package: `internal/integration`.
Shared container via TestMain (same setup as 16-integration-basic).

Depends on: `16-integration-basic`

## Validation Commands
- `go test -tags integration ./internal/integration/... -race -count=1 -run 'Test(OrderBy|Limit|Skip|Count|Distinct|Pluck|Without|Merge|HasFields|Map|Reduce|Group|Index|Stream|Cursor|Change|Pseudo|Error|ConnManager|Noreply)'`
- `make build`

### Task 1: OrderBy, Limit, Skip, Count, Distinct

- [ ] Test: orderBy ascending -> documents in correct order
- [ ] Test: orderBy descending -> reverse order
- [ ] Test: limit(5) on 20 docs -> exactly 5 returned
- [ ] Test: skip(10) on 20 docs -> 10 returned
- [ ] Test: skip(5).limit(5) -> correct slice
- [ ] Test: count on filtered result -> correct number
- [ ] Test: distinct on field with duplicates -> unique values only

### Task 2: Pluck, Without, Merge, HasFields

- [ ] Test: pluck("name") -> documents with only id and name fields
- [ ] Test: without("password") -> documents without password field
- [ ] Test: merge({new_field: "value"}) -> field added to each document
- [ ] Test: hasFields("email") -> only documents that have email field

### Task 3: Map, Reduce, Group

- [ ] Test: map extracts single field -> array of values
- [ ] Test: reduce with ADD -> sum of values
- [ ] Test: group by field -> grouped object with arrays
- [ ] Test: group + count -> count per group
- [ ] Test: ungroup -> array of {group, reduction} objects

### Task 4: Secondary indexes

- [ ] Test: INDEX_CREATE on field -> index created
- [ ] Test: INDEX_LIST -> includes new index
- [ ] Test: INDEX_WAIT -> index ready
- [ ] Test: INDEX_STATUS -> status shows ready=true
- [ ] Test: GetAll with secondary index -> uses index
- [ ] Test: Between with secondary index -> correct range
- [ ] Test: INDEX_DROP removes index
- [ ] Test: INDEX_RENAME renames index

### Task 5: Streaming and cursors

- [ ] Test: query returning >1 batch (insert 1000+ small docs, read all) -> multiple CONTINUE roundtrips
- [ ] Test: cursor Next() returns documents one by one
- [ ] Test: cursor All() collects everything into slice
- [ ] Test: cursor Close() mid-stream -> sends STOP, no error
- [ ] Test: cursor with context cancel -> stops iteration, no leak
- [ ] Test: two concurrent cursors on same connection -> both complete correctly

### Task 6: Changefeeds

- [ ] Test: changes() on table -> insert a doc in separate goroutine, cursor receives the change
- [ ] Test: change object has old_val=null, new_val=<doc> for insert
- [ ] Test: update triggers change with old_val and new_val
- [ ] Test: delete triggers change with old_val=<doc>, new_val=null
- [ ] Test: cursor Close() stops changefeed cleanly
- [ ] Test: changes with include_initial=true -> receives existing docs first

### Task 7: Pseudo-types

- [ ] Test: insert document with r.now() -> returned epoch_time is recent timestamp
- [ ] Test: TIME pseudo-type in response converts to time.Time correctly
- [ ] Test: timezone preserved in roundtrip
- [ ] Test: BINARY pseudo-type -> insert base64 data, read back as []byte, matches original
- [ ] Test: r.uuid() -> returns valid UUID string

### Task 8: Error handling

- [ ] Test: query non-existent table -> RUNTIME_ERROR with NON_EXISTENCE
- [ ] Test: query non-existent database -> RUNTIME_ERROR with NON_EXISTENCE
- [ ] Test: malformed ReQL JSON -> COMPILE_ERROR
- [ ] Test: type mismatch (e.g. add string + number) -> RUNTIME_ERROR
- [ ] Test: query timeout via context -> context.DeadlineExceeded, no dangling connection

### Task 9: Connection manager, reconnect and noreply

- [ ] Test: 50 concurrent queries through single multiplexed connection -> all succeed, no races
- [ ] Test: kill container mid-query -> ConnManager reconnects after restart (uses `startRethinkDBForRestart`, `ctr.Stop(ctx)` + `ctr.Start(ctx)`)
- [ ] Test: ConnManager Close() with active queries -> all queries return error
- [ ] Test: insert with noreply=true -> no response, document appears in table
- [ ] Test: NOREPLY_WAIT after noreply inserts -> WAIT_COMPLETE, all writes visible
