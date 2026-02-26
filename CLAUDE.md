# R-CLI Project Instructions

## Development Tool

This project is developed using [ralphex](https://github.com/umputun/ralphex) CLI utility.
- Config directory: `.ralphex` (in project root)
- Run with `--config-dir .ralphex` option

## Project Files

- `docs/protocol-spec.md` - RethinkDB wire protocol specification for driver implementation (handshake, SCRAM-SHA-256, ReQL serialization, term types, response format, streaming)
- `docs/plans/` - numbered TDD implementation plans (e.g. `04-reql-core.md`); each plan has phases with test cases as checklist

## Package Structure

- `internal/proto` - RethinkDB protocol constants only (Version, QueryType, ResponseType, ErrorType, ResponseNote, DatumType, TermType); pure constants, no I/O. Max payload constraint: 64MB.
- `internal/wire` - Binary frame encode/decode (Encode, DecodeHeader) and I/O helpers (ReadResponse, WriteQuery); depends on internal/proto
- `internal/scram` - SCRAM-SHA-256 authentication per RFC 5802 / RFC 7677; functions: GenerateNonce, ClientFirstMessage, ParseServerFirst, ComputeProof, ClientFinalMessage, VerifyServerFinal; Conversation struct for stateful 3-step exchange; pure cryptographic computation, no I/O
- `internal/conn` - TCP/TLS connection with V1_0 SCRAM-SHA-256 handshake and multiplexed query dispatch; exported: `Conn`, `Config`, `Dial`, `DialTLS`, `ErrClosed`, `ErrReqlAuth`, `Handshake`, `IsClosed`, `NextToken`, `WriteFrame`; `DialTLS(ctx, addr, tlsCfg)` establishes a raw TLS TCP connection without the RethinkDB handshake (used for TLS connectivity tests); background `readLoop` dispatches responses by token into buffered channels; `WriteFrame` writes raw frames without registering a waiter (used for noreply and STOP); set `RCLI_DEBUG=wire` for hex-dump wire tracing to stderr; depends on `internal/proto`, `internal/wire`, `internal/scram`
- `internal/response` - RethinkDB response parsing; exported: `Response` struct (fields: Type, Results, ErrType, Backtrace, Notes, Profile), `Parse(data []byte) (*Response, error)`, `ConvertPseudoTypes(v interface{}) interface{}`, `MapError(resp *Response) error`; error types: `ReqlClientError`, `ReqlCompileError`, `ReqlRuntimeError`, `ReqlNonExistenceError`, `ReqlPermissionError`; `ConvertPseudoTypes` recursively converts TIME -> `time.Time`, BINARY -> `[]byte`, GEOMETRY passes through; depends on `internal/proto`
- `internal/cursor` - Result iteration over RethinkDB responses; exported interface: `Cursor` with `Next() (json.RawMessage, error)`, `All() ([]json.RawMessage, error)`, `Close() error`; constructors: `NewAtom(resp)` for SUCCESS_ATOM, `NewSequence(resp)` for SUCCESS_SEQUENCE, `NewStream(ctx, initial, ch, send)` for paginated SUCCESS_PARTIAL streams (sends CONTINUE, terminates on SUCCESS_SEQUENCE), `NewChangefeed(ctx, initial, ch, send)` for infinite changefeed streams (never auto-completes, All() returns error, only Close() terminates); streaming cursors send STOP exactly once via sync.Once on Close or context cancel; depends on `internal/proto`, `internal/response`
- `internal/connmgr` - lazy-connect connection manager with auto-reconnect; exported: `ConnManager`, `DialFunc`, `New(dial DialFunc) *ConnManager`, `NewFromConfig(cfg conn.Config, tlsCfg *tls.Config) *ConnManager`; `Get(ctx)` returns existing connection or re-dials if closed; `Close()` closes the managed connection; depends on `internal/conn`
- `internal/query` - high-level ReQL query executor; exported: `Executor`, `ServerInfo` struct (fields: ID, Name), `New(mgr *connmgr.ConnManager) *Executor`; `Run(ctx, term, opts) (json.RawMessage, cursor.Cursor, error)` builds and executes a START query, first return is profile data (non-nil only when server sends profiling data), returns nil cursor for noreply; `ServerInfo(ctx) (*ServerInfo, error)` sends query type 5 and parses the response; auto-selects cursor type (Atom/Sequence/Stream/Changefeed) based on response type and notes; depends on `internal/conn`, `internal/connmgr`, `internal/cursor`, `internal/proto`, `internal/reql`, `internal/response`
- `internal/output` - result formatters for query output; exported: `RowIterator` interface (`Next() (json.RawMessage, error)`), `JSON(w io.Writer, iter RowIterator) error` (pretty-printed; single doc direct, multiple wrapped in array, empty as `[]`), `JSONL(w io.Writer, iter RowIterator) error` (one compact JSON per line), `Raw(w io.Writer, iter RowIterator) error` (strings unquoted, others compact JSON), `Table(w io.Writer, iter RowIterator) error` (aligned ASCII table; buffers up to 10000 rows, truncates with warning to stderr, non-object rows fall back to raw; max column width 50 chars, truncation marker `~`), `DetectFormat(stdout *os.File, flagFormat string) string` (explicit flag wins; TTY -> "json", non-TTY -> "jsonl"), `NoColor() bool` (true when NO_COLOR env var is set); depends on nothing
- `internal/reql` - ReQL term builder; exported: `Term`, `Datum`, `Array`, `DB`, `Table`, `DBCreate`, `DBDrop`, `DBList`, `Asc`, `Desc`, `OptArgs`, `Row`, `Var`, `Func`, `Now`, `UUID`, `Binary`, `Do`, `BuildQuery`, `JSON`, `ISO8601`, `EpochTime`, `Time`, `Branch`, `Error`, `Literal`, `Args`, `MinVal`, `MaxVal`, `GeoJSON`, `Point`, `Line`, `Polygon`, `Circle`, `Grant`, `Monday`-`Sunday`, `January`-`December`; chainable methods on `Term`: `Table`, `TableCreate`, `TableDrop`, `TableList`, `Filter`, `Insert`, `Update`, `Delete`, `Replace`, `Get`, `GetAll`, `Between`, `OrderBy`, `Limit`, `Skip`, `Sample`, `Count`, `Pluck`, `Without`, `GetField`, `HasFields`, `Merge`, `Distinct`, `Map`, `Reduce`, `Group`, `Ungroup`, `Sum`, `Avg`, `Min`, `Max`, `Eq`, `Ne`, `Lt`, `Le`, `Gt`, `Ge`, `Not`, `And`, `Or`, `Add`, `Sub`, `Mul`, `Div`, `Mod`, `Floor`, `Ceil`, `Round`, `IndexCreate`, `IndexDrop`, `IndexList`, `IndexWait`, `IndexStatus`, `IndexRename`, `Changes`, `Config`, `Status`, `Grant`, `InnerJoin`, `OuterJoin`, `EqJoin`, `Zip`, `Match`, `Split`, `Upcase`, `Downcase`, `ToJSONString`, `ToISO8601`, `ToEpochTime`, `Date`, `TimeOfDay`, `Timezone`, `Year`, `Month`, `Day`, `DayOfWeek`, `DayOfYear`, `Hours`, `Minutes`, `Seconds`, `InTimezone`, `During`, `ToGeoJSON`, `Append`, `Prepend`, `Slice`, `Difference`, `InsertAt`, `DeleteAt`, `ChangeAt`, `SpliceAt`, `SetInsert`, `SetIntersection`, `SetUnion`, `SetDifference`, `ForEach`, `Default`, `CoerceTo`, `TypeOf`, `ConcatMap`, `Nth`, `Union`, `IsEmpty`, `Contains`, `Bracket`, `WithFields`, `Keys`, `Values`, `Sync`, `Reconfigure`, `Rebalance`, `Wait`, `Distance`, `Intersects`, `Includes`, `GetIntersecting`, `GetNearest`, `Fill`, `PolygonSub`; terms serialize to ReQL wire JSON via `MarshalJSON`; datum terms (termType==0) serialize as raw values; `Filter` auto-wraps predicates containing `Row()` (IMPLICIT_VAR) in FUNC, errors if `Row()` appears inside explicit nested FUNC; `Do` API order is `Do(arg1, ..., fn)` but wire order puts fn first; `Term` carries deferred errors propagated through `MarshalJSON`; `Insert`, `Update`, `Delete`, `TableCreate`, `Changes` accept optional `OptArgs` as last variadic arg; `OrderBy` and `GetAll` accept `OptArgs` as the last element of their `...interface{}` variadic for index/options; `Between`, `EqJoin`, `Reconfigure`, `Circle`, `Distance`, `GetIntersecting`, `GetNearest`, `IndexCreate` accept optional `OptArgs`; `Branch` requires 3+ odd-count arguments (returns errTerm otherwise); `Line` requires 2+ points, `Polygon` requires 3+ points (return errTerm otherwise); `BuildQuery(qt, term, opts)` serializes full query envelope: START `[1,term,opts]` (string `"db"` opt auto-wrapped as DB term), CONTINUE `[2]`, STOP `[3]`, returns error for unsupported query types; depends on `internal/proto`
- `internal/reql/parser` - ReQL string expression parser; exported: `Parse(input string) (reql.Term, error)` converts a human-readable ReQL expression into a `reql.Term`; supports all `r.*` builders (`r.db`, `r.table`, `r.row`, `r.minval`/`r.maxval` without parens, `r.branch`, `r.error`, `r.args`, `r.expr`, `r.now`, `r.uuid`, `r.json`, `r.iso8601`, `r.epochTime`, `r.literal`, `r.point`, `r.geoJSON`, `r.dbCreate`, `r.dbDrop`, `r.dbList`, `r.desc`, `r.asc`) and 100+ chain methods; object `{key: val}` and array `[...]` literals; number/string/bool/null datums; bracket notation `term("field")` (string -> BRACKET) and `term(0)` (integer -> NTH, negative index supported); recognized string escapes: `\"`, `\'`, `\\`, `\n`, `\t`, `\r`; maxDepth=256 guard; error messages include byte position; commas required between arguments; `r.branch` validates odd argument count >= 3 at parse time; arrow/lambda syntax: `(x) => expr` (single-param), `(x, y) => expr` (multi-param), `x => expr` (bare single-param without parens); lambdas compile to `reql.Func(body, paramIDs...)` with `reql.Var(id)` references; param IDs assigned sequentially from 1; parenthesized grouping `(expr)` supported as primary expression (enables `=> ({key: val})` for returning object literals from lambdas); `insert(doc, {key: val})` and `update(doc, {key: val})` accept optional OptArgs object as second argument; `delete({key: val})` accepts optional OptArgs as sole argument; OptArgs values restricted to datum literals (string, number, bool, null); scoping rules: `r.row` inside any lambda scope is an error, nested lambdas supported with proper scoping (top-level IDs start at 1, inner IDs continue from max+1 to avoid collisions), reserved names `true`/`false`/`null` rejected; `r` is allowed as a lambda parameter name -- param lookup takes priority over `r.*` dispatch inside the body; `isLambdaAhead` lookahead detects `( params ) =>` before committing; `paramsStack []map[string]int` and `nextVarID int` fields on parser struct manage nested lambda scopes via `pushScope`/`popScope`, cleaned up via `defer`; `filter` with arrow lambda does not double-wrap (`wrapImplicitVar` skips FUNC terms); `function(params){ return expr }` syntax also supported (JS Data Explorer style); `return` keyword and trailing `;` before `}` are both optional; produces identical FUNC wire JSON as the equivalent arrow lambda; lexer gained `tokenSemicolon` to allow optional `;` before `}`; depends on `internal/reql`
- `internal/repl` - interactive REPL for ReQL expressions; exported: `ErrInterrupt` (sentinel returned by Reader on Ctrl+C), `Reader` interface (`Readline() (string, error)`, `SetPrompt(string)`, `AddHistory(string) error`, `Close() error`), `ExecFunc` type (`func(ctx, expr string, w io.Writer) error`), `Config` struct (fields: `Reader`, `Exec ExecFunc`, `Out`, `ErrOut`, `InterruptCh <-chan struct{}`, `Prompt`, `OnUseDB func(string)`, `OnFormat func(string)`), `Repl` struct, `New(cfg *Config) *Repl`, `Repl.Run(ctx) error`; `TabCompleter` interface (`Do(line []rune, pos int) ([][]rune, int)`), `Completer` struct (fields: `FetchDBs func(ctx) ([]string, error)`, `FetchTables func(ctx, db string) ([]string, error)`; `currentDB string` unexported, updated via `SetCurrentDB(db string)` which is safe to call concurrently); `NewReadlineReader(prompt, historyFile string, out, errOut io.Writer, interruptHook func(), completer ...TabCompleter) (Reader, error)` creates a readline-backed Reader using `github.com/chzyer/readline`; `interruptHook` is called (non-blocking) when Ctrl+C is pressed while readline is in raw mode; pass nil to disable; multiline input: continuation prompt `"... "` shown until parens/braces/brackets balance and depth == 0 (string literals excluded from depth count, escape sequences handled); dot-commands processed only on fresh lines (not during multiline): `.exit`/`.quit` exits, `.use <db>` calls OnUseDB, `.format <fmt>` calls OnFormat, `.help` prints command list; history saved to `~/.r-cli_history` via `AddHistory` after each successful complete expression; depends on `github.com/chzyer/readline`, nothing from internal packages
- `internal/integration` - integration tests against a live RethinkDB instance via testcontainers-go; build tag `//go:build integration`; package `integration`; shared `TestMain` spins up a passwordless `rethinkdb:2.4.4` container and exposes `containerHost`/`containerPort`; auth/permission tests use their own isolated container via `startRethinkDBWithPassword(t, password)` (not the shared one); helpers: `defaultCfg()`, `newExecutor()`, `closeCursor(cur)`, `setupTestDB(t, exec, dbName)`, `createTestTable(t, exec, dbName, tableName)`, `startRethinkDBWithPassword(t, password)`, `dialAs(ctx, host, port, user, password)`, `execAs(t, host, port, user, password)`, `createUser(t, exec, username, password)`, `isPermissionError(err)`; write operation results parsed via `writeResult` struct / `parseWriteResult` helper; tests cover connection/handshake, server info, database/table CRUD, document CRUD, filter, get/getAll, update/replace/delete, SCRAM-SHA-256 auth (correct/wrong/nonexistent credentials, password change, special chars, empty password), global/db-level/table-level permission grants and revocations, permission inheritance and override, user deletion and cleanup
- `cmd/r-cli` - CLI entry point; persistent global flags: `-H/--host` (localhost), `-P/--port` (28015), `-d/--db`, `-u/--user` (admin), `-p/--password`, `--password-file`, `-t/--timeout` (30s, applied via context.WithTimeout), `-f/--format` (empty = auto: json on TTY, jsonl when piped), `--profile` (enable query profiling output), `--time-format` (native|raw, default native; native converts TIME pseudo-types to time.Time), `--binary-format` (native|raw, default native; native converts BINARY pseudo-types to []byte), `--quiet` (suppress non-data stderr output), `--verbose` (show connection info and query timing on stderr), `--tls-cert` (path to CA certificate PEM file), `--tls-client-cert` (path to client certificate PEM file), `--tls-key` (path to client private key; must be used with `--tls-client-cert`), `--insecure-skip-verify` (skip TLS certificate verification); env vars `RETHINKDB_HOST/PORT/USER/PASSWORD/DATABASE` override defaults (CLI flag wins); exit codes: 0 ok, 1 connection, 2 query, 3 auth, 130 SIGINT/SIGTERM; `PersistentPreRunE` calls `resolveEnvVars` + `resolvePassword` for every subcommand; root command itself acts as implicit `query` when invoked with an expression arg or piped stdin (Args: cobra.ArbitraryArgs, RunE delegates to readQueryExpr/runQueryExpr; starts REPL when called on interactive TTY with no args); `stdinIsTTY` package-level var reports whether stdin is a terminal and is replaceable in tests; `newExecutor(cfg)` shared helper builds `*query.Executor` and returns a cleanup func and error (returns error if TLS config is invalid); `execTerm` shared helper connects, runs a ReQL term, writes formatted output; subcommands: `query [expression]` (executes a ReQL expression; input priority: arg > stdin; `-F/--file` reads from file; file supports multiple queries separated by `---` on its own line; `--stop-on-error` stops on first failure in file mode, default continues and prints each error to stderr; `--file` and expression arg are mutually exclusive), `run` (raw ReQL JSON term from arg or stdin), `db list/create/drop` (drop has `--yes/-y` to skip confirmation), `table list/create/drop/info/reconfigure/rebalance/wait/sync` (requires `--db`; `reconfigure` accepts `--shards`, `--replicas`, `--dry-run`), `index list/create/drop/rename/status/wait` (requires `--db`; `create` accepts `--geo`, `--multi`), `user list/create/delete/set-password` (`create` accepts `--new-password`; prompts with no-echo on TTY if omitted; `delete` has `--yes/-y`), `grant <user>` (top-level; `--read`, `--write`, `--table`; scope: global / `--db` / `--db --table`; `--read=false` revokes), `insert <db.table>` (`-F/--file`, `--batch-size` default 200, `--conflict error|replace|update`; reads JSONL from stdin or JSON/JSONL from file; format from flag or `.json` extension; prints `{"inserted":N,"errors":N}`), `status` (server info as JSON), `repl` (start an interactive REPL; no extra flags beyond inherited globals; auto-started by root command when stdin is a TTY and no args given), `completion bash/zsh/fish` (cobra built-in); `confirmDrop` reads y/yes from io.Reader for destructive operations; `promptPassword` prompts on stderr, reads without echo on TTY (via `golang.org/x/term`), falls back to line-read for non-TTY; format auto-detection uses `output.DetectFormat(os.Stdout, cfg.format)` - explicit flag always wins, empty default triggers TTY check

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
- Fuzz tests: `func FuzzXxx(f *testing.F)` with `f.Add(seed...)` seed corpus; run with `go test -fuzz=FuzzXxx ./path/...`
- Integration tests require Docker; run with `make test-integration` (uses `-tags integration`) or `go test -tags integration ./internal/integration/... -race -count=1`; run all tests: `make test-all`
- Readline tests: `internal/repl/readline_test.go` uses `//go:build !race` due to a known data race inside `github.com/chzyer/readline` between `Terminal.ioloop()` and `Terminal.Close()`; do not remove this constraint

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
