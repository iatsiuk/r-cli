# Plan: Connection and Handshake

## Overview

TCP connection with V1_0 handshake and multiplexed query dispatch on a single connection. Background `readLoop` goroutine dispatches responses to waiters by token.

Package: `internal/conn`

Depends on: `internal/proto`, `internal/wire`, `internal/scram`

## Validation Commands
- `go test ./internal/conn/... -race -count=1`
- `make build`

### Task 1: Null-terminated message framing

Handshake uses null-terminated JSON messages (not token+length framing).

- [x] Test: `readNullTerminated` reads until `\x00`, returns data without terminator
- [x] Test: `readNullTerminated` with data arriving in 1-byte chunks (partial reads)
- [x] Test: `readNullTerminated` on EOF before `\x00` -> error
- [x] Test: `readNullTerminated` exceeding maxHandshakeSize (16KB) -> error (prevent OOM)
- [x] Test: `writeNullTerminated` appends `\x00` to output
- [x] Implement: `readNullTerminated(r io.Reader) ([]byte, error)`, `writeNullTerminated(w io.Writer, data []byte) error`

### Task 2: Handshake message building and response parsing

- [x] Test: build step 1 bytes (magic number LE)
- [x] Test: build step 3 JSON (protocol_version, authentication_method, authentication) + `\x00`
- [x] Test: build step 5 JSON (client-final-message) + `\x00`
- [x] Implement: handshake message builders
- [x] Test: parse step 2 JSON -> server version, protocol range
- [x] Test: parse step 2 non-JSON error string -> error
- [x] Test: parse step 4 success -> extract authentication field
- [x] Test: parse step 4 with error_code 10-20 -> ReqlAuthError
- [x] Test: parse step 6 success -> extract server signature
- [x] Implement: handshake response parsers

### Task 3: Full handshake over mock connection

- [x] Test: simulate full 6-step handshake using `net.Pipe()`, verify all messages
- [x] Test: handshake with wrong password -> auth error
- [x] Test: handshake with incompatible protocol version -> error
- [x] Test: pipelined handshake (steps 1+3 sent together, then read steps 2+4) reduces RTT
- [x] Implement: `Handshake(rw io.ReadWriter, user, password string) error`

### Task 4: Token counter and Config

- [x] Test: sequential tokens from same connection are monotonically increasing
- [x] Test: concurrent token generation is safe (no duplicates)
- [x] Implement: atomic uint64 counter in `Conn`
- [x] Test: Config.String() does not contain password

### Task 5: Connection struct and multiplexing

Architecture: `Conn` owns a `net.Conn` and runs a background `readLoop` goroutine.
- `readLoop` continuously reads wire frames, dispatches `RawResponse` to correct waiter via `map[uint64]chan RawResponse` (guarded by mutex).
- Dispatch channels are buffered (size 1) with non-blocking send.
- `Send()` registers a response channel, acquires write mutex, writes framed query.
- `Close()` stops readLoop, unblocks all pending waiters.
- `Dial()` accepts optional `*tls.Config` parameter (nil = plain TCP).
- Debug wire dump: `RCLI_DEBUG=wire` env var hex-dumps frames to stderr.

- [x] Test: connect to mock server (net.Pipe), handshake, send query, receive response
- [x] Test: concurrent queries on same connection -> each receives its own response
- [x] Test: out-of-order responses (server replies token 2 before token 1) -> correct dispatch
- [x] Test: slow consumer on token 1 does not block delivery to token 2
- [x] Test: late response after STOP (token removed) -> silently discarded, no panic
- [x] Test: close connection unblocks all pending waiters with error
- [x] Test: Send() after Close() returns error immediately
- [x] Test: context cancellation during query sends STOP and cleans up dispatch entry
- [x] Test: context cancellation during handshake -> no goroutine leak
- [x] Test: STOP sent while server sends one more SUCCESS_PARTIAL -> no deadlock
- [x] Implement: `Conn` struct with `Dial()`, `Close()`, `Send()`, background `readLoop`
