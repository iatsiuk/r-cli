# Plan: Integration Tests for Missing Methods

## Overview

Add integration tests for 26 methods listed in `docs/without-integration.md`. All methods already have parser support and unit tests but lack integration tests against a live RethinkDB instance. Tests are added to existing files following established patterns.

## Context

- Methods grouped into 5 categories: Joins (1), Aggregation (4), Strings (2), Logic (6), Time (11), Administration (3)
- Existing test patterns: `t.Parallel()`, `sanitizeID(t.Name())`, `setupTestDB`/`createTestTable`/`seedTable`, `atomRows` for array results
- Scalar operations tested via `reql.Datum(x).Method()`, table operations via `reql.Func`/`reql.Var` lambdas

## File Placement

| Category | Target file | Rationale |
|---|---|---|
| Joins (`outerJoin`) | `string_time_join_test.go` | already has EqJoin, Zip |
| Aggregation (`sum`, `avg`, `min`, `max`) | `orderby_agg_test.go` | already has Count, Distinct |
| Strings (`split`, `downcase`) | `string_time_join_test.go` | already has Match, ToJSONString |
| Logic (`or`, `ne`, `lt`, `le`, `ge`, `not`) | new `logic_ops_test.go` | no existing file for comparison ops |
| Time (11 methods) | `time_ops_test.go` | already has During |
| Administration (`sync`, `rebalance`, `reconfigure`) | `table_test.go` | already has Config, Status, Wait |

## Development Approach

- Complete each task fully before moving to the next
- Run `make test-integration` after each task
- Each test function follows the standard pattern: `t.Parallel()` + `newExecutor` + `setupTestDB` + `createTestTable` + `seedTable`

## Validation Commands

- `go test -tags integration ./internal/integration/... -run TestName -race -count=1`
- `make test-integration`
- `make build`

## Implementation Steps

### Task 1: Aggregation tests (sum, avg, min, max)
File: `internal/integration/orderby_agg_test.go` (append)

- [x] `TestSum` -- `reql.DB(db).Table(t).Sum("score")` on seeded numeric docs, verify correct total
- [x] `TestAvg` -- `reql.DB(db).Table(t).Avg("score")` on seeded docs, verify average
- [x] `TestMin` -- `reql.DB(db).Table(t).Min("score")` returns doc with minimum score
- [x] `TestMax` -- `reql.DB(db).Table(t).Max("score")` returns doc with maximum score
- [x] run `make test-integration` -- must pass

### Task 2: Logic operation tests (or, ne, lt, le, ge, not)
File: `internal/integration/logic_ops_test.go` (new)

- [x] `TestNe` -- scalar: `reql.Datum(1).Ne(2)` -> true, `reql.Datum(1).Ne(1)` -> false
- [x] `TestLt` -- scalar: `reql.Datum(1).Lt(2)` -> true, `reql.Datum(2).Lt(1)` -> false
- [x] `TestLe` -- scalar: `reql.Datum(1).Le(1)` -> true, `reql.Datum(2).Le(1)` -> false
- [x] `TestGe` -- scalar: `reql.Datum(2).Ge(2)` -> true, `reql.Datum(1).Ge(2)` -> false
- [x] `TestOr` -- scalar: `reql.Datum(false).Or(true)` -> true, `reql.Datum(false).Or(false)` -> false
- [x] `TestNot` -- scalar: `reql.Datum(true).Not()` -> false, `reql.Datum(false).Not()` -> true
- [x] `TestFilterNe` -- filter table docs where field != value using `.Ne()` in predicate
- [x] `TestFilterLtLe` -- filter table docs using `.Lt()` and `.Le()` boundary conditions
- [x] `TestFilterGeOr` -- filter table docs using `.Ge()` combined with `.Or()`
- [x] run `make test-integration` -- must pass

### Task 3: String operation tests (split, downcase)
File: `internal/integration/string_time_join_test.go` (append)

- [x] `TestSplit` -- scalar: `reql.Datum("a,b,c").Split(",")` -> `["a","b","c"]`; no-arg split on whitespace
- [x] `TestDowncase` -- scalar: `reql.Datum("HELLO").Downcase()` -> `"hello"`
- [x] `TestSplitOnTableField` -- map over table docs, split a string field, verify array results
- [x] `TestDowncaseOnTableField` -- map over table docs, downcase a string field, verify lowercased results
- [x] run `make test-integration` -- must pass

### Task 4: OuterJoin test
File: `internal/integration/string_time_join_test.go` (append)

- [ ] `TestOuterJoin` -- two tables (users, orders); outerJoin with lambda predicate matching user_id; verify all left rows present even without matching right rows (null right side for unmatched)
- [ ] run `make test-integration` -- must pass

### Task 5: Time operation tests (11 methods)
File: `internal/integration/time_ops_test.go` (append)

- [ ] `TestToISO8601` -- `reql.EpochTime(1704067200).ToISO8601()` -> ISO string containing "2024-01-01"
- [ ] `TestInTimezone` -- `reql.EpochTime(epoch).InTimezone("+05:00").Hours()` -> verify shifted hour
- [ ] `TestTimezone` -- `reql.EpochTime(epoch).InTimezone("+03:00").Timezone()` -> `"+03:00"`
- [ ] `TestDate` -- `reql.EpochTime(epoch).Date().ToEpochTime()` -> midnight of that day
- [ ] `TestTimeOfDay` -- `reql.EpochTime(epoch).TimeOfDay()` -> seconds since midnight
- [ ] `TestMonth` -- `reql.EpochTime(1704067200).Month()` -> 1 (January)
- [ ] `TestDay` -- `reql.EpochTime(1704067200).Day()` -> 1
- [ ] `TestDayOfWeek` -- `reql.EpochTime(1704067200).DayOfWeek()` -> 1 (Monday, 2024-01-01)
- [ ] `TestDayOfYear` -- `reql.EpochTime(1704067200).DayOfYear()` -> 1
- [ ] `TestHoursMinutesSeconds` -- `reql.Time(2024, 1, 15, 14, 30, 45, "+00:00")` -> verify `.Hours()` = 14, `.Minutes()` = 30, `.Seconds()` = 45
- [ ] run `make test-integration` -- must pass

### Task 6: Administration tests (sync, rebalance, reconfigure)
File: `internal/integration/table_test.go` (append)

- [ ] `TestTableSync` -- create table, insert docs, call `.Sync()`, verify returns sync result (no error)
- [ ] `TestTableReconfigure` -- create table, call `.Reconfigure(reql.OptArgs{"shards": 1, "replicas": 1})`, verify returns reconfigure result with `reconfigured` field
- [ ] `TestTableRebalance` -- create table, call `.Rebalance()`, verify returns rebalance result (no error)
- [ ] run `make test-integration` -- must pass

### Task 7: Final verification
- [ ] run full `make test-integration` -- all tests must pass
- [ ] run `make build` -- linter must pass
- [ ] update `docs/without-integration.md` -- remove covered methods or mark as done

## Technical Details

- `Sum("field")`, `Avg("field")` return scalar values (float64)
- `Min("field")`, `Max("field")` return the full document with min/max value, not just the value
- `OuterJoin` takes a predicate function (2-arg Func), returns `{left: ..., right: ...}` pairs; unmatched left rows have `right: null`
- `Split()` with no args splits on whitespace; `Split(",")` splits on delimiter
- `Timezone()` returns string like `"+03:00"`; `Date()` returns TIME pseudo-type at midnight; `TimeOfDay()` returns seconds as float
- `DayOfWeek()` returns 1=Monday through 7=Sunday
- `Sync()` returns `{"synced": 1}`; `Reconfigure` and `Rebalance` return result objects with status fields
- `reql.Time(year, month, day, hour, min, sec, tz)` constructs a time value server-side
