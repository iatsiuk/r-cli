# Integration tests for uncovered operations

## Overview

Add integration tests for all operations listed in `without-tests.md` -- 39 operations across 8 categories that have parser + unit tests but no integration test coverage against a live RethinkDB instance.

## Context

- All operations already have working `reql.Term` builders and unit tests
- Integration test infrastructure exists in `internal/integration/` with helpers: `newExecutor`, `setupTestDB`, `createTestTable`, `seedTable`, `parseWriteResult`, `atomRows`, `closeCursor`
- Tests run against RethinkDB 2.4.4 container via testcontainers-go
- Build tag: `//go:build integration`

## Development Approach

- Add tests to existing integration test files where logical grouping fits
- Create new test files for categories not yet represented
- Each task adds tests, seeds data, validates results
- Follow existing patterns: `t.Parallel()`, `sanitizeID`, `setupTestDB`, `t.Cleanup`
- **All tests must pass before starting next task**

## Implementation Steps

### Task 1: Arithmetic operations (sub, div, mod, floor, ceil, round)

File: `internal/integration/arithmetic_test.go` (new)

- [ ] create test file with build tag and helpers
- [ ] test `Sub` -- `r.expr(10).sub(3)` returns 7, chained `r.expr(10).sub(3).sub(2)` returns 5
- [ ] test `Div` -- `r.expr(10).div(3)` returns float, division by zero returns error
- [ ] test `Mod` -- `r.expr(10).mod(3)` returns 1, mod by zero returns error
- [ ] test `Floor` -- `r.expr(3.7).floor()` returns 3, negative `r.expr(-2.3).floor()` returns -3
- [ ] test `Ceil` -- `r.expr(3.2).ceil()` returns 4, negative `r.expr(-2.7).ceil()` returns -2
- [ ] test `Round` -- `r.expr(3.5).round()` returns 4, `r.expr(3.4).round()` returns 3
- [ ] test arithmetic on table fields -- seed docs with numeric fields, `map` with `Sub`/`Div`/`Mod`/`Floor`/`Ceil`/`Round` via `Func`+`Var`
- [ ] run `make test-integration` -- must pass

### Task 2: Type operations (coerceTo, typeOf)

File: `internal/integration/type_ops_test.go` (new)

- [ ] test `TypeOf` -- `r.expr(1).typeOf()` returns `"NUMBER"`, `r.expr("s").typeOf()` returns `"STRING"`, `r.expr([]).typeOf()` returns `"ARRAY"`, `r.expr(null).typeOf()` returns `"NULL"`, table typeOf returns `"TABLE"`
- [ ] test `CoerceTo` -- number to string `r.expr(42).coerceTo("STRING")` returns `"42"`, string to number `r.expr("123").coerceTo("NUMBER")` returns 123, invalid coercion returns error
- [ ] test `CoerceTo` on table field -- seed docs, map field with coerceTo
- [ ] run `make test-integration` -- must pass

### Task 3: String operation (toJSONString)

File: `internal/integration/string_time_join_test.go` (append to existing)

- [ ] test `ToJSONString` -- `r.expr({"a":1}).toJSONString()` returns `"{\"a\":1}"`, `r.expr([1,2]).toJSONString()` returns `"[1,2]"`, `r.expr("hello").toJSONString()` returns `"\"hello\""`, number returns `"42"`
- [ ] run `make test-integration` -- must pass

### Task 4: Sequence/collection operations (concatMap, isEmpty, contains, union, withFields, keys, values)

File: `internal/integration/collection_ops_test.go` (new)

- [ ] test `IsEmpty` -- empty table returns true, non-empty returns false, empty filter result returns true
- [ ] test `Contains` -- `r.expr([1,2,3]).contains(2)` returns true, `contains(5)` returns false; table `.contains({"id":"x"})` variant
- [ ] test `ConcatMap` -- `r.expr([[1,2],[3,4]]).concatMap(fn)` flattens to `[1,2,3,4]`; table with array field, concatMap to extract nested arrays
- [ ] test `Union` -- `r.expr([1,2]).union([3,4])` returns `[1,2,3,4]`; union of two table queries
- [ ] test `WithFields` -- seed docs with optional fields, `withFields("name","email")` returns only docs that have both fields
- [ ] test `Keys` -- `r.expr({"a":1,"b":2}).keys()` returns `["a","b"]`; on table row via map
- [ ] test `Values` -- `r.expr({"a":1,"b":2}).values()` returns `[1,2]`; on table row via map
- [ ] run `make test-integration` -- must pass

### Task 5: Array mutation operations (append, prepend, slice, difference, insertAt, deleteAt, changeAt, spliceAt)

File: `internal/integration/array_ops_test.go` (new)

- [ ] test `Append` -- `r.expr([1,2]).append(3)` returns `[1,2,3]`
- [ ] test `Prepend` -- `r.expr([2,3]).prepend(1)` returns `[1,2,3]`
- [ ] test `Slice` -- `r.expr([0,1,2,3,4]).slice(1,3)` returns `[1,2]`; single arg `slice(2)` returns `[2,3,4]`
- [ ] test `Difference` -- `r.expr([1,2,3,2]).difference([2])` returns `[1,3]`
- [ ] test `InsertAt` -- `r.expr([0,1,3]).insertAt(2, 2)` returns `[0,1,2,3]`
- [ ] test `DeleteAt` -- `r.expr([0,1,2,3]).deleteAt(1)` returns `[0,2,3]`
- [ ] test `ChangeAt` -- `r.expr([0,1,2]).changeAt(1, 9)` returns `[0,9,2]`
- [ ] test `SpliceAt` -- `r.expr([0,3,4]).spliceAt(1, [1,2])` returns `[0,1,2,3,4]`
- [ ] test array ops on table fields -- seed docs with array fields, update using append/prepend, read back and verify
- [ ] run `make test-integration` -- must pass

### Task 6: Set operations (setInsert, setIntersection, setUnion, setDifference)

File: `internal/integration/set_ops_test.go` (new)

- [ ] test `SetInsert` -- `r.expr([1,2,3]).setInsert(2)` returns `[1,2,3]` (no dup), `setInsert(4)` returns `[1,2,3,4]`
- [ ] test `SetIntersection` -- `r.expr([1,2,3]).setIntersection([2,3,4])` returns `[2,3]`
- [ ] test `SetUnion` -- `r.expr([1,2]).setUnion([2,3])` returns `[1,2,3]`
- [ ] test `SetDifference` -- `r.expr([1,2,3]).setDifference([2])` returns `[1,3]`
- [ ] test set ops on table fields -- seed docs with array fields, update using set operations, verify uniqueness invariants
- [ ] run `make test-integration` -- must pass

### Task 7: Top-level constructors (r.minval, r.maxval, r.error, r.args, r.literal, r.geoJSON)

File: `internal/integration/constructors_test.go` (new)

- [ ] test `MinVal`/`MaxVal` -- `between(r.minval, r.maxval)` returns all docs; `between(r.minval, "m")` returns docs with id < "m"; `between("m", r.maxval)` returns docs with id >= "m"
- [ ] test `Error` -- `r.error("boom")` returns ReqlRuntimeError with message "boom"; `r.branch(cond, val, r.error("bad"))` returns error on false branch
- [ ] test `Args` -- `r.args(["a","b"])` used as argument spread: `getAll(r.args(["id1","id2"]))` returns matching docs
- [ ] test `Literal` -- update with `r.literal({"new":"obj"})` replaces nested object entirely instead of merging; update with `r.literal()` (no args) removes the field
- [ ] test `GeoJSON` -- `r.geoJSON({"type":"Point","coordinates":[-73.9,40.7]})` creates a geometry; insert into geo-indexed table and query with getIntersecting
- [ ] run `make test-integration` -- must pass

### Task 8: Time operation (during) and geo operations (toGeoJSON, intersects, includes, fill, polygonSub)

File for during: `internal/integration/time_ops_test.go` (new)
File for geo: `internal/integration/geo_test.go` (append to existing)

- [ ] test `During` -- seed docs with time fields, filter with `during(start, end)` returns docs in range; test open/closed bounds with optargs
- [ ] test `ToGeoJSON` -- insert point, call `toGeoJSON()`, verify returns GeoJSON object `{"type":"Point","coordinates":[...]}`
- [ ] test `Intersects` -- create two geometries (polygon and point), `polygon.intersects(point)` returns true/false
- [ ] test `Includes` -- `polygon.includes(point)` returns true for point inside, false for outside
- [ ] test `Fill` -- create a line that forms a closed loop, `line.fill()` returns a polygon
- [ ] test `PolygonSub` -- create outer polygon and inner polygon (hole), `outer.polygonSub(inner)` returns polygon with hole; verify with includes on point inside hole returns false
- [ ] run `make test-integration` -- must pass

### Task 9: Verify all operations covered

- [ ] run full `make test-integration` -- all tests pass
- [ ] run `make build` (includes linter) -- no issues
- [ ] verify every operation from `without-tests.md` has at least one integration test
- [ ] mark all items in `without-tests.md` as `[x]`

## Technical Details

- All tests use `reql.Datum()` or direct Go literals for input data
- Arithmetic/array/set operations tested both as standalone expressions (`r.expr(val).op()`) and on table fields via `Func`+`Var` pattern
- Geo tests reuse existing geo infrastructure from `geo_test.go` (geo index creation, point insertion)
- `During` requires time fields -- insert docs with `r.now()` or `r.epochTime()`, then filter with known bounds
- `Literal` semantics: when used inside `update()`, replaces the nested value instead of merging; `r.literal()` with no args removes the field

## Post-Completion

- Remove `without-tests.md` from project root (or keep as tracking doc with all items checked)
