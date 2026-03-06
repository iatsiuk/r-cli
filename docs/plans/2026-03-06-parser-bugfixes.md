# Parser Bugfixes: Nested Object Fields + toJSON/toJsonString Aliases

## Overview

Two parser bugs discovered from `~/.r-cli/parser-errors.log` analysis:

1. **`without`/`pluck`/`hasFields`/`withFields` reject object arguments** -- `parseStringList()` accepts only string literals, but RethinkDB supports nested objects for nested field selection (e.g. `without({perks: {refill: true}})`)
2. **Missing `toJSON`/`toJsonString` aliases** -- RethinkDB JS driver provides `toJSON()` and `toJsonString()` (lowercase j), but parser only registers `toJSONString` (non-standard capitalization)

## Context

- Parser chain methods: `internal/reql/parser/parser.go`
- Parser tests: `internal/reql/parser/parser_test.go`
- Term builder: `internal/reql/term.go`
- Term tests: `internal/reql/term_test.go`
- Integration tests: `internal/integration/`
- `chainWithout`, `chainPluck`, `chainHasFields`, `chainWithFields` all use `parseStringList()` which only accepts `("str1", "str2", ...)`
- RethinkDB `without`/`pluck` accept mixed args: strings AND objects (e.g. `pluck("name", {address: ["city"]})`)
- `toJSONString` registered at line ~1575 as `noArgChain`
- RethinkDB JS driver (`ast.coffee`) registers `toJsonString` (lowercase j) and `toJSON` -- NOT `toJSONString`

### Critical: MAKE_ARRAY serialization bug

`parseObjectTerm()` delegates values to `parseExpr()`, which parses `["city"]` via `parseArrayTerm()` -> `reql.Array("city")` = `Term{termType: MAKE_ARRAY}`. When stored in `Datum(map)`, this serializes as `{"address":[2,["city"]]}` instead of the correct `{"address":["city"]}`. Field selectors need native Go data (maps/slices), not reql.Term trees.

### Signature change: `[]string...` call sites

Changing `Pluck(fields ...string)` to `Pluck(fields ...interface{})` breaks 4 call sites in `parser.go` that use `[]string...` expansion. These must be updated along with the signature change.

## Development Approach

- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Progress Tracking

- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Implementation Steps

### Task 1: Add toJSON + toJsonString aliases -- tests

- [x] add parser test: `toJSON` chain produces same term as `toJSONString` (e.g. `r.db("d").table("t").toJSON()` == `dbterm.ToJSONString()`)
- [x] add parser test: `toJsonString` chain (lowercase j) produces same term as `toJSONString`
- [x] run parser tests -- new tests must FAIL (methods not registered)

### Task 2: Add toJSON + toJsonString aliases -- implementation

- [x] register `toJSON` as `noArgChain` in `registerStringChain()` mapping to `ToJSONString()`, next to existing `toJSONString` entry
- [x] register `toJsonString` as `noArgChain` similarly
- [x] run parser tests -- all must pass
- [x] run `make build` -- linter must pass

### Task 3: Support object args in pluck/without/hasFields/withFields -- term layer tests

The term builder (`term.go`) currently accepts only `...string`. Before changing the parser, the builder must support mixed args (strings + maps).

- [x] add term test: `Pluck` with native Go map arg -- `t.Pluck("name", map[string]interface{}{"address": []interface{}{"city"}})` serializes as `[33,[term,"name",{"address":["city"]}]]` (raw array, NOT `[2,["city"]]`)
- [x] add term test: `Without` with map arg serializes correctly
- [x] add term test: `HasFields` with map arg serializes correctly
- [x] add term test: `WithFields` with map arg serializes correctly
- [x] add term test: backward compat -- `Pluck("a", "b")` still serializes as before
- [x] run term tests -- new tests must FAIL (methods accept only strings)

### Task 4: Support object args in pluck/without/hasFields/withFields -- term layer implementation

- [x] change `Pluck` signature from `(fields ...string)` to `(fields ...interface{})`, use `toTerm(f)` for each arg (same pattern as `GetAll`)
- [x] change `Without` signature similarly
- [x] change `HasFields` signature similarly
- [x] change `WithFields` signature similarly
- [x] fix 4 `[]string...` call sites in `parser.go` (`chainPluck`, `chainWithout`, `chainHasFields`, `chainWithFields`) -- these will be fully rewritten in Task 6, but must compile now
- [x] run term tests -- all must pass
- [x] run `go test ./internal/reql/... -race` -- no regressions
- [x] run `make build` -- linter must pass

### Task 5: Parser field selector support -- tests

- [x] add parser test: `without({perks: {refill: true}})` parses to correct term with nested object
- [x] add parser test: `pluck("name", {address: ["city"]})` parses to correct term with mixed string + object args
- [x] add parser test: `hasFields({profile: true})` parses to correct term
- [x] add parser test: `withFields("id", {stats: true})` parses to correct term
- [x] add parser test: `pluck("a", "b")` still works (backward compat, string-only)
- [x] add parser wire JSON test: marshal `pluck("name", {address: ["city"]})` and verify output contains `{"address":["city"]}`, NOT `[2,["city"]]`
- [x] add parser error test: `pluck(123)` -- rejects non-string/non-object arg
- [x] add parser error test: `pluck("a",)` -- rejects trailing comma
- [x] run parser tests -- new tests must FAIL (parseStringList rejects objects)

### Task 6: Parser field selector support -- implementation

- [x] add `parseDatumValue()` method -- parses a JSON-like datum literal returning native Go types: string, number, bool, null, `[]interface{}` (for arrays), `map[string]interface{}` (for objects); NO `reql.Term` / `reql.Array` -- pure Go data; nested recursion for objects/arrays
- [x] add `parseFieldSelectors()` method -- parses `(arg, arg, ...)` where each arg is a string literal or a `{...}` object; delegates object values to `parseDatumValue()`; rejects other arg types (numbers, bools, r.* calls) with error
- [x] replace `parseStringList()` calls in `chainPluck`/`chainWithout`/`chainHasFields`/`chainWithFields` with `parseFieldSelectors()`, pass results to updated `...interface{}` methods
- [x] run parser tests -- all must pass
- [x] run `make build` -- linter must pass

### Task 7: Integration tests

- [x] add integration test: `pluck` with nested object -- insert doc with nested fields, `pluck("name", {address: ["city"]})`, verify only selected fields returned
- [x] add integration test: `without` with nested object -- insert doc with nested field, `without({address: {zip: true}})`, verify nested field removed
- [x] add integration test: `hasFields` with nested object -- filter docs by nested field presence
- [x] add integration test: `withFields` with nested object -- select docs having nested fields
- [x] add integration test: `toJSON()` -- verify returns JSON string representation
- [x] add integration test: `toJsonString()` -- verify same result as `toJSON()`
- [x] run `make test-integration` -- all must pass

NOTE: Integration tests revealed that `parseDatumArray` was returning `[]interface{}` which
marshals as plain JSON array `["city"]`. RethinkDB interprets bare arrays in term arg positions
as term arrays (first element must be a TermType integer), causing "Expected a TermType as a
NUMBER but found STRING" error. Fixed `parseDatumArray` to return `reql.Array(...)` (MAKE_ARRAY
term) so arrays serialize as `[2, ["city"]]` per protocol spec. Updated parser_test.go and
term_test.go to expect correct MAKE_ARRAY wire format.

### Task 8: Verify acceptance criteria

- [x] verify `toJSON()` and `toJsonString()` work as aliases for `toJSONString()`
- [x] verify `without({nested: true})` parses and executes correctly
- [x] verify `pluck("name", {address: ["city"]})` parses, serializes correctly (no MAKE_ARRAY), and executes
- [x] verify backward compat: string-only `pluck`/`without`/`hasFields`/`withFields` unchanged
- [x] verify error on invalid selector args: `pluck(123)`, `without(true)`
- [x] run full test suite: `go test ./... -race`
- [x] run `make test-integration`
- [x] run `make build` -- linter passes

## Technical Details

### toJSON / toJsonString aliases

RethinkDB JS driver (`ast.coffee`) registers two names: `toJsonString` (lowercase j) and `toJSON`. Both create `ToJsonString` AST class -> wire term type `TO_JSON_STRING` (172). Our parser already has `toJSONString` (non-standard capitalization but functional). Adding `toJSON` and `toJsonString` as aliases; keeping existing `toJSONString` for backward compat.

### Nested object field selectors

RethinkDB `pluck`/`without`/`hasFields`/`withFields` accept mixed arguments:
- String: `"fieldName"` -- selects/removes top-level field
- Object: `{fieldName: true}` -- selects/removes nested field
- Object: `{fieldName: ["sub1", "sub2"]}` -- selects specific nested subfields
- Object: `{fieldName: {nested: true}}` -- deep nesting

Wire JSON example for `pluck("name", {address: ["city"]})`:
```json
[33, [<term>, "name", {"address": ["city"]}]]
```

The array `["city"]` MUST be a raw JSON array, NOT a MAKE_ARRAY term `[2,["city"]]`.

### Term builder change

Change `Pluck`/`Without`/`HasFields`/`WithFields` from `...string` to `...interface{}`, use `toTerm(f)` per arg (same pattern as existing `GetAll`). When caller passes native `map[string]interface{}` with `[]interface{}` arrays, `Datum()` wraps them correctly -- `json.Marshal` produces raw JSON objects/arrays with no term type wrappers.

### parseDatumValue() -- new parser helper

Parses JSON-like datum literals into native Go types:
- string `"..."` -> `string`
- number `123`, `3.14` -> `float64` / `int`
- bool `true`/`false` -> `bool`
- null -> `nil`
- array `[v, ...]` -> `[]interface{}` (recursive, NOT reql.Array)
- object `{k: v, ...}` -> `map[string]interface{}` (recursive, keys as-is -- no camelToSnake)

This avoids the MAKE_ARRAY problem: `parseExpr()` -> `parseArrayTerm()` -> `reql.Array()` produces Term wrappers, but `parseDatumValue()` produces plain Go slices.

### parseFieldSelectors() -- new parser helper

Replaces `parseStringList()` for field-selector chains:
```
parseFieldSelectors() -> ([]interface{}, error)
```
Parses `(arg, arg, ...)` where each arg is:
- string literal -> string value (via `parseDatumValue` or direct token check)
- `{key: val, ...}` -> `map[string]interface{}` (via `parseDatumValue`)
- anything else -> error (numbers, bools, r.* calls rejected)

Returns mixed `[]interface{}` suitable for the updated `Pluck`/`Without`/`HasFields`/`WithFields` methods.

## Post-Completion

**Manual verification:**
- Test in REPL: `r.table("users").pluck("name", {address: ["city"]})`
- Test in REPL: `r.table("users").get("id").toJSON()`
- Test in REPL: `r.table("users").get("id").toJsonString()`
- Clean up test entries from `~/.r-cli/parser-errors.log`
