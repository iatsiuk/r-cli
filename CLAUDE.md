# R-CLI Project Instructions

## Development Tool

This project is developed using [ralphex](https://github.com/umputun/ralphex) CLI utility.
- Config directory: `.ralphex` (in project root)
- Run with `--config-dir .ralphex` option

## Project Files

- `rethink-driver.md` - RethinkDB wire protocol specification for driver implementation (handshake, SCRAM-SHA-256, ReQL serialization, term types, response format, streaming)
- `plan.md` - TDD implementation plan with 13 phases, test cases as checklist

## Package Structure

- `internal/proto` - RethinkDB protocol constants only (Version, QueryType, ResponseType, ErrorType, ResponseNote, DatumType, TermType); pure constants, no I/O. Max payload constraint: 64MB.
- `internal/wire` - Binary frame encode/decode (Encode, DecodeHeader) and I/O helpers (ReadResponse, WriteQuery); depends on internal/proto
- `internal/scram` - SCRAM-SHA-256 authentication per RFC 5802 / RFC 7677; functions: GenerateNonce, ClientFirstMessage, ParseServerFirst, ComputeProof, ClientFinalMessage, VerifyServerFinal; Conversation struct for stateful 3-step exchange; pure cryptographic computation, no I/O

## Code Style

### Imports

Group imports in order, separated by blank lines:
1. Standard library
2. External packages
3. Local packages (`r-cli/...`)

```go
import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "r-cli/internal/db"
)
```

### Naming

- Package names: short, lowercase, no underscores (`db`, `config`, `query`)
- Exported types: PascalCase (`Config`, `Session`, `Query`)
- Unexported: camelCase (`validateQuery`, `buildRequest`)
- Acronyms: consistent case (`URL`, `HTTP`, `API` or `url`, `http`, `api`)
- Receivers: short, 1-2 letters (`s` for `*Session`, `q` for `*Query`)
- Errors: `Err` prefix for sentinel errors (`ErrConnectionFailed`)

### Functions

- Max 80 lines, 50 statements (enforced by `funlen` linter)
- Max cyclomatic complexity: 10 (enforced by `cyclop` linter)
- Max nesting depth: 5 (enforced by `nestif` linter)
- Early returns for error handling
- Group related functions together

### Error Handling

- Wrap errors with context: `fmt.Errorf("operation: %w", err)`
- Check all errors (enforced by `errcheck` linter)
- Use `errors.Is`/`errors.As` for error comparison
- Sentinel errors as package-level variables

### Comments

- Only for non-obvious logic
- English, lowercase, brief
- No comments for self-explanatory code

### Structs

- JSON tags on all exported fields: `json:"field_name"`
- Use `omitempty` for optional fields
- Pointer types for optional values (`*float64`, `*int`)
- Group related fields together

### Variables

- Package-level constants in `const` block
- Related constants grouped together
- Unexported package variables with `var`

### Control Flow

- Use `range` with index for modifying slices
- Prefer `for i := range n` over `for i := 0; i < n; i++` (Go 1.22+)
- Use `switch` over long `if-else` chains

### Concurrency

- Use `context.Context` as first parameter
- Use `sync.Mutex` for simple locking
- Use `errgroup` for parallel operations

## Testing

- Use table-driven tests for multiple scenarios
- Use stdlib `testing` package only (no testify)
- Test error paths: timeouts, context cancellation
- Run with race detector: `go test -race ./...`
- Use `t.Parallel()` for independent tests
- Test files: `*_test.go` in same package

## Language

All documentation, comments, and text must be in English.

## Building

- Always build with `make build` (runs linter automatically)
- Direct `go build` skips linting - avoid it

## Linting and Formatting

- Run `golangci-lint run` before committing (executed automatically via `make build`)
- Fix formatting issues with `goimports -w <file>` or `gofmt -w <file>`
- Config: `.golangci.yml` defines enabled linters
- No trailing whitespace, proper import grouping (stdlib, external, local)
