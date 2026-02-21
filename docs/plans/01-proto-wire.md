# Plan: Protocol Constants and Wire Framing

## Overview

Foundation layer: define all RethinkDB protocol constants from `ql2.proto` as typed Go constants, and implement binary message encoding/decoding (8-byte token + 4-byte length + JSON payload).

Packages: `internal/proto`, `internal/wire`

## Validation Commands
- `go test ./internal/proto/... ./internal/wire/... -race -count=1`
- `make build`

### Task 1: Version and query type constants

Package: `internal/proto`

- [ ] Test: verify magic number values match spec (V1_0 = 0x34c2bdc3, etc.)
- [ ] Implement: `version.go` -- Version type + constants
- [ ] Test: verify QueryType values (START=1, CONTINUE=2, STOP=3, NOREPLY_WAIT=4, SERVER_INFO=5)
- [ ] Implement: `query.go` -- QueryType type + constants

### Task 2: Response and datum type constants

- [ ] Test: verify ResponseType values (SUCCESS_ATOM=1 .. RUNTIME_ERROR=18)
- [ ] Test: `IsError()` method returns true for types >= 16
- [ ] Implement: `response.go` -- ResponseType, ErrorType, ResponseNote types + constants
- [ ] Test: verify DatumType values (R_NULL=1 .. R_JSON=7)
- [ ] Implement: `datum.go` -- DatumType type + constants

### Task 3: Term type constants

- [ ] Test: verify core term values (DB=14, TABLE=15, FILTER=39, INSERT=56, etc.)
- [ ] Implement: `term.go` -- TermType type + all constants grouped by category

### Task 4: Wire protocol encode/decode

Package: `internal/wire`

- [ ] Test: encode token=1, payload `[1,"foo",{}]` -> expected bytes (LE token + LE length + JSON)
- [ ] Test: encode token=0 (edge case)
- [ ] Test: encode large payload (verify length field correctness)
- [ ] Implement: `Encode(token uint64, payload []byte) []byte`
- [ ] Test: decode 12-byte header -> token + payload length
- [ ] Test: decode with insufficient bytes -> error
- [ ] Implement: `DecodeHeader(data [12]byte) (token uint64, length uint32)`

### Task 5: Wire protocol read/write

- [ ] Test: read header + payload from `bytes.Reader` -> token + JSON
- [ ] Test: read from reader that returns partial data (simulate slow network)
- [ ] Test: read from reader that returns EOF mid-header -> error
- [ ] Test: payload length > MaxFrameSize (64MB) -> error (prevent OOM)
- [ ] Implement: `ReadResponse(r io.Reader) (token uint64, payload []byte, err error)`
- [ ] Test: write query message to `bytes.Buffer`, verify bytes
- [ ] Implement: `WriteQuery(w io.Writer, token uint64, payload []byte) error`
