# OptArgs camelCase to snake_case conversion

## Overview
- RethinkDB wire protocol expects snake_case keys in OptArgs (`left_bound`, `return_changes`, `dry_run`)
- Users coming from JS Data Explorer write camelCase (`leftBound`, `returnChanges`, `dryRun`)
- r-cli parser passes keys as-is, so camelCase keys cause RethinkDB errors like `Unrecognized optional argument 'leftBound'`
- Add automatic camelCase -> snake_case conversion for OptArgs keys in the parser

## Context
- `OptArgs` type: `map[string]interface{}` in `internal/reql/term.go:200`
- Parser entry: `parseOptArgs()` in `internal/reql/parser/parser.go:1900` calls `parseObjectBody()`
- `parseObjectKey()` at line 2128 returns key as-is (no transformation)
- Object literals for data (e.g. `filter({firstName: "Alice"})`) must NOT be converted -- only OptArgs
- Conversion point: `parseObjectBody()` -- shared by `parseOptArgs()` and `parseFoldOpts()`, both build OptArgs
- `parseObjectTerm()` (data objects) has its own independent parsing loop and is NOT affected
- Keys already in snake_case must pass through unchanged

## Approach
- Add a `camelToSnake(s string) string` helper function
- Apply it in `parseObjectBody()` to each key -- single chokepoint covering both `parseOptArgs()` and `parseFoldOpts()`
- `parseObjectTerm()` is separate and unaffected -- data object keys preserved as-is
- Pure transformation: `leftBound` -> `left_bound`, `returnChanges` -> `return_changes`
- Single-letter words and already snake_case keys pass through unchanged
- No hardcoded key list -- generic conversion handles any camelCase key

## Development Approach
- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**

## Implementation Steps

### Task 1: Write unit tests for camelToSnake helper
- [ ] add `TestCamelToSnake` table-driven test in `internal/reql/parser/parser_test.go` covering:
  - `leftBound` -> `left_bound`
  - `rightBound` -> `right_bound`
  - `returnChanges` -> `return_changes`
  - `dryRun` -> `dry_run`
  - `includeInitial` -> `include_initial`
  - `maxResults` -> `max_results`
  - `primaryKey` -> `primary_key`
  - `nonVotingReplicaTags` -> `non_voting_replica_tags`
  - already snake_case: `left_bound` -> `left_bound`
  - single word: `index` -> `index`
  - all lowercase: `shards` -> `shards`
  - empty string: `""` -> `""`
  - single uppercase: `X` -> `x`
  - consecutive uppercase (naive): `maxBPS` -> `max_b_p_s` (no real RethinkDB opts have acronyms)
- [ ] run tests -- `TestCamelToSnake` must fail (function not yet implemented)

### Task 2: Implement camelToSnake helper
- [ ] add `camelToSnake(s string) string` unexported function in `internal/reql/parser/parser.go`
- [ ] logic: iterate runes, insert `_` before each uppercase letter and lowercase it; skip insert at position 0
- [ ] run tests -- `TestCamelToSnake` must pass
- [ ] run full parser test suite -- all tests must pass

### Task 3: Write unit tests for OptArgs key conversion in parser
- [ ] add `TestParse_OptArgs_CamelCaseConversion` in `internal/reql/parser/parser_test.go`:
  - `r.db("d").table("t").between(1, 10, {index: "x", leftBound: "closed"})` -> OptArgs keys are `index`, `left_bound`
  - `r.db("d").table("t").insert({a: 1}, {returnChanges: true})` -> OptArgs key is `return_changes`
  - `r.db("d").table("t").getAll("a", {index: "idx"})` -> no conversion needed (already snake_case)
  - `r.db("d").table("t").changes({includeInitial: true})` -> OptArgs key is `include_initial`
  - `r.db("d").table("t").filter({firstName: "Alice"})` -> data object key stays `firstName` (NOT converted)
  - `r.table("t").changes({"includeInitial": true})` -> string-literal OptArgs key also converted to `include_initial`
  - `r.expr([1]).fold(0, (a, x) => a.add(x), {finalEmit: a => a})` -> fold OptArgs key `final_emit`
- [ ] run tests -- new tests must fail (conversion not yet wired)

### Task 4: Wire camelToSnake into parseObjectBody
- [ ] modify `parseObjectBody()` in `internal/reql/parser/parser.go` to apply `camelToSnake()` to each key (line `opts[key] = val` -> `opts[camelToSnake(key)] = val`)
- [ ] this covers both `parseOptArgs()` and `parseFoldOpts()` automatically (both delegate to `parseObjectBody`)
- [ ] run tests -- `TestParse_OptArgs_CamelCaseConversion` must pass (including fold opts and data object preservation)
- [ ] run full parser test suite -- all existing tests must still pass
- [ ] run `make build` (includes linter) -- must pass

### Task 5: Write integration tests
- [ ] add `TestParserOptArgsCamelCaseConversion` in `internal/integration/parser_optargs_test.go`:
  - `r.table("t").between(lowVal, highVal, {leftBound: "closed", rightBound: "closed"})` -- must succeed
  - `r.table("t").insert({doc}, {returnChanges: true})` -- must return changes
  - `r.table("t").getAll(val, {index: "idx"})` with camelCase -- must use index correctly
- [ ] run integration tests -- must pass

### Task 6: Verify acceptance criteria
- [ ] verify camelCase OptArgs keys are converted: `leftBound` -> `left_bound`
- [ ] verify snake_case keys still work: `left_bound` -> `left_bound`
- [ ] verify data object keys are NOT converted: `filter({firstName: "Alice"})` keeps `firstName`
- [ ] run full test suite (unit tests)
- [ ] run integration tests
- [ ] run linter -- all issues must be fixed

## Technical Details

### camelToSnake algorithm
```
input: "leftBound"
iterate runes:
  'l' -> 'l'
  'e' -> 'e'
  'f' -> 'f'
  't' -> 't'
  'B' (uppercase, pos > 0) -> '_' + 'b'
  'o' -> 'o'
  'u' -> 'u'
  'n' -> 'n'
  'd' -> 'd'
output: "left_bound"
```

### Affected parsing functions
- `parseObjectBody()` -- add `camelToSnake(key)` here; single chokepoint for all OptArgs
- `parseOptArgs()` -- delegates to `parseObjectBody`, no changes needed
- `parseFoldOpts()` -- delegates to `parseObjectBody`, no changes needed
- `parseObjectTerm()` -- separate code path for data objects, NOT affected

### Known camelCase OptArgs keys users may write
| camelCase | snake_case |
|-----------|------------|
| leftBound | left_bound |
| rightBound | right_bound |
| returnChanges | return_changes |
| dryRun | dry_run |
| includeInitial | include_initial |
| maxResults | max_results |
| primaryKey | primary_key |
| nonVotingReplicaTags | non_voting_replica_tags |
| includeOffsets | include_offsets |
| includeTypes | include_types |
| maxBatchRows | max_batch_rows |
| minBatchRows | min_batch_rows |
| maxBatchBytes | max_batch_bytes |
| maxBatchSeconds | max_batch_seconds |
| stopOnError | stop_on_error |
