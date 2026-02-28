# Implement Missing ReQL Operations

## Overview

Add 13 missing ReQL operations to r-cli so they work end-to-end:
1. Parser (`internal/reql/parser`) recognizes the syntax
2. REPL uses the same parser -- works automatically
3. Full test coverage: unit tests (reql + parser) and integration tests (live RethinkDB)

Operations to implement:
- **Geo constructors (parser only):** `r.line`, `r.polygon`, `r.circle` -- reql functions already exist
- **Time/binary (parser only):** `r.time`, `r.binary` -- reql functions already exist
- **Top-level builders (reql + parser):** `r.object`, `r.range`, `r.random`
- **Chain methods (reql + parser):** `.info()`, `.offsetsOf()`, `.fold()`
- **Bitwise chain methods (reql + parser):** `.bitAnd()`, `.bitOr()`, `.bitXor()`, `.bitNot()`, `.bitSal()`, `.bitSar()`
- **Control flow (parser only, complex):** `r.do` / `any.do()` -- `reql.Do()` already exists

All term type constants already exist in `internal/proto/term.go`.

## Context

- Files involved: `internal/reql/term.go`, `internal/reql/parser/parser.go`, `internal/reql/term_test.go`, `internal/reql/parser/parser_test.go`, `internal/integration/`, `docs/rethinkdb-js-api.md`
- Patterns: well-established -- table-driven unit tests, `rBuilders`/`chainBuilders` maps, integration helpers (`newExecutor`, `setupTestDB`, `seedTable`)
- Existing reql functions: `Time()`, `Binary()`, `Do()`, `Circle()`, `Line()`, `Polygon()` -- only need parser + tests
- Missing reql functions: `Range()`, `Object()`, `Random()`, `Fold()`, `OffsetsOf()`, `Info()`, `BitAnd()`..`BitSar()`

## Development Approach

- **Testing approach**: TDD -- write tests first, then implement
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**
- Run `make build` after each change (includes linter)
- Run `go test ./internal/reql/... ./internal/reql/parser/... -race -count=1` for unit tests
- Run `make test-integration` for integration tests

## Testing Strategy

- **Unit tests (reql):** table-driven in `term_test.go`, verify JSON serialization matches wire format
- **Unit tests (parser):** table-driven in `parser_test.go`, verify `Parse()` produces correct `reql.Term`
- **Integration tests:** against live RethinkDB 2.4.4 in Docker via testcontainers-go, verify actual query execution

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Implementation Steps

### Task 1: Geo constructors -- r.line, r.polygon, r.circle (parser only)

reql functions `Line()`, `Polygon()`, `Circle()` already exist. Need parser support + tests.

- [ ] write parser tests for `r.line(point1, point2, ...)` with 2+ points (parser_test.go)
- [ ] write parser tests for `r.polygon(point1, point2, point3, ...)` with 3+ points (parser_test.go)
- [ ] write parser tests for `r.circle(center, radius)` and `r.circle(center, radius, {opts})` (parser_test.go)
- [ ] write parser error tests: r.line with <2 points, r.polygon with <3 points
- [ ] implement `parseRLine` -- parse variadic point args, call `reql.Line()` (parser.go)
- [ ] implement `parseRPolygon` -- parse variadic point args, call `reql.Polygon()` (parser.go)
- [ ] implement `parseRCircle` -- parse center + radius + optional opts, call `reql.Circle()` (parser.go)
- [ ] register `"line"`, `"polygon"`, `"circle"` in `buildRBuilders()` (parser.go)
- [ ] run unit tests -- must pass
- [ ] write integration test: insert geo doc with `r.line()`, verify with `toGeoJSON()` (integration)
- [ ] write integration test: insert geo doc with `r.polygon()`, verify with `toGeoJSON()` (integration)
- [ ] write integration test: create `r.circle()`, verify it returns polygon geometry (integration)
- [ ] run `make build` + `make test-integration` -- must pass

### Task 2: r.time and r.binary (parser only)

reql functions `Time()` and `Binary()` already exist. Need parser support + tests.

`r.time` has two forms:
- `r.time(year, month, day, timezone)` -- 4 args
- `r.time(year, month, day, hour, minute, second, timezone)` -- 7 args

`r.binary` takes a single base64-encoded string argument in the parser context (CLI can't pass raw binary, so base64 string is the practical input).

- [ ] write parser tests for `r.time(2024, 1, 15, "+00:00")` -- 4-arg form (parser_test.go)
- [ ] write parser tests for `r.time(2024, 1, 15, 10, 30, 0, "+00:00")` -- 7-arg form (parser_test.go)
- [ ] write parser error test: r.time with wrong arg count (not 4 or 7)
- [ ] write parser test for `r.binary(data)` (parser_test.go)
- [ ] implement `parseRTime` -- parse 4 or 7 args, extend `reql.Time()` if needed for 7-arg form (parser.go, term.go)
- [ ] implement `parseRBinary` -- parse single arg, call `reql.Binary()` (parser.go)
- [ ] register `"time"`, `"binary"` in `buildRBuilders()` (parser.go)
- [ ] run unit tests -- must pass
- [ ] write integration test: `r.time(2024, 1, 15, "+00:00")` returns valid time object (integration)
- [ ] write integration test: `r.time(2024, 1, 15, 10, 30, 0, "+00:00")` includes hours/minutes (integration)
- [ ] write integration test: `r.binary()` round-trip with base64 data (integration)
- [ ] run `make build` + `make test-integration` -- must pass

### Task 3: New top-level builders -- r.object, r.range, r.random

Need new reql functions + parser support + tests.

**r.object(key, val, ...):** creates object from key-value pairs, wire format `[143, [key, val, ...]]`
**r.range()** / **r.range(end)** / **r.range(start, end):** generates integer sequence, wire format `[173, []]` / `[173, [end]]` / `[173, [start, end]]`
**r.random()** / **r.random(n)** / **r.random(n, m)** with optional `{float: true}`: random number, wire format `[151, [...], opts?]`

- [ ] write reql unit tests for `Object()`, `Range()`, `Random()` -- all arg variants + JSON wire format (term_test.go)
- [ ] implement `func Object(pairs ...interface{}) Term` in term.go -- validate even arg count
- [ ] implement `func Range(args ...interface{}) Term` in term.go -- 0, 1, or 2 int args
- [ ] implement `func Random(args ...interface{}) Term` in term.go -- 0, 1, or 2 args + optional OptArgs
- [ ] run reql unit tests -- must pass
- [ ] write parser tests for `r.object("a", 1, "b", 2)` (parser_test.go)
- [ ] write parser tests for `r.range()`, `r.range(10)`, `r.range(1, 10)` (parser_test.go)
- [ ] write parser tests for `r.random()`, `r.random(100)`, `r.random(1, 10, {float: true})` (parser_test.go)
- [ ] write parser error tests: r.object with odd arg count, r.range with >2 args
- [ ] implement `parseRObject`, `parseRRange`, `parseRRandom` (parser.go)
- [ ] register `"object"`, `"range"`, `"random"` in `buildRBuilders()` (parser.go)
- [ ] run parser unit tests -- must pass
- [ ] write integration test: `r.object("a", 1, "b", 2)` returns `{"a": 1, "b": 2}` (integration)
- [ ] write integration test: `r.range(5)` returns `[0, 1, 2, 3, 4]` (integration)
- [ ] write integration test: `r.range(2, 5)` returns `[2, 3, 4]` (integration)
- [ ] write integration test: `r.random()` returns float in [0, 1), `r.random(10)` returns int in [0, 10) (integration)
- [ ] run `make build` + `make test-integration` -- must pass

### Task 4: Chain methods -- .info() and .offsetsOf()

Need new reql methods + parser support + tests.

**.info():** returns metadata about a value, wire format `[79, [term]]`
**.offsetsOf(datum)** / **.offsetsOf(pred_fn):** find indexes of matching elements, wire format `[87, [seq, datum_or_fn]]`

- [ ] write reql unit tests for `Info()` and `OffsetsOf()` -- JSON wire format (term_test.go)
- [ ] implement `func (t Term) Info() Term` in term.go
- [ ] implement `func (t Term) OffsetsOf(predicate interface{}) Term` in term.go
- [ ] run reql unit tests -- must pass
- [ ] write parser tests for `.info()`, `.offsetsOf("value")`, `.offsetsOf(lambda)` (parser_test.go)
- [ ] implement parser: register `"info"` as `noArgChain`, `"offsetsOf"` as `oneArgChain` (parser.go)
- [ ] run parser unit tests -- must pass
- [ ] write integration test: `r.db("test").table("t").info()` returns table metadata (integration)
- [ ] write integration test: `r.expr(["a","b","c","b"]).offsetsOf("b")` returns `[1, 3]` (integration)
- [ ] run `make build` + `make test-integration` -- must pass

### Task 5: Chain method -- .fold()

Need new reql method + parser support + tests.

**.fold(base, fn)** / **.fold(base, fn, {emit: fn, finalEmit: fn}):** accumulator over sequence, wire format `[187, [seq, base, fn], opts?]`

- [ ] write reql unit tests for `Fold()` -- basic and with opts (term_test.go)
- [ ] implement `func (t Term) Fold(base, fn Term, opts ...OptArgs) Term` in term.go
- [ ] run reql unit tests -- must pass
- [ ] write parser tests for `.fold(0, (acc, x) => acc.add(x))` (parser_test.go)
- [ ] implement parser: custom `chainFold` function handling base + lambda + optional opts (parser.go)
- [ ] register `"fold"` in chain builders (parser.go)
- [ ] run parser unit tests -- must pass
- [ ] write integration test: `r.expr([1,2,3]).fold(0, (acc, x) => acc.add(x))` returns 6 (integration)
- [ ] write integration test: fold on table field for running sum (integration)
- [ ] run `make build` + `make test-integration` -- must pass

### Task 6: Bitwise operations (6 methods)

Need 6 new reql methods + parser support + tests.

`.bitAnd(n)`, `.bitOr(n)`, `.bitXor(n)` -- single int arg, wire format `[191..193, [val, n]]`
`.bitNot()` -- no args, wire format `[194, [val]]`
`.bitSal(n)`, `.bitSar(n)` -- single int arg, wire format `[195..196, [val, n]]`

- [ ] write reql unit tests for all 6 bitwise methods -- JSON wire format (term_test.go)
- [ ] implement `BitAnd(n)`, `BitOr(n)`, `BitXor(n)`, `BitNot()`, `BitSal(n)`, `BitSar(n)` on Term (term.go)
- [ ] run reql unit tests -- must pass
- [ ] write parser tests for `r.expr(5).bitAnd(3)`, `.bitOr()`, `.bitXor()`, `.bitNot()`, `.bitSal()`, `.bitSar()` (parser_test.go)
- [ ] register all 6 in chain builders: `bitNot` as `noArgChain`, rest as `oneArgChain` (parser.go)
- [ ] run parser unit tests -- must pass
- [ ] write integration tests: `r.expr(5).bitAnd(3)` = 1, `r.expr(5).bitOr(3)` = 7, etc. (integration)
- [ ] write integration test: `r.expr(7).bitNot()` returns -8, shift operations (integration)
- [ ] run `make build` + `make test-integration` -- must pass

### Task 7: r.do / any.do() (most complex)

`reql.Do()` already exists (top-level). Need parser support for both forms + chain method for `any.do(fn)`.

**r.do(args..., fn):** top-level form, wire format `[64, [fn, args...]]` (fn first in wire)
**any.do(fn):** chain form, equivalent to `r.do(any, fn)`, wire format `[64, [fn, any]]`

- [ ] write reql unit test for chain method `Term.Do(fn)` (term_test.go)
- [ ] implement `func (t Term) Do(fn Term) Term` chain method in term.go
- [ ] run reql unit tests -- must pass
- [ ] write parser tests for `r.do(r.table("t"), (t) => t.count())` -- top-level (parser_test.go)
- [ ] write parser tests for `r.table("t").do((t) => t.count())` -- chain form (parser_test.go)
- [ ] write parser tests for `r.do(r.expr(1), r.expr(2), (a, b) => a.add(b))` -- multi-arg (parser_test.go)
- [ ] implement `parseRDo` for top-level `r.do()` (parser.go) -- last arg is function, rest are data args
- [ ] implement `chainDo` for chain form `.do(fn)` (parser.go)
- [ ] register `"do"` in `buildRBuilders()` and chain builders (parser.go)
- [ ] run parser unit tests -- must pass
- [ ] write integration test: `r.do(r.db("d").table("t"), (t) => t.count())` (integration)
- [ ] write integration test: `r.db("d").table("t").do((t) => t.count())` chain form (integration)
- [ ] run `make build` + `make test-integration` -- must pass

### Task 8: Verify and update documentation

- [ ] verify all 13 operations work via `r-cli query` with sample expressions
- [ ] run full test suite: `go test ./internal/reql/... ./internal/reql/parser/... -race -count=1`
- [ ] run full integration suite: `make test-integration`
- [ ] run linter: `make build`
- [ ] update `docs/rethinkdb-js-api.md` -- set `[x]` for all 13 newly implemented operations

## Technical Details

### Operations requiring only parser (reql functions exist):

| Operation | reql function | Parser syntax | Wire format |
|-----------|--------------|---------------|-------------|
| r.line | `Line(points...)` | `r.line([lon,lat], [lon,lat], ...)` | `[160, [p1, p2, ...]]` |
| r.polygon | `Polygon(points...)` | `r.polygon([lon,lat], ...)` | `[161, [p1, p2, ...]]` |
| r.circle | `Circle(center, r, opts?)` | `r.circle(center, r, {opts?})` | `[165, [center, r], opts?]` |
| r.time | `Time(y,m,d,tz)` | `r.time(y, m, d, tz)` or 7-arg | `[136, [y,m,d,tz]]` |
| r.binary | `Binary(data)` | `r.binary(data)` | `[155, [data]]` |
| r.do | `Do(args..., fn)` | `r.do(args, fn)` / `.do(fn)` | `[64, [fn, args...]]` |

### Operations requiring new reql functions + parser:

| Operation | New reql function | Wire format |
|-----------|------------------|-------------|
| r.object | `Object(pairs...)` | `[143, [k, v, ...]]` |
| r.range | `Range(args...)` | `[173, []]` / `[173, [end]]` / `[173, [s, e]]` |
| r.random | `Random(args...)` | `[151, [...], opts?]` |
| .info() | `Term.Info()` | `[79, [term]]` |
| .offsetsOf | `Term.OffsetsOf(pred)` | `[87, [seq, pred]]` |
| .fold | `Term.Fold(base, fn, opts?)` | `[187, [seq, base, fn], opts?]` |
| .bitAnd(n) | `Term.BitAnd(n)` | `[191, [val, n]]` |
| .bitOr(n) | `Term.BitOr(n)` | `[192, [val, n]]` |
| .bitXor(n) | `Term.BitXor(n)` | `[193, [val, n]]` |
| .bitNot() | `Term.BitNot()` | `[194, [val]]` |
| .bitSal(n) | `Term.BitSal(n)` | `[195, [val, n]]` |
| .bitSar(n) | `Term.BitSar(n)` | `[196, [val, n]]` |

### r.time 7-arg form

Current `reql.Time(year, month, day int, timezone string)` only supports 4 args. Need to extend or add `TimeWithTime(year, month, day, hour, minute, second int, timezone string)` for the 7-arg form. Alternative: make Time variadic.

### r.do wire format

`Do(a, b, fn)` serializes as `[64, [fn, a, b]]` -- function goes first in wire args but last in API call. The existing `reql.Do()` already handles this reversal. The chain form `expr.do(fn)` should produce `[64, [fn, expr]]`.
