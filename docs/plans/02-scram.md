# Plan: SCRAM-SHA-256 Authentication

## Overview

Implement SCRAM-SHA-256 per RFC 5802 / RFC 7677 for RethinkDB handshake authentication.

Package: `internal/scram`

## Validation Commands
- `go test ./internal/scram/... -race -count=1`
- `make build`

### Task 1: Nonce generation and client-first-message

- [x] Test: generated nonce is at least 18 bytes, base64-encoded, no commas
- [x] Implement: `GenerateNonce() string`
- [x] Test: build message with known user and nonce, verify format `n,,n=<user>,r=<nonce>`
- [x] Test: username with special characters (=, ,) is properly escaped
- [x] Implement: `ClientFirstMessage(user, nonce string) string`

### Task 2: Parse server-first-message

- [ ] Test: parse `r=<nonce>,s=<salt>,i=<iter>` -> nonce, salt bytes, iteration count
- [ ] Test: parse malformed message -> error
- [ ] Test: parse message with wrong nonce prefix -> error
- [ ] Implement: `ParseServerFirst(msg, clientNonce string) (*ServerFirst, error)`

### Task 3: SCRAM proof computation

- [ ] Test: compute ClientProof with known inputs (use RFC 7677 test vectors)
- [ ] Test: compute ServerSignature with known inputs
- [ ] Implement: `ComputeProof(password string, salt []byte, iter int, authMsg string) (clientProof, serverSig []byte)`

### Task 4: Client-final-message and server-final verification

- [ ] Test: build message with known combined nonce and proof, verify format
- [ ] Implement: `ClientFinalMessage(combinedNonce string, proof []byte) string`
- [ ] Test: verify correct server signature -> success
- [ ] Test: verify wrong server signature -> error
- [ ] Implement: `VerifyServerFinal(msg string, expectedSig []byte) error`

### Task 5: Full SCRAM conversation

- [ ] Test: simulate full 3-step exchange with hardcoded messages, verify all outputs
- [ ] Implement: `Conversation` struct that tracks state across steps
