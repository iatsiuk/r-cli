# R-CLI Implementation Plan

RethinkDB query CLI tool with a built-in driver (no third-party RethinkDB dependencies).

## Architecture

```
cmd/r-cli/main.go          -- entry point, cobra commands, flags
internal/
  proto/                    -- protocol constants (term types, response types, query types)
  wire/                     -- message framing: encode/decode token+length+json
  scram/                    -- SCRAM-SHA-256 authentication (RFC 5802 / RFC 7677)
  conn/                     -- TCP connection, TLS, handshake, multiplexed send/receive
  reql/                     -- ReQL term builder, JSON serialization, query parser
  response/                 -- response parsing, pseudo-type conversion, error mapping
  cursor/                   -- cursor/streaming for partial results
  connmgr/                  -- single connection manager with lazy connect and reconnect
  query/                    -- high-level query executor (combines connmgr + reql + cursor)
  output/                   -- result formatting (JSON, JSONL, table, raw)
  repl/                     -- interactive REPL (readline, history, completion)
```

## Phases

Each phase follows TDD: write failing tests first, then implement until tests pass.

---

### Phase 1: Protocol Constants

Package: `internal/proto`

Define all protocol constants from `ql2.proto` as typed Go constants.

#### 1.1 Version constants

- [ ] Test: verify magic number values match spec (V1_0 = 0x34c2bdc3, etc.)
- [ ] Implement: `version.go` -- Version type + constants

#### 1.2 Query types

- [ ] Test: verify QueryType values (START=1, CONTINUE=2, STOP=3, NOREPLY_WAIT=4, SERVER_INFO=5)
- [ ] Implement: `query.go` -- QueryType type + constants

#### 1.3 Response types

- [ ] Test: verify ResponseType values (SUCCESS_ATOM=1 .. RUNTIME_ERROR=18)
- [ ] Test: `IsError()` method returns true for types >= 16
- [ ] Implement: `response.go` -- ResponseType, ErrorType, ResponseNote types + constants

#### 1.4 Term types

- [ ] Test: verify core term values (DB=14, TABLE=15, FILTER=39, INSERT=56, etc.)
- [ ] Implement: `term.go` -- TermType type + all constants grouped by category

#### 1.5 Datum types

- [ ] Test: verify DatumType values (R_NULL=1 .. R_JSON=7)
- [ ] Implement: `datum.go` -- DatumType type + constants

---

### Phase 2: Wire Protocol (Message Framing)

Package: `internal/wire`

Binary message encoding/decoding: 8-byte token + 4-byte length + JSON payload.

#### 2.1 Encode query message

- [ ] Test: encode token=1, payload `[1,"foo",{}]` -> expected bytes (LE token + LE length + JSON)
- [ ] Test: encode token=0 (edge case)
- [ ] Test: encode large payload (verify length field correctness)
- [ ] Implement: `Encode(token uint64, payload []byte) []byte`

#### 2.2 Decode response header

- [ ] Test: decode 12-byte header -> token + payload length
- [ ] Test: decode with insufficient bytes -> error
- [ ] Implement: `DecodeHeader(data [12]byte) (token uint64, length uint32)`

#### 2.3 Read full response from reader

- [ ] Test: read header + payload from `bytes.Reader` -> token + JSON
- [ ] Test: read from reader that returns partial data (simulate slow network)
- [ ] Test: read from reader that returns EOF mid-header -> error
- [ ] Test: payload length > MaxFrameSize (64MB) -> error (prevent OOM)
- [ ] Implement: `ReadResponse(r io.Reader) (token uint64, payload []byte, err error)`

#### 2.4 Write query to writer

- [ ] Test: write query message to `bytes.Buffer`, verify bytes
- [ ] Implement: `WriteQuery(w io.Writer, token uint64, payload []byte) error`

---

### Phase 3: SCRAM-SHA-256 Authentication

Package: `internal/scram`

Implements SCRAM-SHA-256 per RFC 5802 / RFC 7677.

#### 3.1 Nonce generation

- [x] Test: generated nonce is at least 18 bytes, base64-encoded, no commas
- [x] Implement: `GenerateNonce() string`

#### 3.2 Client-first-message

- [x] Test: build message with known user and nonce, verify format `n,,n=<user>,r=<nonce>`
- [x] Test: username with special characters (=, ,) is properly escaped
- [x] Implement: `ClientFirstMessage(user, nonce string) string`

#### 3.3 Parse server-first-message

- [x] Test: parse `r=<nonce>,s=<salt>,i=<iter>` -> nonce, salt bytes, iteration count
- [x] Test: parse malformed message -> error
- [x] Test: parse message with wrong nonce prefix -> error
- [x] Implement: `ParseServerFirst(msg, clientNonce string) (*ServerFirst, error)`

#### 3.4 SCRAM proof computation

- [x] Test: compute ClientProof with known inputs (use RFC 7677 test vectors)
- [x] Test: compute ServerSignature with known inputs
- [x] Implement: `ComputeProof(password string, salt []byte, iter int, authMsg string) (clientProof, serverSig []byte)`

#### 3.5 Client-final-message

- [x] Test: build message with known combined nonce and proof, verify format
- [x] Implement: `ClientFinalMessage(combinedNonce string, proof []byte) string`

#### 3.6 Verify server-final

- [x] Test: verify correct server signature -> success
- [x] Test: verify wrong server signature -> error
- [x] Implement: `VerifyServerFinal(msg string, expectedSig []byte) error`

#### 3.7 Full SCRAM conversation (integration)

- [x] Test: simulate full 3-step exchange with hardcoded messages, verify all outputs
- [x] Implement: `Conversation` struct that tracks state across steps

---

### Phase 4: Connection and Handshake

Package: `internal/conn`

TCP connection with V1_0 handshake. Multiplexed query dispatch on a single connection.

#### 4.1 Null-terminated message framing

Handshake uses null-terminated JSON messages (not token+length framing from Phase 2).

- [ ] Test: `readNullTerminated` reads until `\x00`, returns data without terminator
- [ ] Test: `readNullTerminated` with data arriving in 1-byte chunks (partial reads)
- [ ] Test: `readNullTerminated` on EOF before `\x00` -> error
- [ ] Test: `readNullTerminated` exceeding maxHandshakeSize (16KB) -> error (prevent OOM)
- [ ] Test: `writeNullTerminated` appends `\x00` to output
- [ ] Implement: `readNullTerminated(r io.Reader) ([]byte, error)`, `writeNullTerminated(w io.Writer, data []byte) error`
- maxHandshakeSize = 16384 bytes; `readNullTerminated` returns error if exceeded

#### 4.2 Handshake message building

- [ ] Test: build step 1 bytes (magic number LE)
- [ ] Test: build step 3 JSON (protocol_version, authentication_method, authentication) + `\x00`
- [ ] Test: build step 5 JSON (client-final-message) + `\x00`
- [ ] Implement: handshake message builders

#### 4.3 Handshake response parsing

- [ ] Test: parse step 2 JSON -> server version, protocol range
- [ ] Test: parse step 2 non-JSON error string -> error
- [ ] Test: parse step 4 success -> extract authentication field
- [ ] Test: parse step 4 with error_code 10-20 -> ReqlAuthError
- [ ] Test: parse step 6 success -> extract server signature
- [ ] Implement: handshake response parsers

#### 4.4 Full handshake over mock connection

- [ ] Test: simulate full 6-step handshake using `net.Pipe()`, verify all messages
- [ ] Test: handshake with wrong password -> auth error
- [ ] Test: handshake with incompatible protocol version -> error
- [ ] Test: pipelined handshake (steps 1+3 sent together, then read steps 2+4) reduces RTT
- [ ] Implement: `Handshake(rw io.ReadWriter, user, password string) error`

#### 4.5 Token counter

- [ ] Test: sequential tokens from same connection are monotonically increasing
- [ ] Test: concurrent token generation is safe (no duplicates)
- [ ] Implement: atomic uint64 counter in `Conn`

#### 4.6 Connection struct and multiplexing

Architecture: `Conn` owns a `net.Conn` and runs a background `readLoop` goroutine.
- `readLoop` continuously reads wire frames, extracts token, parses minimal
  envelope (`t` and `n` fields only), dispatches `RawResponse{Token, Type, Notes,
  Payload []byte}` to the correct waiter via `map[uint64]chan RawResponse`
  (guarded by mutex). Full response parsing (pseudo-types, error mapping) happens
  above in cursor/query layer.
- Dispatch channels are **buffered (size 1)**. `readLoop` uses non-blocking send
  via `select`/`default`; if the channel is full (slow consumer), log warning and
  drop the response. This prevents one slow token from blocking all others.
- `Send()` registers a response channel in the dispatch map, acquires a write
  mutex, and writes the framed query to TCP.
- For one-shot queries (SUCCESS_ATOM, SUCCESS_SEQUENCE, errors): dispatch map
  entry is removed after delivering the response.
- For streaming queries (SUCCESS_PARTIAL): dispatch map entry stays until
  SUCCESS_SEQUENCE, error, or STOP.
- Late responses (token already removed from dispatch map after STOP/cancel):
  silently discarded, no panic.
- `Close()` stops readLoop, sets closed flag (rejects new `Send()` calls),
  and unblocks all pending waiters with a closed-connection error.
- `Dial()` accepts optional `*tls.Config` parameter (nil = plain TCP).
  TLS wrapping implementation deferred to Phase 14.
- `Config.String()` masks password as "***" to prevent leaks in logs/panic traces.
- Debug wire dump: when `RCLI_DEBUG=wire` env var is set, hex-dump all sent and
  received frames to stderr.

Tests:

- [ ] Test: connect to mock server (net.Pipe), handshake, send query, receive response
- [ ] Test: concurrent queries on same connection -> each receives its own response
- [ ] Test: out-of-order responses (server replies token 2 before token 1) -> correct dispatch
- [ ] Test: slow consumer on token 1 does not block delivery to token 2
- [ ] Test: late response after STOP (token removed) -> silently discarded, no panic
- [ ] Test: close connection unblocks all pending waiters with error
- [ ] Test: Send() after Close() returns error immediately
- [ ] Test: context cancellation during query sends STOP and cleans up dispatch entry
- [ ] Test: context cancellation during handshake -> no goroutine leak
- [ ] Test: STOP sent while server sends one more SUCCESS_PARTIAL -> no deadlock
- [ ] Test: Config.String() does not contain password
- [ ] Implement: `Conn` struct with `Dial()`, `Close()`, `Send()`, background `readLoop`

---

### Phase 5: ReQL Term Builder

Package: `internal/reql`

Builds ReQL terms as JSON-serializable structures.

#### 5.1 Datum encoding

- [ ] Test: string "foo" -> `"foo"` (raw JSON)
- [ ] Test: number 42 -> `42`
- [ ] Test: bool true -> `true`
- [ ] Test: nil -> `null`
- [ ] Implement: datum pass-through in term serialization

#### 5.2 MAKE_ARRAY wrapping

- [ ] Test: Go slice `[10,20,30]` -> `[2,[10,20,30]]`
- [ ] Test: empty slice -> `[2,[]]`
- [ ] Test: nested array -> properly wrapped
- [ ] Implement: `Array(items ...interface{}) Term`

#### 5.3 Core term builder

- [ ] Test: `DB("test")` -> `[14,["test"]]`
- [ ] Test: `DB("test").Table("users")` -> `[15,[[14,["test"]],"users"]]`
- [ ] Test: chained `.Filter({...})` -> correct nested structure
- [ ] Implement: `Term` struct with chainable methods, `MarshalJSON()`

#### 5.4 Write operations

- [ ] Test: `.Insert(doc)` -> `[56,[<table_term>,<doc>]]`
- [ ] Test: `.Update(doc)` -> `[53,[<table_term>,<doc>]]`
- [ ] Test: `.Delete()` -> `[54,[<table_term>]]`
- [ ] Test: `.Replace(doc)` -> `[55,[<table_term>,<doc>]]`
- [ ] Implement: Insert, Update, Delete, Replace methods

#### 5.5 Read operations

- [ ] Test: `.Get(key)` -> `[16,[<table_term>,<key>]]`
- [ ] Test: `.GetAll(keys..., index)` -> correct term with optional index arg
- [ ] Test: `.Between(lower, upper)` -> `[182,[<term>,<lower>,<upper>]]`
- [ ] Test: `.OrderBy(field)` -> `[41,[<term>,<field>]]` with ASC/DESC
- [ ] Test: `.Limit(n)` -> `[71,[<term>,<n>]]`
- [ ] Test: `.Skip(n)` -> `[70,[<term>,<n>]]`
- [ ] Test: `.Count()` -> `[43,[<term>]]`
- [ ] Test: `.Pluck(fields...)` -> `[33,[<term>,<fields>...]]`
- [ ] Test: `.Without(fields...)` -> `[34,[<term>,<fields>...]]`
- [ ] Implement: all read operation methods

#### 5.6 Comparison and logic operators

- [ ] Test: `.Eq(value)` -> `[17,[<term>,<value>]]`
- [ ] Test: `.Ne(value)` -> `[18,[<term>,<value>]]`
- [ ] Test: `.Lt(value)` -> `[19,[<term>,<value>]]`
- [ ] Test: `.Le(value)` -> `[20,[<term>,<value>]]`
- [ ] Test: `.Gt(value)` -> `[21,[<term>,<value>]]`
- [ ] Test: `.Ge(value)` -> `[22,[<term>,<value>]]`
- [ ] Test: `.Not()` -> `[23,[<term>]]`
- [ ] Test: `.And(other)` -> `[67,[<term>,<other>]]`
- [ ] Test: `.Or(other)` -> `[66,[<term>,<other>]]`
- [ ] Implement: comparison and logic operator methods

#### 5.7 Object operations

- [ ] Test: `.GetField("name")` -> `[31,[<term>,"name"]]`
- [ ] Test: `.HasFields("a","b")` -> `[32,[<term>,"a","b"]]`
- [ ] Test: `.Merge(obj)` -> `[35,[<term>,<obj>]]`
- [ ] Test: `.Distinct()` -> `[42,[<term>]]`
- [ ] Implement: object operation methods

#### 5.8 Aggregation

- [ ] Test: `.Map(func)` -> `[38,[<term>,<func>]]`
- [ ] Test: `.Reduce(func)` -> `[37,[<term>,<func>]]`
- [ ] Test: `.Group(field)` -> `[144,[<term>,<field>]]`
- [ ] Test: `.Ungroup()` -> `[150,[<term>]]`
- [ ] Test: `.Sum(field)` -> `[145,[<term>,<field>]]`
- [ ] Test: `.Avg(field)` -> `[146,[<term>,<field>]]`
- [ ] Test: `.Min(field)` -> `[147,[<term>,<field>]]`
- [ ] Test: `.Max(field)` -> `[148,[<term>,<field>]]`
- [ ] Implement: aggregation methods

#### 5.9 Arithmetic

- [ ] Test: `.Add(value)` -> `[24,[<term>,<value>]]`
- [ ] Test: `.Sub(value)` -> `[25,[<term>,<value>]]`
- [ ] Test: `.Mul(value)` -> `[26,[<term>,<value>]]`
- [ ] Test: `.Div(value)` -> `[27,[<term>,<value>]]`
- [ ] Implement: arithmetic methods

#### 5.10 Index operations

- [ ] Test: `.IndexCreate("name")` -> `[75,[<table_term>,"name"]]`
- [ ] Test: `.IndexDrop("name")` -> `[76,[<table_term>,"name"]]`
- [ ] Test: `.IndexList()` -> `[77,[<table_term>]]`
- [ ] Test: `.IndexWait("name")` -> `[140,[<table_term>,"name"]]`
- [ ] Test: `.IndexStatus("name")` -> `[139,[<table_term>,"name"]]`
- [ ] Test: `.IndexRename("old","new")` -> `[156,[<table_term>,"old","new"]]`
- [ ] Implement: index operation methods

#### 5.11 Changefeed and misc

- [ ] Test: `.Changes()` -> `[152,[<term>]]`
- [ ] Test: `.Changes()` with optarg `include_initial=true`
- [ ] Test: `Now()` -> `[103,[]]`
- [ ] Test: `UUID()` -> `[169,[]]`
- [ ] Test: `Binary(data)` -> `[155,[<data>]]`
- [ ] Test: `.Config()` -> `[174,[<term>]]`
- [ ] Test: `.Status()` -> `[175,[<term>]]`
- [ ] Test: `Grant("user", perms)` -> `[188,[<scope>,"user",<perms>]]`
- [ ] Implement: changefeed, time, binary, admin term methods

#### 5.12 Function serialization

- [ ] Test: single-arg function -> `[69,[[2,[1]],<body>]]`
- [ ] Test: multi-arg function -> correct param IDs
- [ ] Test: VAR reference -> `[10,[<id>]]`
- [ ] Implement: `Func` builder with VAR references

#### 5.13 IMPLICIT_VAR auto-wrapping

The driver must detect IMPLICIT_VAR (term 13) in term arguments,
replace it with VAR(1), and wrap the argument in FUNC(69).
See rethink-driver.md section 6: "the driver must wrap it".

- [ ] Test: term containing `[13,[]]` is wrapped -> `[69,[[2,[1]],<body_with_var_1>]]`
- [ ] Test: nested IMPLICIT_VAR in deeply nested term -> correctly replaced at all levels
- [ ] Test: IMPLICIT_VAR in nested function context -> error (ambiguous per spec)
- [ ] Test: term without IMPLICIT_VAR -> no wrapping applied
- [ ] Implement: `wrapImplicitVar(term Term) Term` tree traversal

#### 5.14 FUNCALL (r.do) argument reordering

API order: `Do(arg1, arg2, func)`. Wire order: `[64, [func, arg1, arg2]]`.
Function goes first on the wire. See rethink-driver.md section 7.

- [ ] Test: `Do(10, 20, func)` -> `[64,[<func>,10,20]]`
- [ ] Test: `Do(func)` with no extra args -> `[64,[<func>]]`
- [ ] Implement: `Do` builder with argument reordering

#### 5.15 Database/table admin

- [ ] Test: `DBCreate("name")` -> `[57,["name"]]`
- [ ] Test: `DBDrop("name")` -> `[58,["name"]]`
- [ ] Test: `DBList()` -> `[59,[]]`
- [ ] Test: `TableCreate("name")` -> `[60,[<db_term>,"name"]]`
- [ ] Test: `TableDrop("name")` -> `[61,[<db_term>,"name"]]`
- [ ] Test: `TableList()` -> `[62,[<db_term>]]`
- [ ] Implement: admin term builders

#### 5.16 Term optargs

- [ ] Test: `.Insert(doc)` with `conflict` optarg -> `[56,[<table>,<doc>],{"conflict":"replace"}]`
- [ ] Test: `.Insert(doc)` with `return_changes` optarg
- [ ] Test: `.Changes()` with `include_initial` optarg
- [ ] Test: `.TableCreate("name")` with `primary_key` optarg
- [ ] Test: `.OrderBy()` with `index` optarg
- [ ] Implement: optarg support on all term methods that need it

#### 5.17 Full query serialization

- [ ] Test: wrap term in START query -> `[1,<term>,<optargs>]`
- [ ] Test: query with `db` optarg -> db value wrapped as DB term
- [ ] Test: CONTINUE query -> `[2]`
- [ ] Test: STOP query -> `[3]`
- [ ] Implement: `BuildQuery(queryType, term, opts) []byte`

#### 5.18 Join operations

- [ ] Test: `INNER_JOIN(48)` -> `[48,[<seq>,<seq>,<func>]]`
- [ ] Test: `OUTER_JOIN(49)` -> `[49,[<seq>,<seq>,<func>]]`
- [ ] Test: `EQ_JOIN(50)` with index optarg -> `[50,[<seq>,"field",<table>],{"index":"name"}]`
- [ ] Test: `ZIP(72)` -> `[72,[<term>]]`
- [ ] Test: eqJoin with index optarg, innerJoin with predicate function, zip after join
- [ ] Implement: InnerJoin, OuterJoin, EqJoin, Zip methods

#### 5.19 String operations

- [ ] Test: `MATCH(97)` -> `[97,[<term>,"pattern"]]`
- [ ] Test: `SPLIT(149)` with delimiter -> `[149,[<term>,"delim"]]`
- [ ] Test: `SPLIT(149)` without delimiter -> `[149,[<term>]]`
- [ ] Test: `UPCASE(141)` -> `[141,[<term>]]`
- [ ] Test: `DOWNCASE(142)` -> `[142,[<term>]]`
- [ ] Test: `TO_JSON_STRING(172)` -> `[172,[<term>]]`
- [ ] Test: `JSON(98)` -> `[98,["json_string"]]`
- [ ] Implement: Match, Split, Upcase, Downcase, ToJsonString, Json methods

#### 5.20 Time operations

Construction:

- [ ] Test: `ISO8601(99)` -> `[99,["2024-01-01T00:00:00Z"]]`
- [ ] Test: `EPOCH_TIME(101)` -> `[101,[1234567890]]`
- [ ] Test: `TIME(136)` -> `[136,[2024,1,1,"Z"]]`
- [ ] Test: `NOW(103)` -> `[103,[]]` (already exists in 5.11, keep reference)

Extraction:

- [ ] Test: `TO_ISO8601(100)` -> `[100,[<time_term>]]`
- [ ] Test: `TO_EPOCH_TIME(102)` -> `[102,[<time_term>]]`
- [ ] Test: `DATE(106)` -> `[106,[<time_term>]]`
- [ ] Test: `TIME_OF_DAY(126)` -> `[126,[<time_term>]]`
- [ ] Test: `TIMEZONE(127)` -> `[127,[<time_term>]]`
- [ ] Test: `YEAR(128)` -> `[128,[<time_term>]]`
- [ ] Test: `MONTH(129)` -> `[129,[<time_term>]]`
- [ ] Test: `DAY(130)` -> `[130,[<time_term>]]`
- [ ] Test: `DAY_OF_WEEK(131)` -> `[131,[<time_term>]]`
- [ ] Test: `DAY_OF_YEAR(132)` -> `[132,[<time_term>]]`
- [ ] Test: `HOURS(133)` -> `[133,[<time_term>]]`
- [ ] Test: `MINUTES(134)` -> `[134,[<time_term>]]`
- [ ] Test: `SECONDS(135)` -> `[135,[<time_term>]]`

Operations:

- [ ] Test: `IN_TIMEZONE(104)` -> `[104,[<time_term>,"+02:00"]]`
- [ ] Test: `DURING(105)` -> `[105,[<time_term>,<start>,<end>]]`

Constants:

- [ ] Test: `MONDAY(107)` through `SUNDAY(113)` -> `[107,[]]` .. `[113,[]]`
- [ ] Test: `JANUARY(114)` through `DECEMBER(125)` -> `[114,[]]` .. `[125,[]]`
- [ ] Implement: all time construction, extraction, operation, and constant methods

#### 5.21 Array operations

- [ ] Test: `APPEND(29)` -> `[29,[<term>,<value>]]`
- [ ] Test: `PREPEND(80)` -> `[80,[<term>,<value>]]`
- [ ] Test: `SLICE(30)` -> `[30,[<term>,<start>,<end>]]`
- [ ] Test: `DIFFERENCE(95)` -> `[95,[<term>,<array>]]`
- [ ] Test: `INSERT_AT(82)` -> `[82,[<term>,<index>,<value>]]`
- [ ] Test: `DELETE_AT(83)` -> `[83,[<term>,<index>]]`
- [ ] Test: `CHANGE_AT(84)` -> `[84,[<term>,<index>,<value>]]`
- [ ] Test: `SPLICE_AT(85)` -> `[85,[<term>,<index>,<array>]]`
- [ ] Test: `SET_INSERT(88)` -> `[88,[<term>,<value>]]`
- [ ] Test: `SET_INTERSECTION(89)` -> `[89,[<term>,<array>]]`
- [ ] Test: `SET_UNION(90)` -> `[90,[<term>,<array>]]`
- [ ] Test: `SET_DIFFERENCE(91)` -> `[91,[<term>,<array>]]`
- [ ] Implement: Append, Prepend, Slice, Difference, InsertAt, DeleteAt, ChangeAt, SpliceAt, SetInsert, SetIntersection, SetUnion, SetDifference methods

#### 5.22 Control flow

- [ ] Test: `BRANCH(65)` -> `[65,[<cond>,<true_val>,<false_val>]]`
- [ ] Test: `BRANCH(65)` with multiple condition pairs -> `[65,[<c1>,<v1>,<c2>,<v2>,<else>]]`
- [ ] Test: `FOR_EACH(68)` -> `[68,[<seq>,<func>]]`
- [ ] Test: `DEFAULT(92)` -> `[92,[<term>,<default_val>]]`
- [ ] Test: `ERROR(12)` -> `[12,["message"]]`
- [ ] Test: `COERCE_TO(51)` -> `[51,[<term>,"string"]]`
- [ ] Test: `TYPE_OF(52)` -> `[52,[<term>]]`
- [ ] Implement: Branch, ForEach, Default, Error, CoerceTo, TypeOf methods

#### 5.23 Additional sequence operations

- [ ] Test: `CONCAT_MAP(40)` -> `[40,[<seq>,<func>]]`
- [ ] Test: `NTH(45)` -> `[45,[<seq>,<index>]]`
- [ ] Test: `UNION(44)` -> `[44,[<seq>,<seq>]]`
- [ ] Test: `UNION(44)` with multiple sequences -> `[44,[<seq>,<seq>,<seq>]]`
- [ ] Test: `IS_EMPTY(86)` -> `[86,[<seq>]]`
- [ ] Test: `CONTAINS(93)` -> `[93,[<seq>,<value>]]`
- [ ] Test: `BRACKET(170)` -> `[170,[<term>,"field"]]`
- [ ] Test: `BRACKET(170)` chained -> `[170,[[170,[<term>,"a"]],"b"]]`
- [ ] Implement: ConcatMap, Nth, Union, IsEmpty, Contains, Bracket methods

#### 5.24 Additional object operations

- [ ] Test: `WITH_FIELDS(96)` -> `[96,[<seq>,"field1","field2"]]`
- [ ] Test: `KEYS(94)` -> `[94,[<term>]]`
- [ ] Test: `VALUES(186)` -> `[186,[<term>]]`
- [ ] Test: `LITERAL(137)` -> `[137,[<value>]]`
- [ ] Test: `LITERAL(137)` in merge -> merge uses literal to replace nested field
- [ ] Implement: WithFields, Keys, Values, Literal methods

#### 5.25 Admin operations

- [ ] Test: `SYNC(138)` -> `[138,[<table_term>]]`
- [ ] Test: `RECONFIGURE(176)` -> `[176,[<table_term>],{"shards":2,"replicas":1}]`
- [ ] Test: `REBALANCE(179)` -> `[179,[<table_term>]]`
- [ ] Test: `WAIT(177)` -> `[177,[<table_term>]]`
- [ ] Test: `ARGS(154)` -> `[154,[<array>]]`
- [ ] Test: `MINVAL(180)` -> `[180,[]]`
- [ ] Test: `MAXVAL(181)` -> `[181,[]]`
- [ ] Test: Between with minval/maxval -> `[182,[<seq>,[180,[]],[181,[]]]]`
- [ ] Implement: Sync, Reconfigure, Rebalance, Wait, Args, MinVal, MaxVal methods

#### 5.26 Geospatial operations

- [ ] Test: `GEOJSON(157)` -> `[157,[<geojson_obj>]]`
- [ ] Test: `TO_GEOJSON(158)` -> `[158,[<geo_term>]]`
- [ ] Test: `POINT(159)` -> `[159,[-122.4,37.7]]`
- [ ] Test: `LINE(160)` -> `[160,[[159,[-122.4,37.7]],[159,[-122.3,37.8]]]]`
- [ ] Test: `POLYGON(161)` -> `[161,[[159,[...]],[159,[...]],[159,[...]]]]`
- [ ] Test: `CIRCLE(165)` -> `[165,[[159,[-122.4,37.7]],1000],{"unit":"m"}]`
- [ ] Test: `DISTANCE(162)` -> `[162,[<geo1>,<geo2>],{"unit":"km"}]`
- [ ] Test: `INTERSECTS(163)` -> `[163,[<geo1>,<geo2>]]`
- [ ] Test: `INCLUDES(164)` -> `[164,[<geo>,<point>]]`
- [ ] Test: `GET_INTERSECTING(166)` with index optarg -> `[166,[<table>,<geo>],{"index":"location"}]`
- [ ] Test: `GET_NEAREST(168)` with index optarg -> `[168,[<table>,<point>],{"index":"location"}]`
- [ ] Test: `FILL(167)` -> `[167,[<line_term>]]`
- [ ] Test: `POLYGON_SUB(171)` -> `[171,[<polygon1>,<polygon2>]]`
- [ ] Implement: GeoJson, ToGeoJson, Point, Line, Polygon, Circle, Distance, Intersects, Includes, GetIntersecting, GetNearest, Fill, PolygonSub methods

#### 5.27 Additional arithmetic

- [ ] Test: `MOD(28)` -> `[28,[<term>,<value>]]`
- [ ] Test: `FLOOR(183)` -> `[183,[<term>]]`
- [ ] Test: `CEIL(184)` -> `[184,[<term>]]`
- [ ] Test: `ROUND(185)` -> `[185,[<term>]]`
- [ ] Implement: Mod, Floor, Ceil, Round methods

---

### Phase 6: Response Parsing

Package: `internal/response`

Separate from `conn` to keep transport and interpretation concerns apart.

#### 6.1 Response struct

- [ ] Test: unmarshal `{"t":1,"r":["foo"]}` -> ResponseType=SUCCESS_ATOM, results=["foo"]
- [ ] Test: unmarshal error response with `e` and `b` fields
- [ ] Test: unmarshal response with `n` (notes) field
- [ ] Test: unmarshal response with `p` (profile) field
- [ ] Implement: `Response` struct with JSON unmarshaling

#### 6.2 Pseudo-type conversion

- [ ] Test: TIME pseudo-type -> Go `time.Time`
- [ ] Test: BINARY pseudo-type -> Go `[]byte`
- [ ] Test: nested pseudo-types in result documents
- [ ] Test: GEOMETRY pseudo-type -> pass-through as GeoJSON object (no conversion needed)
- [ ] Test: nested GEOMETRY in result documents
- [ ] Test: pass-through when conversion disabled
- [ ] Implement: `ConvertPseudoTypes(v interface{}) interface{}`

#### 6.3 Error mapping

- [ ] Test: CLIENT_ERROR (16) -> ReqlClientError
- [ ] Test: COMPILE_ERROR (17) -> ReqlCompileError
- [ ] Test: RUNTIME_ERROR (18) -> ReqlRuntimeError
- [ ] Test: RUNTIME_ERROR with ErrorType NON_EXISTENCE -> ReqlNonExistenceError
- [ ] Test: RUNTIME_ERROR with ErrorType PERMISSION_ERROR -> ReqlPermissionError
- [ ] Test: backtrace included in error message
- [ ] Implement: error types and mapping function

---

### Phase 7: Cursor and Streaming

Package: `internal/cursor`

Cursor receives data from `conn` via a response channel tied to the query token.
For streaming cursors, `conn` keeps the dispatch map entry alive until
SUCCESS_SEQUENCE, error, or explicit STOP. Cursor sends CONTINUE/STOP through
an interface (not directly to TCP) to avoid tight coupling with `conn` internals.

#### 7.1 Atom cursor (single result)

- [ ] Test: create from SUCCESS_ATOM response, read single value, then EOF
- [ ] Test: `All()` returns single-element slice
- [ ] Implement: atom cursor

#### 7.2 Sequence cursor (finite)

- [ ] Test: create from SUCCESS_SEQUENCE, iterate all items
- [ ] Test: `All()` collects everything
- [ ] Implement: sequence cursor

#### 7.3 Streaming cursor (partial results)

- [ ] Test: SUCCESS_PARTIAL triggers CONTINUE, next batch arrives, ends with SUCCESS_SEQUENCE
- [ ] Test: premature `Close()` sends STOP
- [ ] Test: context cancellation sends STOP
- [ ] Test: concurrent `Next()` calls are safe
- [ ] Implement: streaming cursor with CONTINUE/STOP lifecycle

#### 7.4 Changefeed cursor

- [ ] Test: infinite SUCCESS_PARTIAL stream, values arrive incrementally
- [ ] Test: `Close()` sends STOP and terminates
- [ ] Test: connection drop -> error on next `Next()`
- [ ] Implement: changefeed cursor (never auto-completes)

---

### Phase 8: Connection Manager

Package: `internal/connmgr`

Single multiplexed connection with lazy connect and automatic reconnect.
RethinkDB supports thousands of concurrent tokens per connection --
a full pool is unnecessary for a CLI tool.

#### 8.1 Lazy connect

- [ ] Test: `Get()` on fresh manager creates connection on first call
- [ ] Test: subsequent `Get()` returns the same connection (no reconnect)
- [ ] Test: `Close()` closes the underlying connection
- [ ] Implement: `ConnManager` struct with `Get(ctx) (*Conn, error)`, `Close()`

#### 8.2 Reconnect on failure

- [ ] Test: `Get()` after connection drop -> reconnects automatically
- [ ] Test: `Get()` during server downtime -> returns dial error
- [ ] Test: reconnect preserves config (host, port, user, password, tls)
- [ ] Implement: detect closed/errored connection in `Get()`, re-dial

---

### Phase 9: Query Executor

Package: `internal/query`

High-level API combining connmgr + reql + cursor.

#### 9.1 Execute query

- [ ] Test: execute `r.db("test").table("users")` against mock server, get cursor
- [ ] Test: execute with `db` option
- [ ] Test: execute with timeout
- [ ] Test: execute with noreply
- [ ] Implement: `Executor` struct with `Run(ctx, term, opts) (*Cursor, error)`

#### 9.2 Server info

- [ ] Test: `ServerInfo()` returns server name and ID
- [ ] Implement: `ServerInfo(ctx) (*ServerInfo, error)`

---

### Phase 10: Output Formatting

Package: `internal/output`

Formatters accept a `RowIterator` interface (cursor-like) to support streaming
without buffering entire result sets in memory. For atom results, a single-value
iterator is used.

#### 10.1 JSON output

- [ ] Test: format single document as pretty JSON
- [ ] Test: format array of documents as streaming JSON array
- [ ] Test: format empty result
- [ ] Implement: `JSON(w io.Writer, iter RowIterator) error`

#### 10.2 Raw output

- [ ] Test: format single value as plain string
- [ ] Test: format each row on separate line (streaming)
- [ ] Implement: `Raw(w io.Writer, iter RowIterator) error`

#### 10.3 Table output

- [ ] Test: format array of objects as aligned ASCII table
- [ ] Test: handle missing fields (fill with empty)
- [ ] Test: truncate long values
- [ ] Test: handle non-object results (fallback to raw)
- [ ] Test: rows exceeding maxTableRows (10000) -> truncate with warning to stderr
- [ ] Implement: `Table(w io.Writer, iter RowIterator) error` (buffers up to maxTableRows=10000)

#### 10.4 JSONL output

- [ ] Test: format single document as compact single-line JSON
- [ ] Test: format sequence as one JSON object per line (no wrapping array)
- [ ] Test: format streaming (changefeed) output as continuous JSONL
- [ ] Implement: `JSONL(w io.Writer, iter RowIterator) error`

#### 10.5 Non-TTY detection and auto-format

- [ ] Test: isatty(stdout) true -> default to pretty JSON
- [ ] Test: isatty(stdout) false -> default to JSONL
- [ ] Test: explicit --format flag overrides auto-detection
- [ ] Test: NO_COLOR env var disables colored output
- [ ] Implement: `DetectFormat(stdout *os.File, flagFormat string) string`

---

### Phase 11: CLI Commands

Package: `cmd/r-cli`

#### 11.1 Root command and global flags

- [ ] Test: `--host` / `-H` flag defaults to "localhost"
- [ ] Test: `--port` / `-P` flag defaults to 28015
- [ ] Test: `--db` / `-d` flag sets default database
- [ ] Test: `--user` / `-u` flag defaults to "admin"
- [ ] Test: `--password` / `-p` flag (also `RETHINKDB_PASSWORD` env)
- [ ] Test: `--password-file` reads password from file (avoids shell history leaks)
- [ ] Test: `--timeout` / `-t` flag defaults to 30s
- [ ] Test: `--format` / `-f` flag: "json" (default), "jsonl", "raw", "table"
- [ ] Test: `--version` flag
- [ ] Test: `RETHINKDB_HOST` env var overrides default host
- [ ] Test: `RETHINKDB_PORT` env var overrides default port
- [ ] Test: `RETHINKDB_USER` env var overrides default user
- [ ] Test: `RETHINKDB_PASSWORD` env var overrides default password
- [ ] Test: `RETHINKDB_DATABASE` env var overrides default db
- [ ] Test: CLI flag takes precedence over env var
- [ ] Test: `--profile` flag enables query profiling output
- [ ] Test: `--time-format` flag: "native" (default, pseudo-type conversion), "raw" (pass-through)
- [ ] Test: `--binary-format` flag: "native" (default), "raw" (pass-through)
- [ ] Test: `--quiet` suppresses non-data output to stderr
- [ ] Test: `--verbose` shows connection info and query timing to stderr
- [ ] Test: exit code 0 on success
- [ ] Test: exit code 1 on connection error
- [ ] Test: exit code 2 on query/parse error
- [ ] Test: exit code 3 on auth error
- [ ] Test: SIGINT during query -> cancel context, clean exit code 130
- [ ] Test: SIGINT during output streaming -> stop output, clean exit
- [ ] Implement: root command with persistent flags
- [ ] Implement: signal handler (SIGINT/SIGTERM) -> cancel root context

#### 11.2 `query` command (default)

Execute a ReQL query string. This is the primary command.
**Depends on Phase 12 (parser).** Implement after Phase 12 is complete.

- [ ] Test: `r-cli query 'r.db("test").table("users")'` -> executes and prints result
- [ ] Test: `r-cli 'r.db("test").table("users")'` -> query as default command
- [ ] Test: pipe query from stdin: `echo '...' | r-cli query`
- [ ] Test: `--file` / `-F` flag reads query from file
- [ ] Test: `--file` with multiple queries separated by `---` -> execute sequentially, output each
- [ ] Test: `--file` with multiple queries, `--stop-on-error` -> stop on first failure
- [ ] Test: invalid query string -> parse error
- [ ] Test: connection failure -> descriptive error
- [ ] Implement: query command with input modes (arg, stdin, file)

#### 11.3 `run` command

Execute a raw ReQL JSON term directly (pre-serialized).

- [ ] Test: `r-cli run '[15,[[14,["test"]],"users"]]'` -> sends term as-is
- [ ] Test: stdin input
- [ ] Implement: run command

#### 11.4 `db` subcommands

- [ ] Test: `r-cli db list` -> list databases
- [ ] Test: `r-cli db create <name>` -> create database
- [ ] Test: `r-cli db drop <name>` -> drop database (with confirmation)
- [ ] Implement: db command group

#### 11.5 `table` subcommands

- [ ] Test: `r-cli table list` -> list tables in current db
- [ ] Test: `r-cli table create <name>` -> create table
- [ ] Test: `r-cli table drop <name>` -> drop table (with confirmation)
- [ ] Test: `r-cli table info <name>` -> table status/config
- [ ] Implement: table command group

#### 11.6 `status` command

- [ ] Test: `r-cli status` -> shows server info, connection status
- [ ] Implement: status command

#### 11.7 `completion` command

- [ ] Test: `r-cli completion bash` generates valid bash completion script
- [ ] Test: `r-cli completion zsh` generates valid zsh completion script
- [ ] Test: `r-cli completion fish` generates valid fish completion script
- [ ] Implement: cobra built-in completion generation

#### 11.8 `index` subcommands

- [ ] Test: `r-cli index list <table>` -> list secondary indexes
- [ ] Test: `r-cli index create <table> <name>` -> create secondary index
- [ ] Test: `r-cli index create <table> <name> --geo` -> create geo index
- [ ] Test: `r-cli index create <table> <name> --multi` -> create multi index
- [ ] Test: `r-cli index drop <table> <name>` -> drop index
- [ ] Test: `r-cli index rename <table> <old> <new>` -> rename index
- [ ] Test: `r-cli index status <table> [name]` -> show index status
- [ ] Test: `r-cli index wait <table> [name]` -> wait for index readiness
- [ ] Implement: index command group

#### 11.9 `user` subcommands

- [ ] Test: `r-cli user list` -> list users from rethinkdb.users table
- [ ] Test: `r-cli user create <name> --password <pwd>` -> insert user
- [ ] Test: `r-cli user create <name>` (no password flag) -> prompt for password (no echo)
- [ ] Test: `r-cli user delete <name>` -> delete user (with confirmation)
- [ ] Test: `r-cli user set-password <name>` -> prompt and update password (no echo)
- [ ] Implement: user command group (uses `golang.org/x/term` for password prompt)

#### 11.10 `grant` command

- [ ] Test: `r-cli grant <user> --read --write` -> global permissions
- [ ] Test: `r-cli grant <user> --read --db test` -> database permissions
- [ ] Test: `r-cli grant <user> --read --db test --table users` -> table permissions
- [ ] Test: `r-cli grant <user> --read=false` -> revoke permission
- [ ] Implement: grant command with scope flags

#### 11.11 `table reconfigure` and `table rebalance`

- [ ] Test: `r-cli table reconfigure <name> --shards 4 --replicas 2`
- [ ] Test: `r-cli table reconfigure <name> --dry-run` -> preview without applying
- [ ] Test: `r-cli table rebalance <name>`
- [ ] Test: `r-cli table wait <name>`
- [ ] Test: `r-cli table sync <name>`
- [ ] Implement: extend table command group

#### 11.12 `insert` command (bulk)

- [ ] Test: `cat data.jsonl | r-cli insert <db.table>` -> bulk insert from stdin
- [ ] Test: `r-cli insert <db.table> -F data.json` -> bulk insert from JSON file
- [ ] Test: `r-cli insert <db.table> -F data.jsonl --format jsonl` -> JSONL file
- [ ] Test: `--batch-size N` controls documents per insert (default 200)
- [ ] Test: `--conflict replace|update|error` conflict strategy
- [ ] Test: reports total inserted/errors on completion
- [ ] Implement: insert command with streaming stdin reader

---

### Phase 12: Query Language Parser

Package: `internal/reql/parser` (subpackage of Phase 5 term builder)

Parse human-readable ReQL string into term tree.

**After Phase 12 is complete, implement Phase 11.2 (`query` command).**

#### Grammar scope

Supported syntax (strict subset -- no JS lambdas):

- Chained method calls: `r.db("x").table("y").filter({...}).limit(10)`
- `r.row("field")` for field access in predicates (IMPLICIT_VAR)
- Method-style comparisons: `.gt(21)`, `.lt(10)`, `.eq("foo")`, etc.
- Nested `r.*` calls as arguments: `r.desc("name")`, `r.now()`, `r.uuid()`
- Literals: strings (double-quoted), numbers, booleans, null
- Object literals: `{name: "foo", age: 42}` (unquoted keys allowed)
- Array literals: `[1, 2, 3]`

NOT supported (explicit exclusion):

- JavaScript lambdas: `function(row) { ... }`, `row => ...`
- Infix operators: `+`, `-`, `>`, `<` (use method syntax instead)
- Variable declarations, assignments, semicolons

#### 12.1 Lexer

- [ ] Test: tokenize `r.db("test")` -> [IDENT:r, DOT, IDENT:db, LPAREN, STRING:"test", RPAREN]
- [ ] Test: tokenize numbers, bools, null
- [ ] Test: tokenize object literals `{name: "foo", age: 42}`
- [ ] Test: tokenize array literals `[1, 2, 3]`
- [ ] Test: tokenize chained methods `.table("x").filter({...})`
- [ ] Test: tokenize single-quoted strings `'foo'` (in addition to double-quoted)
- [ ] Test: tokenize `r.minval` / `r.maxval` as IDENT (no parens)
- [ ] Implement: lexer producing token stream

#### 12.2 Parser

- [ ] Test: parse `r.db("test")` -> DB("test") term
- [ ] Test: parse `r.db("test").table("users")` -> chained terms
- [ ] Test: parse `.filter({name: "foo"})` -> FILTER with object arg
- [ ] Test: parse `.get("id")` -> GET term
- [ ] Test: parse `.insert({...})` -> INSERT term
- [ ] Test: parse `.orderBy(r.desc("name"))` -> ORDER_BY with DESC
- [ ] Test: parse `.limit(10)` -> LIMIT term
- [ ] Test: parse `r.row("field").gt(21)` -> IMPLICIT_VAR with GT comparison
- [ ] Test: parse nested `r.row` in filter -> auto-wrapped via Phase 5.13
- [ ] Test: parse bracket notation `row("field")("subfield")` -> nested BRACKET terms
- [ ] Test: parse `r.expr([1,2,3])` -> MAKE_ARRAY wrapped
- [ ] Test: parse `r.minval` (no parens) -> MINVAL term
- [ ] Test: parse `r.maxval` (no parens) -> MAXVAL term
- [ ] Test: parse `r.branch(cond, trueVal, falseVal)` -> BRANCH term
- [ ] Test: parse `r.error("msg")` -> ERROR term
- [ ] Test: parse `r.args([...])` -> ARGS term
- [ ] Test: parse all new method names -> correct term types (mapping table test)
- [ ] Test: parse `.eqJoin("field", r.table("other"))` -> EQ_JOIN with table arg
- [ ] Test: parse `.match("^foo")` -> MATCH with string arg
- [ ] Test: parse `r.point(-122.4, 37.7)` -> POINT term
- [ ] Test: parse `r.epochTime(1234567890)` -> EPOCH_TIME term
- [ ] Test: parse `.coerceTo("string")` -> COERCE_TO term
- [ ] Test: parse `.default(0)` -> DEFAULT term
- [ ] Test: syntax error -> descriptive error with position
- [ ] Test: deeply nested expression (depth > 256) -> error (prevent stack overflow)
- [ ] Implement: recursive descent parser producing Term tree (maxDepth=256)

#### 12.3 Fuzz testing

- [ ] Fuzz: lexer does not panic on arbitrary input
- [ ] Fuzz: parser does not panic on arbitrary token sequences
- [ ] Implement: `func FuzzParse(f *testing.F)` with seed corpus from 12.1-12.2 test cases

---

### Phase 13: Integration Testing

Build tag: `//go:build integration`
Run: `make test-integration`
Containers managed by `testcontainers-go` -- no manual Docker setup needed.
Image version pinned to `rethinkdb:2.4.4` for reproducibility.

Package: `internal/integration` (separate package, only compiled with `integration` tag)

**Container fixtures** (test helpers in `internal/integration/testhelper_test.go`):

`startRethinkDB(t) string` -- no password (admin with empty password).
Image `rethinkdb:2.4.4`, cmd `rethinkdb --bind all`, wait `ForListeningPort("28015/tcp")`.
Returns `host:port`. Cleanup via `testcontainers.CleanupContainer(t, ctr)`.

`startRethinkDBWithPassword(t, password string) string` -- same but with
`--initial-password <password>`. For auth tests (13.18).

`startRethinkDBForRestart(t) (testcontainers.Container, string)` -- returns container
handle + addr. For reconnect test (13.20) that needs `ctr.Stop()`/`ctr.Start()`.

**TestMain pattern** for bulk of tests (13.1-13.17, 13.19-13.22):

```go
var testAddr string

func TestMain(m *testing.M) {
    os.Exit(run(m))
}

func run(m *testing.M) int {
    ctx := context.Background()
    ctr, err := testcontainers.Run(ctx, "rethinkdb:2.4.4",
        testcontainers.WithExposedPorts("28015/tcp"),
        testcontainers.WithCmd("rethinkdb", "--bind", "all"),
        testcontainers.WithWaitStrategy(
            wait.ForListeningPort("28015/tcp").WithStartupTimeout(30*time.Second),
        ),
    )
    if err != nil {
        log.Fatalf("start rethinkdb: %v", err)
    }
    defer testcontainers.TerminateContainer(ctr)

    host, _ := ctr.Host(ctx)
    port, _ := ctr.MappedPort(ctx, "28015/tcp")
    testAddr = fmt.Sprintf("%s:%s", host, port.Port())

    return m.Run()
}
```

One shared container per package run (fast). Auth tests (13.18) start their own
password-protected container via `startRethinkDBWithPassword`.

All integration tests share test helpers:
- `setupTestDB(t) string` -- creates a unique database per test run
  (`rcli_test_<unix_ts>`), cleans up in `t.Cleanup()`
- `createTestTable(t, conn, db) string` -- creates a unique table per test
  (`t_<sanitized_test_name>_<rand>`), cleans up in `t.Cleanup()`.
  Unique tables per test prevent DDL/changefeed conflicts when tests run in parallel.

#### 13.1 Connection and handshake

- [ ] Test: connect with default credentials (admin, no password) -> handshake succeeds
- [ ] Test: verify server version is returned in handshake response
- [ ] Test: connect to non-existent host -> dial error with timeout
- [ ] Test: open connection, close it, verify TCP socket released
- [ ] Test: concurrent Dial from multiple goroutines -> all succeed

#### 13.2 Server info

- [ ] Test: SERVER_INFO query returns valid server name and id (non-empty strings)
- [ ] Test: server id is a valid UUID format

#### 13.3 Database operations

- [ ] Test: DB_LIST returns array containing "rethinkdb" and "test" (default system dbs)
- [ ] Test: DB_CREATE creates a new database, DB_LIST now includes it
- [ ] Test: DB_CREATE with existing name -> RUNTIME_ERROR (OP_FAILED)
- [ ] Test: DB_DROP removes database, DB_LIST no longer includes it
- [ ] Test: DB_DROP non-existent database -> RUNTIME_ERROR (OP_FAILED)

#### 13.4 Table operations

- [ ] Test: TABLE_CREATE in test db, TABLE_LIST includes new table
- [ ] Test: TABLE_CREATE with primary_key option -> table uses custom primary key
- [ ] Test: TABLE_CREATE duplicate name -> RUNTIME_ERROR
- [ ] Test: TABLE_DROP removes table, TABLE_LIST no longer includes it
- [ ] Test: TABLE_DROP non-existent table -> RUNTIME_ERROR
- [ ] Test: CONFIG on table -> returns object with id, name, db, primary_key, shards
- [ ] Test: STATUS on table -> returns object with status.all_replicas_ready = true

#### 13.5 Insert

- [ ] Test: insert single document -> response has inserted=1, generated_keys has 1 UUID
- [ ] Test: insert document with explicit id -> no generated_keys, inserted=1
- [ ] Test: insert duplicate id -> RUNTIME_ERROR (OP_FAILED) or conflict response
- [ ] Test: insert with conflict="replace" -> replaced=1
- [ ] Test: insert with conflict="update" -> unchanged=1 or replaced=1
- [ ] Test: bulk insert 100 documents -> inserted=100, generated_keys has 100 UUIDs
- [ ] Test: insert empty object -> inserted=1 (id auto-generated)
- [ ] Test: insert document with nested objects and arrays -> roundtrip preserves structure

#### 13.6 Get and GetAll

- [ ] Test: GET with existing id -> returns the document
- [ ] Test: GET with non-existent id -> returns null
- [ ] Test: GET_ALL with multiple ids -> returns matching documents
- [ ] Test: GET_ALL with secondary index -> returns matching documents
- [ ] Test: GET_ALL with no matches -> empty sequence

#### 13.7 Filter

- [ ] Test: filter by exact field match -> returns matching docs
- [ ] Test: filter with GT comparison -> correct results
- [ ] Test: filter with compound condition (AND) -> correct results
- [ ] Test: filter returns empty sequence when nothing matches
- [ ] Test: filter with nested field access -> correct results

#### 13.8 Update

- [ ] Test: update single document by GET -> replaced=1, verify field changed
- [ ] Test: update all documents in table (no filter) -> replaced=N
- [ ] Test: update with merge (add new field) -> field appears in document
- [ ] Test: update non-existent document via GET -> skipped=1
- [ ] Test: update with return_changes=true -> old_val and new_val present

#### 13.9 Replace

- [ ] Test: replace document by GET -> replaced=1, old fields gone
- [ ] Test: replace must include primary key -> RUNTIME_ERROR if missing

#### 13.10 Delete

- [ ] Test: delete single document by GET -> deleted=1
- [ ] Test: delete with filter -> deleted=N (matching count)
- [ ] Test: delete all from table -> deleted=total
- [ ] Test: delete non-existent document -> deleted=0

#### 13.11 OrderBy, Limit, Skip, Count, Distinct

- [ ] Test: orderBy ascending -> documents in correct order
- [ ] Test: orderBy descending -> reverse order
- [ ] Test: limit(5) on 20 docs -> exactly 5 returned
- [ ] Test: skip(10) on 20 docs -> 10 returned
- [ ] Test: skip(5).limit(5) -> correct slice
- [ ] Test: count on filtered result -> correct number
- [ ] Test: distinct on field with duplicates -> unique values only

#### 13.12 Pluck, Without, Merge, HasFields

- [ ] Test: pluck("name") -> documents with only id and name fields
- [ ] Test: without("password") -> documents without password field
- [ ] Test: merge({new_field: "value"}) -> field added to each document
- [ ] Test: hasFields("email") -> only documents that have email field

#### 13.13 Map, Reduce, Group

- [ ] Test: map extracts single field -> array of values
- [ ] Test: reduce with ADD -> sum of values
- [ ] Test: group by field -> grouped object with arrays
- [ ] Test: group + count -> count per group
- [ ] Test: ungroup -> array of {group, reduction} objects

#### 13.14 Secondary indexes

- [ ] Test: INDEX_CREATE on field -> index created
- [ ] Test: INDEX_LIST -> includes new index
- [ ] Test: INDEX_WAIT -> index ready
- [ ] Test: INDEX_STATUS -> status shows ready=true
- [ ] Test: GetAll with secondary index -> uses index
- [ ] Test: Between with secondary index -> correct range
- [ ] Test: INDEX_DROP removes index
- [ ] Test: INDEX_RENAME renames index

#### 13.15 Streaming and cursors

- [ ] Test: query returning >1 batch (insert 1000+ small docs, read all) -> multiple CONTINUE roundtrips
- [ ] Test: cursor Next() returns documents one by one
- [ ] Test: cursor All() collects everything into slice
- [ ] Test: cursor Close() mid-stream -> sends STOP, no error
- [ ] Test: cursor with context cancel -> stops iteration, no leak
- [ ] Test: two concurrent cursors on same connection -> both complete correctly

#### 13.16 Changefeeds

- [ ] Test: changes() on table -> insert a doc in separate goroutine, cursor receives the change
- [ ] Test: change object has old_val=null, new_val=<doc> for insert
- [ ] Test: update triggers change with old_val and new_val
- [ ] Test: delete triggers change with old_val=<doc>, new_val=null
- [ ] Test: cursor Close() stops changefeed cleanly
- [ ] Test: changes with include_initial=true -> receives existing docs first

#### 13.17 Pseudo-types

- [ ] Test: insert document with r.now() -> returned epoch_time is recent timestamp
- [ ] Test: TIME pseudo-type in response converts to time.Time correctly
- [ ] Test: timezone preserved in roundtrip
- [ ] Test: BINARY pseudo-type -> insert base64 data, read back as []byte, matches original
- [ ] Test: r.uuid() -> returns valid UUID string

#### 13.18 Authentication and users

Uses `startRethinkDBWithPassword(t, "testpass")` -- separate container with
admin password set via `--initial-password`.

All tests in this section use admin connection to manage users,
then open separate connections with user credentials.

RethinkDB user management:
- `r.db("rethinkdb").table("users").insert({id: "alice", password: "secret"})` -- create user
- `r.db("rethinkdb").table("users").get("alice").delete()` -- delete user
- `r.grant("alice", {read: true, write: false, config: false})` -- global permissions
- `r.db("test").grant("alice", {read: true, write: true})` -- per-database permissions
- `r.db("test").table("t").grant("alice", {read: true})` -- per-table permissions

##### 13.18.1 SCRAM-SHA-256 handshake

- [ ] Test: connect as admin with correct password ("testpass") -> handshake succeeds
- [ ] Test: connect with wrong password -> ReqlAuthError (error_code 10-20)
- [ ] Test: connect with non-existent username -> ReqlAuthError
- [ ] Test: create user with password, connect with correct credentials -> handshake succeeds
- [ ] Test: create user, change password, old password fails, new password works
- [ ] Test: user with special characters in password (unicode, quotes, commas) -> handshake succeeds
- [ ] Test: user with empty password -> handshake succeeds (if server allows)

##### 13.18.2 Global permissions

- [ ] Test: create user with no permissions -> any query returns PERMISSION_ERROR
- [ ] Test: grant global read -> user can r.dbList(), r.table().count()
- [ ] Test: global read without write -> insert returns PERMISSION_ERROR
- [ ] Test: grant global read+write -> insert succeeds
- [ ] Test: global write without read -> select returns PERMISSION_ERROR
- [ ] Test: revoke permissions (grant read: false) -> previously working query fails

##### 13.18.3 Database-level permissions

- [ ] Test: grant read on specific db only -> can query tables in that db
- [ ] Test: query table in different db -> PERMISSION_ERROR
- [ ] Test: grant write on specific db -> insert in that db succeeds
- [ ] Test: insert in other db -> PERMISSION_ERROR
- [ ] Test: config permission on db -> can create/drop tables in that db
- [ ] Test: config=false -> TABLE_CREATE returns PERMISSION_ERROR

##### 13.18.4 Table-level permissions

- [ ] Test: grant read on specific table -> can query that table
- [ ] Test: query different table in same db -> PERMISSION_ERROR
- [ ] Test: grant write on specific table -> insert into that table succeeds
- [ ] Test: insert into different table -> PERMISSION_ERROR

##### 13.18.5 Permission inheritance

- [ ] Test: global read + db-level write override -> user can read globally but write only in specific db
- [ ] Test: db-level read=false overrides global read=true -> PERMISSION_ERROR on that db
- [ ] Test: table-level grant overrides db-level -> more specific wins

##### 13.18.6 Cleanup

- [ ] Test: delete user -> connection with that user fails on next query or reconnect
- [ ] Test: t.Cleanup removes all test users (no leftover state between test runs)

#### 13.19 Error handling

- [ ] Test: query non-existent table -> RUNTIME_ERROR with NON_EXISTENCE
- [ ] Test: query non-existent database -> RUNTIME_ERROR with NON_EXISTENCE
- [ ] Test: malformed ReQL JSON -> COMPILE_ERROR
- [ ] Test: type mismatch (e.g. add string + number) -> RUNTIME_ERROR
- [ ] Test: query timeout via context -> context.DeadlineExceeded, no dangling connection

#### 13.20 Connection manager and reconnect

- [ ] Test: 50 concurrent queries through single multiplexed connection -> all succeed, no races
- [ ] Test: kill container mid-query -> ConnManager reconnects after restart (uses `startRethinkDBForRestart`, `ctr.Stop(ctx)` + `ctr.Start(ctx)`)
- [ ] Test: ConnManager Close() with active queries -> all queries return error

#### 13.21 Noreply and NOREPLY_WAIT

- [ ] Test: insert with noreply=true -> no response, document appears in table
- [ ] Test: NOREPLY_WAIT after noreply inserts -> WAIT_COMPLETE, all writes visible

#### 13.22 CLI end-to-end (binary execution)

Tests execute compiled `r-cli` binary via `os/exec`.
Host and port are taken from `testAddr` (dynamically allocated by testcontainers).

- [ ] Test: `r-cli -H <host> -P <port> 'r.dbList()'` -> output contains "test"
- [ ] Test: `r-cli -H <host> -P <port> db list` -> output contains "test"
- [ ] Test: `r-cli -H <host> -P <port> -d <testdb> table list` -> output is valid JSON array
- [ ] Test: `r-cli -H <host> -P <port> status` -> output contains server name
- [ ] Test: `r-cli -H <host> -P <port> -f json 'r.dbList()'` -> valid JSON output
- [ ] Test: `r-cli -H <host> -P <port> -f table 'r.db("<testdb>").table("<t>").limit(5)'` -> ASCII table output
- [ ] Test: `r-cli -H <host> -P <port> -f raw 'r.dbList()'` -> plain text, one item per line
- [ ] Test: `r-cli -H badhost -P <port> 'r.dbList()'` -> exit code 1, stderr contains error
- [ ] Test: `r-cli -H <host> -P <port> run '[59,[]]'` -> same result as r.dbList()
- [ ] Test: `echo 'r.dbList()' | r-cli -H <host> -P <port>` -> works via stdin
- [ ] Test: `r-cli -H <host> -P <port> query -F /tmp/test.reql` -> reads query from file
- [ ] Test: `r-cli -H <host> -P <port> db create <name>` + `r-cli db drop <name>` -> roundtrip
- [ ] Test: `r-cli -H <host> -P <port> table create <name> -d <testdb>` + `table drop` -> roundtrip

#### 13.23 Geospatial integration

- [ ] Test: create geo index, insert points, getNearest returns sorted by distance
- [ ] Test: getIntersecting with polygon -> correct results
- [ ] Test: distance between two points -> correct meters

#### 13.24 String/time operations

- [ ] Test: filter with match regex -> correct results
- [ ] Test: insert with r.now(), read back -> recent timestamp
- [ ] Test: group by .year() -> correct grouping
- [ ] Test: epochTime roundtrip -> correct value

#### 13.25 Joins

- [ ] Test: eqJoin between two tables on secondary index -> correct joined docs
- [ ] Test: eqJoin + zip -> flattened result

#### 13.26 Control flow

- [ ] Test: update with branch -> conditional field update
- [ ] Test: forEach: select from table A, insert into table B
- [ ] Test: default on missing field -> fallback value

#### 13.27 REPL e2e (binary execution)

- [ ] Test: echo query via pipe to r-cli binary (REPL stdin mode)
- [ ] Test: multiple queries via pipe separated by newlines

#### 13.28 User/permission e2e (uses password container)

- [ ] Test: `r-cli user create` + `r-cli user list` -> user appears
- [ ] Test: `r-cli grant <user> --read --db <testdb>` -> user can query that db
- [ ] Test: `r-cli user delete` -> user removed

#### 13.29 Index e2e

- [ ] Test: `r-cli index create` + `r-cli index list` -> index appears
- [ ] Test: `r-cli index wait` -> returns after index ready
- [ ] Test: `r-cli index drop` -> index removed

#### 13.30 Bulk insert e2e

- [ ] Test: generate JSONL file, pipe to `r-cli insert <db.table>` -> documents in table
- [ ] Test: bulk insert with --conflict replace -> existing docs replaced

---

### Phase 14: TLS Support

Package: `internal/conn` (extends Phase 4)

Post-MVP. RethinkDB supports TLS since 2.3. All official drivers and
`rethinkdb dump` support it. Required for managed/cloud instances.

#### 14.1 TLS connection

- [ ] Test: `DialTLS` with valid CA cert -> handshake succeeds
- [ ] Test: `DialTLS` with wrong CA cert -> TLS verification error
- [ ] Test: `DialTLS` with `InsecureSkipVerify` -> connects despite invalid cert
- [ ] Implement: `DialTLS(ctx, addr, tlsConfig)` using `crypto/tls`

#### 14.2 CLI flags

- [ ] Test: `--tls-cert` flag sets CA certificate path
- [ ] Test: `--tls-key` + `--tls-client-cert` for client certificate auth
- [ ] Test: `--insecure-skip-verify` disables cert verification
- [ ] Implement: TLS flags in root command, pass `*tls.Config` to connection

---

### Phase 15: Interactive REPL

Package: `internal/repl`

#### 15.1 Basic REPL loop

- [ ] Test: start REPL, send query, receive output, prompt reappears
- [ ] Test: empty input (just Enter) -> no query executed, new prompt
- [ ] Test: Ctrl+D (EOF) -> clean exit
- [ ] Test: Ctrl+C during input -> cancel current line, new prompt
- [ ] Test: Ctrl+C during query execution -> cancel query (send STOP), new prompt
- [ ] Implement: REPL loop with github.com/chzyer/readline

#### 15.2 History

- [ ] Test: query is saved to history file (~/.r-cli_history)
- [ ] Test: up/down arrows navigate history
- [ ] Test: history persists between sessions
- [ ] Implement: readline history integration

#### 15.3 Multiline input

- [ ] Test: unclosed parenthesis -> continuation prompt, wait for closing
- [ ] Test: unclosed brace -> continuation prompt
- [ ] Test: complete multiline query executes correctly
- [ ] Implement: paren/brace/bracket counting for continuation detection

#### 15.4 Tab completion

- [ ] Test: `r.` + TAB -> list top-level r.* methods
- [ ] Test: `.` + TAB after table -> list chainable methods
- [ ] Test: `r.db("` + TAB -> list database names (query server)
- [ ] Test: `.table("` + TAB -> list table names in current db (query server)
- [ ] Implement: completer with static methods + dynamic db/table names

#### 15.5 REPL-specific commands

- [ ] Test: `.exit` or `.quit` -> exit REPL
- [ ] Test: `.use <db>` -> change default database
- [ ] Test: `.format <fmt>` -> change output format for session
- [ ] Test: `.help` -> show available commands
- [ ] Implement: dot-command dispatcher

#### 15.6 CLI integration

- [ ] Test: `r-cli` (no args, TTY) -> start REPL
- [ ] Test: `r-cli` (no args, not TTY, stdin has data) -> read from stdin
- [ ] Test: `r-cli repl` -> force REPL mode
- [ ] Test: REPL respects --host/--port/--db/--user flags
- [ ] Implement: REPL command in cobra, auto-detect mode

---

## Nice-to-Have Phases (Post-v1)

### Phase 16: Import/Export

- TBD: `r-cli export` and `r-cli import` for data migration

### Phase 17: Dump/Restore

- TBD: `r-cli dump` and `r-cli restore` for database backup/restore

### Phase 18: Monitoring Commands

- TBD: `r-cli stats`, `r-cli jobs`, `r-cli issues`, `r-cli logs`

### Phase 19: Config File with Profiles

- TBD: `~/.r-cli.toml` with named connection profiles

### Nice-to-have ReQL terms (post-v1)

Bitwise: BIT_AND (191), BIT_OR (192), BIT_XOR (193), BIT_NOT (194), BIT_SAL (195), BIT_SAR (196)

Misc: SAMPLE (81), OFFSETS_OF (87), RANGE (173), FOLD (187), OBJECT (143), RANDOM (151)

Advanced: HTTP (153), INFO (79), SET_WRITE_HOOK (189), GET_WRITE_HOOK (190)

---

## Dependencies

- `github.com/spf13/cobra` -- CLI framework (already added)
- `github.com/testcontainers/testcontainers-go` -- integration test containers (add in Phase 13)
- `github.com/chzyer/readline` -- REPL readline support (add in Phase 15)
- `github.com/mattn/go-isatty` -- TTY detection for auto-format (add in Phase 10)
- `golang.org/x/term` -- password prompt without echo (add in Phase 11.9)
- No third-party RethinkDB driver -- implement protocol from scratch per `rethink-driver.md`
- No third-party test libraries -- stdlib `testing` only

## TDD Workflow

1. Pick the next unchecked test case from the current phase
2. Write the test -- it must fail (red)
3. Write minimal code to make it pass (green)
4. Refactor if needed, all tests must still pass
5. Mark the test case as `[x]`
6. Repeat until phase is complete
7. Run `make build` (lint + build) before moving to next phase

## Phase Order Rationale

```
proto (1) -> wire (2) -> scram (3) -> conn (4) -> reql (5) -> response (6) -> cursor (7) -> connmgr (8) -> query (9) -> output (10) -> cli (11*) -> parser (12) -> cli 11.2 -> integration (13) -> tls (14) -> repl (15)
```

Phase 11 is split: 11.1, 11.3-11.6 are implemented before the parser (they use
the programmatic term builder). Phase 11.2 (`query` command) depends on Phase 12
(parser) and is implemented after it. Phase 11 sub-commands (11.7-11.12) depend
on phases 5+9 but can be implemented alongside 11.1-11.6.

Phase 15 (REPL) depends on phases 11+12 (parser + CLI framework).

Each phase depends only on previous phases. Lower layers are fully tested before
higher layers build on them. Integration tests come last and validate the full stack.

Nice-to-have phases (16-19) are post-v1 and tracked in their own section above.
