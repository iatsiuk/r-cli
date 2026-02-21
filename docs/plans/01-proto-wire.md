# Plan: Protocol Constants and Wire Framing

## Overview

Foundation layer: define all RethinkDB protocol constants from `ql2.proto` as typed Go constants, and implement binary message encoding/decoding (8-byte token + 4-byte length + JSON payload).

Packages: `internal/proto`, `internal/wire`

## Validation Commands
- `go test ./internal/proto/... ./internal/wire/... -race -count=1`
- `make build`

### Task 1: Version and query type constants

Package: `internal/proto`

- [x] Test: verify magic number values match spec (V1_0 = 0x34c2bdc3, etc.)
- [x] Implement: `version.go` -- Version type + constants
- [x] Test: verify QueryType values (START=1, CONTINUE=2, STOP=3, NOREPLY_WAIT=4, SERVER_INFO=5)
- [x] Implement: `query.go` -- QueryType type + constants

### Task 2: Response and datum type constants

- [x] Test: verify ResponseType values (SUCCESS_ATOM=1 .. RUNTIME_ERROR=18)
- [x] Test: `IsError()` method returns true for types >= 16
- [x] Implement: `response.go` -- ResponseType, ErrorType, ResponseNote types + constants
- [x] Test: verify DatumType values (R_NULL=1 .. R_JSON=7)
- [x] Implement: `datum.go` -- DatumType type + constants

### Task 3: Term type constants

- [x] Test: verify core term values (DB=14, TABLE=15, FILTER=39, INSERT=56, etc.)
- [x] Implement: `term.go` -- TermType type + all constants grouped by category

### Task 4: Wire protocol encode/decode

Package: `internal/wire`

- [x] Test: encode token=1, payload `[1,"foo",{}]` -> expected bytes (LE token + LE length + JSON)
- [x] Test: encode token=0 (edge case)
- [x] Test: encode large payload (verify length field correctness)
- [x] Implement: `Encode(token uint64, payload []byte) []byte`
- [x] Test: decode 12-byte header -> token + payload length
- [x] Test: decode with insufficient bytes -> error
- [x] Implement: `DecodeHeader(data [12]byte) (token uint64, length uint32)`

### Task 5: Wire protocol read/write

- [x] Test: read header + payload from `bytes.Reader` -> token + JSON
- [x] Test: read from reader that returns partial data (simulate slow network)
- [x] Test: read from reader that returns EOF mid-header -> error
- [x] Test: payload length > MaxFrameSize (64MB) -> error (prevent OOM)
- [x] Implement: `ReadResponse(r io.Reader) (token uint64, payload []byte, err error)`
- [x] Test: write query message to `bytes.Buffer`, verify bytes
- [x] Implement: `WriteQuery(w io.Writer, token uint64, payload []byte) error`
