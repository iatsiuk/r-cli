# Plan: TLS Support

## Overview

Post-MVP TLS support for RethinkDB connections. Extends `internal/conn` with TLS wrapping and adds CLI flags for certificate configuration.

Package: `internal/conn` (extends Phase 4), `cmd/r-cli`

Depends on: `03-conn`, `12-cli-core`

## Validation Commands
- `go test ./internal/conn/... ./cmd/... -race -count=1`
- `make build`

### Task 1: TLS connection

- [x] Test: `DialTLS` with valid CA cert -> handshake succeeds
- [x] Test: `DialTLS` with wrong CA cert -> TLS verification error
- [x] Test: `DialTLS` with `InsecureSkipVerify` -> connects despite invalid cert
- [x] Implement: `DialTLS(ctx, addr, tlsConfig)` using `crypto/tls`

### Task 2: CLI TLS flags

- [x] Test: `--tls-cert` flag sets CA certificate path
- [x] Test: `--tls-key` + `--tls-client-cert` for client certificate auth
- [x] Test: `--insecure-skip-verify` disables cert verification
- [x] Implement: TLS flags in root command, pass `*tls.Config` to connection
