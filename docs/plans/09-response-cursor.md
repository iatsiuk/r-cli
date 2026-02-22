# Plan: Response Parsing and Cursor

## Overview

Response parsing (pseudo-type conversion, error mapping) and cursor implementation for streaming results (atom, sequence, partial, changefeed).

Packages: `internal/response`, `internal/cursor`

Depends on: `01-proto-wire`, `03-conn`

## Validation Commands
- `go test ./internal/response/... ./internal/cursor/... -race -count=1`
- `make build`

### Task 1: Response struct and unmarshaling

- [x] Test: unmarshal `{"t":1,"r":["foo"]}` -> ResponseType=SUCCESS_ATOM, results=["foo"]
- [x] Test: unmarshal error response with `e` and `b` fields
- [x] Test: unmarshal response with `n` (notes) field
- [x] Test: unmarshal response with `p` (profile) field
- [x] Implement: `Response` struct with JSON unmarshaling

### Task 2: Pseudo-type conversion

- [ ] Test: TIME pseudo-type -> Go `time.Time`
- [ ] Test: BINARY pseudo-type -> Go `[]byte`
- [ ] Test: nested pseudo-types in result documents
- [ ] Test: GEOMETRY pseudo-type -> pass-through as GeoJSON object (no conversion needed)
- [ ] Test: nested GEOMETRY in result documents
- [ ] Test: pass-through when conversion disabled
- [ ] Implement: `ConvertPseudoTypes(v interface{}) interface{}`

### Task 3: Error mapping

- [ ] Test: CLIENT_ERROR (16) -> ReqlClientError
- [ ] Test: COMPILE_ERROR (17) -> ReqlCompileError
- [ ] Test: RUNTIME_ERROR (18) -> ReqlRuntimeError
- [ ] Test: RUNTIME_ERROR with ErrorType NON_EXISTENCE -> ReqlNonExistenceError
- [ ] Test: RUNTIME_ERROR with ErrorType PERMISSION_ERROR -> ReqlPermissionError
- [ ] Test: backtrace included in error message
- [ ] Implement: error types and mapping function

### Task 4: Atom and sequence cursors

- [ ] Test: create from SUCCESS_ATOM response, read single value, then EOF
- [ ] Test: `All()` returns single-element slice
- [ ] Implement: atom cursor
- [ ] Test: create from SUCCESS_SEQUENCE, iterate all items
- [ ] Test: `All()` collects everything
- [ ] Implement: sequence cursor

### Task 5: Streaming cursor (partial results)

Cursor receives data from `conn` via a response channel tied to the query token. For streaming cursors, `conn` keeps the dispatch map entry alive until SUCCESS_SEQUENCE, error, or explicit STOP.

- [ ] Test: SUCCESS_PARTIAL triggers CONTINUE, next batch arrives, ends with SUCCESS_SEQUENCE
- [ ] Test: premature `Close()` sends STOP
- [ ] Test: context cancellation sends STOP
- [ ] Test: concurrent `Next()` calls are safe
- [ ] Implement: streaming cursor with CONTINUE/STOP lifecycle

### Task 6: Changefeed cursor

- [ ] Test: infinite SUCCESS_PARTIAL stream, values arrive incrementally
- [ ] Test: `Close()` sends STOP and terminates
- [ ] Test: connection drop -> error on next `Next()`
- [ ] Implement: changefeed cursor (never auto-completes)
