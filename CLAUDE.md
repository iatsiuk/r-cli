# R-CLI Project Instructions

## Development Tool

This project is developed using [ralphex](https://github.com/umputun/ralphex) CLI utility.
- Config directory: `.ralphex` (in project root)
- Run with `--config-dir .ralphex` option

## Project Files

- `rethink-driver.md` - RethinkDB wire protocol specification for driver implementation (handshake, SCRAM-SHA-256, ReQL serialization, term types, response format, streaming)
- `docs/plans/` - numbered TDD implementation plans (e.g. `04-reql-core.md`); each plan has phases with test cases as checklist

## Package Structure

- `internal/proto` - RethinkDB protocol constants only (Version, QueryType, ResponseType, ErrorType, ResponseNote, DatumType, TermType); pure constants, no I/O. Max payload constraint: 64MB.
- `internal/wire` - Binary frame encode/decode (Encode, DecodeHeader) and I/O helpers (ReadResponse, WriteQuery); depends on internal/proto
- `internal/scram` - SCRAM-SHA-256 authentication per RFC 5802 / RFC 7677; functions: GenerateNonce, ClientFirstMessage, ParseServerFirst, ComputeProof, ClientFinalMessage, VerifyServerFinal; Conversation struct for stateful 3-step exchange; pure cryptographic computation, no I/O
- `internal/conn` - TCP/TLS connection with V1_0 SCRAM-SHA-256 handshake and multiplexed query dispatch; exported: `Conn`, `Config`, `Dial`, `ErrClosed`, `ErrReqlAuth`, `Handshake`, `IsClosed`, `NextToken`, `WriteFrame`; background `readLoop` dispatches responses by token into buffered channels; `WriteFrame` writes raw frames without registering a waiter (used for noreply and STOP); set `RCLI_DEBUG=wire` for hex-dump wire tracing to stderr; depends on `internal/proto`, `internal/wire`, `internal/scram`
- `internal/response` - RethinkDB response parsing; exported: `Response` struct (fields: Type, Results, ErrType, Backtrace, Notes, Profile), `Parse(data []byte) (*Response, error)`, `ConvertPseudoTypes(v interface{}) interface{}`, `MapError(resp *Response) error`; error types: `ReqlClientError`, `ReqlCompileError`, `ReqlRuntimeError`, `ReqlNonExistenceError`, `ReqlPermissionError`; `ConvertPseudoTypes` recursively converts TIME -> `time.Time`, BINARY -> `[]byte`, GEOMETRY passes through; depends on `internal/proto`
- `internal/cursor` - Result iteration over RethinkDB responses; exported interface: `Cursor` with `Next() (json.RawMessage, error)`, `All() ([]json.RawMessage, error)`, `Close() error`; constructors: `NewAtom(resp)` for SUCCESS_ATOM, `NewSequence(resp)` for SUCCESS_SEQUENCE, `NewStream(ctx, initial, ch, send)` for paginated SUCCESS_PARTIAL streams (sends CONTINUE, terminates on SUCCESS_SEQUENCE), `NewChangefeed(ctx, initial, ch, send)` for infinite changefeed streams (never auto-completes, All() returns error, only Close() terminates); streaming cursors send STOP exactly once via sync.Once on Close or context cancel; depends on `internal/proto`, `internal/response`
- `internal/connmgr` - lazy-connect connection manager with auto-reconnect; exported: `ConnManager`, `DialFunc`, `New(dial DialFunc) *ConnManager`, `NewFromConfig(cfg conn.Config, tlsCfg *tls.Config) *ConnManager`; `Get(ctx)` returns existing connection or re-dials if closed; `Close()` closes the managed connection; depends on `internal/conn`
- `internal/query` - high-level ReQL query executor; exported: `Executor`, `ServerInfo` struct (fields: ID, Name), `New(mgr *connmgr.ConnManager) *Executor`; `Run(ctx, term, opts) (json.RawMessage, cursor.Cursor, error)` builds and executes a START query, first return is profile data (non-nil only when server sends profiling data), returns nil cursor for noreply; `ServerInfo(ctx) (*ServerInfo, error)` sends query type 5 and parses the response; auto-selects cursor type (Atom/Sequence/Stream/Changefeed) based on response type and notes; depends on `internal/conn`, `internal/connmgr`, `internal/cursor`, `internal/proto`, `internal/reql`, `internal/response`
- `internal/output` - result formatters for query output; exported: `RowIterator` interface (`Next() (json.RawMessage, error)`), `JSON(w io.Writer, iter RowIterator) error` (pretty-printed; single doc direct, multiple wrapped in array, empty as `[]`), `JSONL(w io.Writer, iter RowIterator) error` (one compact JSON per line), `Raw(w io.Writer, iter RowIterator) error` (strings unquoted, others compact JSON), `Table(w io.Writer, iter RowIterator) error` (aligned ASCII table; buffers up to 10000 rows, truncates with warning to stderr, non-object rows fall back to raw; max column width 50 chars, truncation marker `~`), `DetectFormat(stdout *os.File, flagFormat string) string` (explicit flag wins; TTY -> "json", non-TTY -> "jsonl"), `NoColor() bool` (true when NO_COLOR env var is set); depends on nothing
- `internal/reql` - ReQL term builder; exported: `Term`, `Datum`, `Array`, `DB`, `DBCreate`, `DBDrop`, `DBList`, `Asc`, `Desc`, `OptArgs`, `Row`, `Var`, `Func`, `Now`, `UUID`, `Binary`, `Do`, `BuildQuery`, `JSON`, `ISO8601`, `EpochTime`, `Time`, `Branch`, `Error`, `Literal`, `Args`, `MinVal`, `MaxVal`, `GeoJSON`, `Point`, `Line`, `Polygon`, `Circle`, `Grant`, `Monday`-`Sunday`, `January`-`December`; chainable methods on `Term`: `Table`, `TableCreate`, `TableDrop`, `TableList`, `Filter`, `Insert`, `Update`, `Delete`, `Replace`, `Get`, `GetAll`, `Between`, `OrderBy`, `Limit`, `Skip`, `Count`, `Pluck`, `Without`, `GetField`, `HasFields`, `Merge`, `Distinct`, `Map`, `Reduce`, `Group`, `Ungroup`, `Sum`, `Avg`, `Min`, `Max`, `Eq`, `Ne`, `Lt`, `Le`, `Gt`, `Ge`, `Not`, `And`, `Or`, `Add`, `Sub`, `Mul`, `Div`, `Mod`, `Floor`, `Ceil`, `Round`, `IndexCreate`, `IndexDrop`, `IndexList`, `IndexWait`, `IndexStatus`, `IndexRename`, `Changes`, `Config`, `Status`, `Grant`, `InnerJoin`, `OuterJoin`, `EqJoin`, `Zip`, `Match`, `Split`, `Upcase`, `Downcase`, `ToJSONString`, `ToISO8601`, `ToEpochTime`, `Date`, `TimeOfDay`, `Timezone`, `Year`, `Month`, `Day`, `DayOfWeek`, `DayOfYear`, `Hours`, `Minutes`, `Seconds`, `InTimezone`, `During`, `ToGeoJSON`, `Append`, `Prepend`, `Slice`, `Difference`, `InsertAt`, `DeleteAt`, `ChangeAt`, `SpliceAt`, `SetInsert`, `SetIntersection`, `SetUnion`, `SetDifference`, `ForEach`, `Default`, `CoerceTo`, `TypeOf`, `ConcatMap`, `Nth`, `Union`, `IsEmpty`, `Contains`, `Bracket`, `WithFields`, `Keys`, `Values`, `Sync`, `Reconfigure`, `Rebalance`, `Wait`, `Distance`, `Intersects`, `Includes`, `GetIntersecting`, `GetNearest`, `Fill`, `PolygonSub`; terms serialize to ReQL wire JSON via `MarshalJSON`; datum terms (termType==0) serialize as raw values; `Filter` auto-wraps predicates containing `Row()` (IMPLICIT_VAR) in FUNC, errors if `Row()` appears inside explicit nested FUNC; `Do` API order is `Do(arg1, ..., fn)` but wire order puts fn first; `Term` carries deferred errors propagated through `MarshalJSON`; `Insert`, `TableCreate`, `Changes` accept optional `OptArgs` as last variadic arg; `OrderBy` and `GetAll` accept `OptArgs` as the last element of their `...interface{}` variadic for index/options; `EqJoin`, `Reconfigure`, `Circle`, `Distance`, `GetIntersecting`, `GetNearest`, `IndexCreate` accept optional `OptArgs`; `Branch` requires 3+ odd-count arguments (returns errTerm otherwise); `Line` requires 2+ points, `Polygon` requires 3+ points (return errTerm otherwise); `BuildQuery(qt, term, opts)` serializes full query envelope: START `[1,term,opts]` (string `"db"` opt auto-wrapped as DB term), CONTINUE `[2]`, STOP `[3]`, returns error for unsupported query types; depends on `internal/proto`
- `cmd/r-cli` - CLI entry point; persistent global flags: `-H/--host` (localhost), `-P/--port` (28015), `-d/--db`, `-u/--user` (admin), `-p/--password`, `--password-file`, `-t/--timeout` (30s, applied via context.WithTimeout), `-f/--format` (empty = auto: json on TTY, jsonl when piped); env vars `RETHINKDB_HOST/PORT/USER/PASSWORD/DATABASE` override defaults (CLI flag wins); exit codes: 0 ok, 1 connection, 2 query, 3 auth, 130 SIGINT/SIGTERM; `PersistentPreRunE` calls `resolveEnvVars` + `resolvePassword` for every subcommand; `newExecutor(cfg)` shared helper builds `*query.Executor` and returns a cleanup func; `execTerm` shared helper connects, runs a ReQL term, writes formatted output; subcommands: `run` (raw ReQL JSON term from arg or stdin), `db list/create/drop` (drop has `--yes/-y` to skip confirmation), `table list/create/drop/info/reconfigure/rebalance/wait/sync` (requires `--db`; `reconfigure` accepts `--shards`, `--replicas`, `--dry-run`), `index list/create/drop/rename/status/wait` (requires `--db`; `create` accepts `--geo`, `--multi`), `user list/create/delete/set-password` (`create` accepts `--password`; prompts with no-echo on TTY if omitted; `delete` has `--yes/-y`), `grant <user>` (top-level; `--read`, `--write`, `--table`; scope: global / `--db` / `--db --table`; `--read=false` revokes), `insert <db.table>` (`-F/--file`, `--batch-size` default 200, `--conflict error|replace|update`; reads JSONL from stdin or JSON/JSONL from file; format from flag or `.json` extension; prints `{"inserted":N,"errors":N}`), `status` (server info as JSON), `completion bash/zsh/fish` (cobra built-in); `confirmDrop` reads y/yes from io.Reader for destructive operations; `promptPassword` prompts on stderr, reads without echo on TTY (via `golang.org/x/term`), falls back to line-read for non-TTY; format auto-detection uses `output.DetectFormat(os.Stdout, cfg.format)` - explicit flag always wins, empty default triggers TTY check

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
