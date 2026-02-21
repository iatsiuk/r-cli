# RethinkDB Driver Protocol Specification

Reference: https://rethinkdb.com/docs/writing-drivers/
Proto definitions: https://github.com/rethinkdb/rethinkdb/blob/next/src/rdb_protocol/ql2.proto

---

## 1. Connection

Default TCP port: `28015`.

All multi-byte integers transmitted as **little-endian**.

---

## 2. Handshake (V1_0 protocol)

Six-step SCRAM-SHA-256 authentication flow over null-terminated JSON messages.

### Step 1: Client sends magic number

4 bytes, little-endian: `0x34c2bdc3`

```
SEND: c3 bd c2 34
```

### Step 2: Server responds

Null-terminated JSON:

```json
{
    "success": true,
    "min_protocol_version": 0,
    "max_protocol_version": 0,
    "server_version": "2.3.0"
}
```

If `success` is false or response is not valid JSON -- connection error.

### Step 3: Client sends authentication request

Null-terminated JSON with SCRAM-SHA-256 client-first-message:

```json
{
    "protocol_version": 0,
    "authentication_method": "SCRAM-SHA-256",
    "authentication": "n,,n=user,r=<client_nonce>"
}
```

- `protocol_version` -- must be within server's `[min_protocol_version, max_protocol_version]`
- `authentication` -- SCRAM client-first-message per RFC 5802
- `<client_nonce>` -- random base64-encoded string (min 18 bytes recommended)

### Step 4: Server responds with challenge

```json
{
    "success": true,
    "authentication": "r=<combined_nonce>,s=<salt_base64>,i=<iteration_count>"
}
```

- `r` -- concatenation of client nonce + server nonce
- `s` -- base64-encoded salt
- `i` -- PBKDF2 iteration count (typically 4096)

Error response (codes 10-20 inclusive -> `ReqlAuthError`):

```json
{
    "success": false,
    "error": "...",
    "error_code": 12
}
```

### Step 5: Client sends proof

Null-terminated JSON with SCRAM client-final-message:

```json
{
    "authentication": "c=biws,r=<combined_nonce>,p=<client_proof_base64>"
}
```

SCRAM-SHA-256 computation (RFC 7677 + RFC 5802):

```
SaltedPassword  = PBKDF2-SHA256(password, salt, iteration_count)
ClientKey       = HMAC-SHA256(SaltedPassword, "Client Key")
StoredKey       = SHA256(ClientKey)
AuthMessage     = client-first-message-bare + "," +
                  server-first-message + "," +
                  client-final-message-without-proof
ClientSignature = HMAC-SHA256(StoredKey, AuthMessage)
ClientProof     = ClientKey XOR ClientSignature
```

- `c=biws` -- base64 of "n,," (gs2 header, no channel binding)
- `p` -- base64-encoded ClientProof

### Step 6: Server final verification

```json
{
    "success": true,
    "authentication": "v=<server_signature_base64>"
}
```

Driver should verify ServerSignature:

```
ServerKey       = HMAC-SHA256(SaltedPassword, "Server Key")
ServerSignature = HMAC-SHA256(ServerKey, AuthMessage)
```

If `v` does not match computed ServerSignature -- authentication error.

### Optimization

Steps 1 and 3 can be sent together without waiting for step 2 response.
Then read steps 2 and 4 sequentially. Reduces round-trips from 3 to 2.

### Legacy handshake (V0_3 / V0_4)

1. Send protocol version: 4 bytes LE (`V0_3=0x5f75e83e`, `V0_4=0x400c2d20`)
2. Send auth key length: 4 bytes LE (0 if no key)
3. Send auth key: raw ASCII bytes
4. Send protocol type: 4 bytes LE (`JSON=0x7e6970c7`)
5. Read null-terminated response: `"SUCCESS"` or error string

---

## 3. Message Wire Format

### Query message (client -> server)

```
+------------------+------------------+------------------+
| token (8 bytes)  | length (4 bytes) | JSON payload     |
| uint64 LE        | uint32 LE        | UTF-8 bytes      |
+------------------+------------------+------------------+
```

- `token` -- unique per-connection query ID, unsigned 64-bit LE counter
- `length` -- byte length of JSON payload
- JSON payload -- `[QueryType, query, options]`

### Response message (server -> client)

```
+------------------+------------------+------------------+
| token (8 bytes)  | length (4 bytes) | JSON payload     |
| uint64 LE        | uint32 LE        | UTF-8 bytes      |
+------------------+------------------+------------------+
```

- `token` -- echoed query token
- JSON payload -- response object (see section 8)

---

## 4. Query Types

| Value | Name          | Description                                    |
|-------|---------------|------------------------------------------------|
| 1     | START         | Start a new query                              |
| 2     | CONTINUE      | Fetch next batch from partial sequence         |
| 3     | STOP          | Cancel running query/stream/feed               |
| 4     | NOREPLY_WAIT  | Wait for all noreply queries to finish         |
| 5     | SERVER_INFO   | Get server info                                |

### START

```json
[1, <reql_term>, <global_optargs>]
```

- `<reql_term>` -- serialized ReQL query (see section 5)
- `<global_optargs>` -- optional map: `db`, `noreply`, `profile`, `durability`, `read_mode`, etc.

### CONTINUE / STOP

```json
[2]
[3]
```

Sent with the same token as the original START query.

### NOREPLY_WAIT / SERVER_INFO

```json
[4]
[5]
```

---

## 5. ReQL Term Serialization

Every ReQL term serializes as a JSON array:

```
[<term_type>, [<arguments>], {<optional_args>}]
```

- `<term_type>` -- integer from ql2.proto Term.TermType
- `<arguments>` -- ordered list of sub-terms (recursive)
- `<optional_args>` -- key-value map (can be omitted if empty)

### Example: r.db("blog").table("users").filter({name: "Michel"})

```
DB    = 14
TABLE = 15
FILTER = 39

[39, [[15, [[14, ["blog"]], "users"]], {"name": "Michel"}]]
```

Chain decomposition:
1. `r.db("blog")` -> `[14, ["blog"]]`
2. `.table("users")` -> `[15, [[14, ["blog"]], "users"]]`
3. `.filter({...})` -> `[39, [[15, [...]], {"name": "Michel"}]]`

### Literal values

Strings, numbers, booleans, null -- sent as-is (JSON primitives).

### Arrays

Native arrays must be wrapped in MAKE_ARRAY (term 2):

```
[10, 20, 30]  ->  [2, [10, 20, 30]]
```

This prevents ambiguity with term arrays.

### Objects

Object literals pass through as JSON objects directly (within term arguments).

### db option in run()

When `db` is passed as a global optarg, it must be wrapped as a DB term:

```
r.table("users").run({db: "blog"})

Serialized:
[1, [15, ["users"]], {"db": [14, ["blog"]]}]
```

---

## 6. Functions (Lambdas)

### Serialization format

```
[FUNC(69), [[MAKE_ARRAY(2), [param_ids...]], <body_term>]]
```

Parameter IDs are arbitrary integers. Referenced within the body via VAR (term 10):

```
[VAR(10), [<param_id>]]
```

### Example: function(x, y, z) { return r.add(x, y, z) }

```
[69, [[2, [1, 2, 3]], [24, [[10, [1]], [10, [2]], [10, [3]]]]]]
```

Breakdown:
- `69` = FUNC
- `[2, [1, 2, 3]]` = MAKE_ARRAY with param IDs 1, 2, 3
- `24` = ADD
- `[10, [1]]` = VAR referencing param 1

### IMPLICIT_VAR (r.row)

Term 13. Shorthand for a single-argument function.

When a method argument contains IMPLICIT_VAR, the driver must wrap it:

```
[69, [[2, [1]], <argument_with_implicit_var_replaced_by_var_1>]]
```

IMPLICIT_VAR in nested functions is ambiguous -- throw an error or defer to server.

---

## 7. FUNCALL (r.do)

**Important:** argument order differs between API and wire format.

API: `r.do(arg1, arg2, function)`
Wire: `[FUNCALL(64), [function], arg1, arg2]`

Function goes **first** on the wire, arguments follow.

### Example: r.do(10, 20, function(x, y) { return r.add(x, y) })

```
[64, [69, [[2, [1, 2]], [24, [[10, [1]], [10, [2]]]]]], 10, 20]
```

---

## 8. Response Format

### Response object fields

| Field | Type          | Description                                      |
|-------|---------------|--------------------------------------------------|
| t     | int           | ResponseType                                     |
| r     | array         | Result data                                      |
| e     | int           | ErrorType (error responses only)                 |
| b     | array         | Backtrace frames (error responses only)          |
| p     | object        | Profile data (if `profile: true` in query)       |
| n     | array of int  | ResponseNote values (changefeeds)                |

### Response Types

| Value | Name              | Description                                  |
|-------|-------------------|----------------------------------------------|
| 1     | SUCCESS_ATOM      | Single value in `r[0]`                       |
| 2     | SUCCESS_SEQUENCE  | Complete sequence in `r`, or final batch     |
| 3     | SUCCESS_PARTIAL   | Partial sequence, send CONTINUE for more     |
| 4     | WAIT_COMPLETE     | NOREPLY_WAIT done, `r` is empty              |
| 5     | SERVER_INFO       | Server info object in `r[0]`                 |
| 16    | CLIENT_ERROR      | Driver/client bug, message in `r[0]`         |
| 17    | COMPILE_ERROR     | ReQL parse/type error, message in `r[0]`     |
| 18    | RUNTIME_ERROR     | Execution error, message in `r[0]`           |

### Error Types (field `e`)

| Value   | Name             | Description                          |
|---------|------------------|--------------------------------------|
| 1000000 | INTERNAL         | Internal server error                |
| 2000000 | RESOURCE_LIMIT   | Resource limit exceeded              |
| 3000000 | QUERY_LOGIC      | Query logic error                    |
| 3100000 | NON_EXISTENCE    | Table/database does not exist        |
| 4100000 | OP_FAILED        | Operation failed                     |
| 4200000 | OP_INDETERMINATE | Operation result unknown             |
| 5000000 | USER             | User-generated error (r.error())     |
| 6000000 | PERMISSION_ERROR | Insufficient permissions             |

### Response Notes (changefeeds)

| Value | Name                 | Description                          |
|-------|----------------------|--------------------------------------|
| 1     | SEQUENCE_FEED        | Standard changefeed                  |
| 2     | ATOM_FEED            | Point changefeed (single document)   |
| 3     | ORDER_BY_LIMIT_FEED  | order_by().limit() changefeed        |
| 4     | UNIONED_FEED         | Union of incompatible feeds          |
| 5     | INCLUDES_STATES      | Changefeed includes state objects    |

---

## 9. Streaming and Cursors

Queries returning sequences may produce:
- Single `SUCCESS_SEQUENCE` (small result sets)
- One or more `SUCCESS_PARTIAL` followed by a final `SUCCESS_SEQUENCE`

### Fetching next batch

Send CONTINUE with the same token:

```
Token:   <original_token>
Length:  03 00 00 00
Message: [2]
```

Driver can send CONTINUE immediately after receiving SUCCESS_PARTIAL
(no need to wait for consumer to read data).

### Stopping a stream

Send STOP with the same token:

```
Token:   <original_token>
Length:  03 00 00 00
Message: [3]
```

### Changefeeds

Never return SUCCESS_SEQUENCE -- they produce SUCCESS_PARTIAL indefinitely.
Must be explicitly stopped with STOP or by closing the connection.

---

## 10. Pseudo Types

### TIME

```json
{
    "$reql_type$": "TIME",
    "epoch_time": 1376436985.298,
    "timezone": "+00:00"
}
```

- `epoch_time` -- seconds since Unix epoch (float, millisecond precision)
- `timezone` -- format `[+-]HH:MM`

Driver should convert to/from native datetime type.

### BINARY

```json
{
    "$reql_type$": "BINARY",
    "data": "<base64_encoded_string>"
}
```

As a ReQL term (wrapping another term): `[BINARY(155), [argument]]`
As a literal (native bytes): use the pseudo-type JSON above.

### GEOMETRY (GeoJSON)

Geospatial types use standard GeoJSON format with `$reql_type$: "GEOMETRY"`.

---

## 11. Connection Pool Management

Since V0_4 (RethinkDB 2.0+), queries execute in **parallel** on a single connection.
No read-after-write guarantee on the same connection.

**Pool release rule:** do NOT release a connection back to pool while it has
active streams (SUCCESS_PARTIAL pending). Release only after receiving
SUCCESS_SEQUENCE, SUCCESS_ATOM, WAIT_COMPLETE, SERVER_INFO, or an error.

Each connection maintains its own token counter (uint64, starting from 0).
Multiple queries can be in-flight simultaneously on a single connection.

---

## 12. Global Optional Arguments (run options)

Passed as the third element of START query: `[1, <term>, <optargs>]`

| Option       | Type    | Description                                      |
|--------------|---------|--------------------------------------------------|
| db           | DB term | Default database (wrapped as `[14, ["name"]]`)   |
| noreply      | bool    | Don't wait for response                          |
| profile      | bool    | Include profiling data in response               |
| durability   | string  | "hard" or "soft"                                 |
| read_mode    | string  | "single", "majority", "outdated"                 |
| array_limit  | int     | Max array size (default 100000)                  |

---

## 13. Complete Term Type Reference

### Data construction

| Value | Name       | Description                              |
|-------|------------|------------------------------------------|
| 1     | DATUM      | Raw datum value                          |
| 2     | MAKE_ARRAY | Construct array from arguments           |
| 3     | MAKE_OBJ   | Construct object from key-value pairs    |

### Variables and functions

| Value | Name         | Description                              |
|-------|--------------|------------------------------------------|
| 10    | VAR          | Variable reference                       |
| 11    | JAVASCRIPT   | Execute JavaScript on server             |
| 12    | ERROR        | Throw an error                           |
| 13    | IMPLICIT_VAR | r.row implicit variable                  |
| 64    | FUNCALL      | Call a function (r.do)                   |
| 69    | FUNC         | Define anonymous function                |
| 153   | HTTP         | HTTP request                             |
| 169   | UUID         | Generate UUID                            |

### Database and table operations

| Value | Name          | Description                              |
|-------|---------------|------------------------------------------|
| 14    | DB            | Reference a database                     |
| 15    | TABLE         | Reference a table                        |
| 16    | GET           | Get document by primary key              |
| 57    | DB_CREATE     | Create database                          |
| 58    | DB_DROP       | Drop database                            |
| 59    | DB_LIST       | List databases                           |
| 60    | TABLE_CREATE  | Create table                             |
| 61    | TABLE_DROP    | Drop table                               |
| 62    | TABLE_LIST    | List tables                              |
| 78    | GET_ALL       | Get documents by primary key(s)          |
| 138   | SYNC          | Flush table to disk                      |
| 174   | CONFIG        | Table/database config                    |
| 175   | STATUS        | Table status                             |
| 176   | RECONFIGURE  | Reconfigure table sharding/replication   |
| 177   | WAIT          | Wait for table readiness                 |
| 179   | REBALANCE    | Rebalance table shards                   |

### Write operations

| Value | Name    | Description                                  |
|-------|---------|----------------------------------------------|
| 53    | UPDATE  | Update documents                             |
| 54    | DELETE  | Delete documents                             |
| 55    | REPLACE | Replace documents                            |
| 56    | INSERT  | Insert documents                             |

### Index operations

| Value | Name          | Description                              |
|-------|---------------|------------------------------------------|
| 75    | INDEX_CREATE  | Create secondary index                   |
| 76    | INDEX_DROP    | Drop secondary index                     |
| 77    | INDEX_LIST    | List secondary indexes                   |
| 139   | INDEX_STATUS  | Get index status                         |
| 140   | INDEX_WAIT    | Wait for index readiness                 |
| 156   | INDEX_RENAME  | Rename secondary index                   |

### Write hooks

| Value | Name           | Description                              |
|-------|----------------|------------------------------------------|
| 189   | SET_WRITE_HOOK | Set write hook function                  |
| 190   | GET_WRITE_HOOK | Get write hook function                  |

### Comparison

| Value | Name | Description    |
|-------|------|----------------|
| 17    | EQ   | Equal          |
| 18    | NE   | Not equal      |
| 19    | LT   | Less than      |
| 20    | LE   | Less or equal  |
| 21    | GT   | Greater than   |
| 22    | GE   | Greater or equal |
| 23    | NOT  | Logical not    |

### Arithmetic

| Value | Name  | Description        |
|-------|-------|--------------------|
| 24    | ADD   | Add / concatenate  |
| 25    | SUB   | Subtract           |
| 26    | MUL   | Multiply           |
| 27    | DIV   | Divide             |
| 28    | MOD   | Modulo             |
| 183   | FLOOR | Floor              |
| 184   | CEIL  | Ceiling            |
| 185   | ROUND | Round              |

### Logic

| Value | Name    | Description                              |
|-------|---------|------------------------------------------|
| 65    | BRANCH  | if-then-else                             |
| 66    | OR      | Logical or                               |
| 67    | AND     | Logical and                              |
| 68    | FOR_EACH| Apply write query to each element        |
| 92    | DEFAULT | Provide default for missing/null values  |

### Sequence transformations

| Value | Name        | Description                              |
|-------|-------------|------------------------------------------|
| 37    | REDUCE      | Reduce sequence to single value          |
| 38    | MAP         | Apply function to each element           |
| 39    | FILTER      | Filter elements by predicate             |
| 40    | CONCAT_MAP  | Map then flatten                         |
| 41    | ORDER_BY    | Sort sequence                            |
| 42    | DISTINCT    | Remove duplicates                        |
| 43    | COUNT       | Count elements                           |
| 44    | UNION       | Merge sequences                          |
| 45    | NTH         | Get nth element                          |
| 70    | SKIP        | Skip first n elements                    |
| 71    | LIMIT       | Take first n elements                    |
| 72    | ZIP         | Zip joined results                       |
| 81    | SAMPLE      | Random sample                            |
| 86    | IS_EMPTY    | Check if sequence is empty               |
| 87    | OFFSETS_OF  | Indexes of matching elements             |
| 93    | CONTAINS    | Check if sequence contains value         |
| 170   | BRACKET     | Get field or nth element                 |
| 173   | RANGE       | Generate number range                    |
| 187   | FOLD        | Fold with emit                           |

### Object operations

| Value | Name        | Description                              |
|-------|-------------|------------------------------------------|
| 31    | GET_FIELD   | Get single field value                   |
| 32    | HAS_FIELDS  | Check for field existence                |
| 33    | PLUCK       | Select specific fields                   |
| 34    | WITHOUT     | Remove specific fields                   |
| 35    | MERGE       | Merge objects                            |
| 94    | KEYS        | Get object keys                          |
| 96    | WITH_FIELDS | Filter docs that have all specified fields |
| 137   | LITERAL     | Replace nested value in merge            |
| 143   | OBJECT      | Construct object from pairs              |
| 186   | VALUES      | Get object values                        |

### Array operations

| Value | Name             | Description                          |
|-------|------------------|--------------------------------------|
| 29    | APPEND           | Append to array                      |
| 30    | SLICE            | Array slice                          |
| 80    | PREPEND          | Prepend to array                     |
| 82    | INSERT_AT        | Insert at index                      |
| 83    | DELETE_AT        | Delete at index                      |
| 84    | CHANGE_AT        | Change value at index                |
| 85    | SPLICE_AT        | Splice array at index                |
| 88    | SET_INSERT       | Insert into set                      |
| 89    | SET_INTERSECTION | Set intersection                     |
| 90    | SET_UNION        | Set union                            |
| 91    | SET_DIFFERENCE   | Set difference                       |
| 95    | DIFFERENCE       | Array difference                     |

### Aggregation

| Value | Name    | Description                              |
|-------|---------|------------------------------------------|
| 144   | GROUP   | Group by field or function               |
| 145   | SUM     | Sum values                               |
| 146   | AVG     | Average values                           |
| 147   | MIN     | Minimum value                            |
| 148   | MAX     | Maximum value                            |
| 150   | UNGROUP | Ungroup grouped data                     |

### Join operations

| Value | Name       | Description                              |
|-------|------------|------------------------------------------|
| 48    | INNER_JOIN | Inner join                               |
| 49    | OUTER_JOIN | Outer join                               |
| 50    | EQ_JOIN    | Equality join on index                   |
| 182   | BETWEEN    | Filter by primary key range              |

### Type operations

| Value | Name      | Description                              |
|-------|-----------|------------------------------------------|
| 51    | COERCE_TO | Convert between types                    |
| 52    | TYPE_OF   | Get type name string                     |

### String operations

| Value | Name           | Description                          |
|-------|----------------|--------------------------------------|
| 97    | MATCH          | Regex match                          |
| 98    | JSON           | Parse JSON string                    |
| 141   | UPCASE         | Uppercase                            |
| 142   | DOWNCASE       | Lowercase                            |
| 149   | SPLIT          | Split string                         |
| 172   | TO_JSON_STRING | Serialize to JSON string             |

### Time operations

| Value | Name         | Description                              |
|-------|--------------|------------------------------------------|
| 99    | ISO8601      | Parse ISO 8601 string                    |
| 100   | TO_ISO8601   | Convert to ISO 8601 string              |
| 101   | EPOCH_TIME   | Create time from epoch seconds           |
| 102   | TO_EPOCH_TIME| Convert to epoch seconds                 |
| 103   | NOW          | Current server time                      |
| 104   | IN_TIMEZONE  | Convert to timezone                      |
| 105   | DURING       | Check if time is in interval             |
| 106   | DATE         | Extract date portion                     |
| 126   | TIME_OF_DAY  | Seconds since midnight                   |
| 127   | TIMEZONE     | Get timezone string                      |
| 128   | YEAR         | Extract year                             |
| 129   | MONTH        | Extract month                            |
| 130   | DAY          | Extract day                              |
| 131   | DAY_OF_WEEK  | Day of week (1=Monday)                   |
| 132   | DAY_OF_YEAR  | Day of year                              |
| 133   | HOURS        | Extract hours                            |
| 134   | MINUTES      | Extract minutes                          |
| 135   | SECONDS      | Extract seconds                          |
| 136   | TIME         | Construct time from components           |

### Time constants

| Value | Name      | Value | Name      |
|-------|-----------|-------|-----------|
| 107   | MONDAY    | 114   | JANUARY   |
| 108   | TUESDAY   | 115   | FEBRUARY  |
| 109   | WEDNESDAY | 116   | MARCH     |
| 110   | THURSDAY  | 117   | APRIL     |
| 111   | FRIDAY    | 118   | MAY       |
| 112   | SATURDAY  | 119   | JUNE      |
| 113   | SUNDAY    | 120   | JULY      |
|       |           | 121   | AUGUST    |
|       |           | 122   | SEPTEMBER |
|       |           | 123   | OCTOBER   |
|       |           | 124   | NOVEMBER  |
|       |           | 125   | DECEMBER  |

### Ordering

| Value | Name | Description    |
|-------|------|----------------|
| 73    | ASC  | Ascending      |
| 74    | DESC | Descending     |

### Geospatial

| Value | Name             | Description                          |
|-------|------------------|--------------------------------------|
| 157   | GEOJSON          | Convert GeoJSON to geometry          |
| 158   | TO_GEOJSON       | Convert geometry to GeoJSON          |
| 159   | POINT            | Create point                         |
| 160   | LINE             | Create line                          |
| 161   | POLYGON          | Create polygon                       |
| 162   | DISTANCE         | Distance between geometries          |
| 163   | INTERSECTS       | Check intersection                   |
| 164   | INCLUDES         | Check if geometry includes point     |
| 165   | CIRCLE           | Create circle                        |
| 166   | GET_INTERSECTING | Get docs intersecting geometry       |
| 167   | FILL             | Convert line to polygon              |
| 168   | GET_NEAREST      | Get docs nearest to point            |
| 171   | POLYGON_SUB      | Subtract polygon from polygon        |

### Bitwise operations

| Value | Name    | Description              |
|-------|---------|--------------------------|
| 191   | BIT_AND | Bitwise AND              |
| 192   | BIT_OR  | Bitwise OR               |
| 193   | BIT_XOR | Bitwise XOR              |
| 194   | BIT_NOT | Bitwise NOT              |
| 195   | BIT_SAL | Shift arithmetic left    |
| 196   | BIT_SAR | Shift arithmetic right   |

### Miscellaneous

| Value | Name    | Description                              |
|-------|---------|------------------------------------------|
| 79    | INFO    | Get info about a term                    |
| 151   | RANDOM  | Generate random number                   |
| 152   | CHANGES | Changefeed                               |
| 154   | ARGS    | Splice array as arguments                |
| 155   | BINARY  | Binary data                              |
| 180   | MINVAL  | Minimum value sentinel                   |
| 181   | MAXVAL  | Maximum value sentinel                   |
| 188   | GRANT   | Set permissions                          |

---

## 14. Protocol Version Constants

| Constant | Value      | Description                              |
|----------|------------|------------------------------------------|
| V0_1     | 0x3f61ba36 | Initial version                          |
| V0_2     | 0x723081e1 | Auth key in handshake                    |
| V0_3     | 0x5f75e83e | Auth key + protocol in handshake         |
| V0_4     | 0x400c2d20 | Parallel query execution                 |
| V1_0     | 0x34c2bdc3 | Users and permissions (SCRAM-SHA-256)    |

---

## 15. Datum Types

| Value | Name    | Description                              |
|-------|---------|------------------------------------------|
| 1     | R_NULL  | null                                     |
| 2     | R_BOOL  | boolean                                  |
| 3     | R_NUM   | double-precision float                   |
| 4     | R_STR   | string                                   |
| 5     | R_ARRAY | array                                    |
| 6     | R_OBJECT| object                                   |
| 7     | R_JSON  | pre-encoded JSON string                  |

---

## 16. Backtrace Format

Error responses include `b` field -- array of Frame objects.

Frame types:
- `POS (1)` -- positional argument index
- `OPT (2)` -- optional argument name

Used to pinpoint the exact sub-term that caused the error.

---

## 17. Wire Format Examples

### Simple query: r.expr("foo")

```
Client sends:
  Token:   00 00 00 00 00 00 00 01
  Length:  0c 00 00 00
  JSON:    [1,"foo",{}]

Server responds:
  Token:   00 00 00 00 00 00 00 01
  Length:  13 00 00 00
  JSON:    {"t":1,"r":["foo"]}
```

### Table query: r.db("test").table("users")

```
[1, [15, [[14, ["test"]], "users"]], {}]
```

### Insert: r.db("test").table("users").insert({name: "Alice"})

```
[1, [56, [[15, [[14, ["test"]], "users"]], {"name": "Alice"}]], {}]
```

### Filter with function: r.table("users").filter(r.row("age").gt(21))

Using IMPLICIT_VAR, auto-wrapped:

```
[1, [39, [[15, ["users"]], [69, [[2, [1]], [21, [[31, [[10, [1]], "age"]], 21]]]]]], {}]
```

### CONTINUE (fetch next batch)

```
Token:   <same as original query>
Length:  03 00 00 00
JSON:    [2]
```

### STOP (cancel stream)

```
Token:   <same as original query>
Length:  03 00 00 00
JSON:    [3]
```

---

## 18. Driver Implementation Checklist

1. **TCP connection** -- connect to port 28015
2. **Handshake** -- V1_0 SCRAM-SHA-256 authentication
3. **Token counter** -- uint64 per connection, incrementing
4. **Message framing** -- 8-byte token + 4-byte length + JSON
5. **ReQL serialization** -- recursive term encoding
6. **Array wrapping** -- MAKE_ARRAY for literal arrays
7. **Function serialization** -- FUNC + MAKE_ARRAY params + body
8. **IMPLICIT_VAR detection** -- auto-wrap in FUNC
9. **FUNCALL argument reordering** -- function first on wire
10. **Response parsing** -- dispatch on ResponseType
11. **Cursor/streaming** -- CONTINUE/STOP for partial results
12. **Pseudo-type conversion** -- TIME, BINARY, GEOMETRY
13. **Error handling** -- CLIENT_ERROR, COMPILE_ERROR, RUNTIME_ERROR with ErrorType
14. **Connection pooling** -- don't release connections with active streams
15. **Changefeed support** -- ResponseNote handling, infinite streams
16. **noreply mode** -- fire-and-forget queries
17. **Profiling** -- pass `profile: true`, parse `p` field in response
