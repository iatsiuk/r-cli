# Plan: ReQL Core Term Builder

## Overview

Build ReQL terms as JSON-serializable structures. Core layer: datum encoding, array wrapping, chainable term builder, write and read operations.

Package: `internal/reql`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Datum encoding and MAKE_ARRAY

- [x] Test: string "foo" -> `"foo"` (raw JSON)
- [x] Test: number 42 -> `42`
- [x] Test: bool true -> `true`
- [x] Test: nil -> `null`
- [x] Implement: datum pass-through in term serialization
- [x] Test: Go slice `[10,20,30]` -> `[2,[10,20,30]]`
- [x] Test: empty slice -> `[2,[]]`
- [x] Test: nested array -> properly wrapped
- [x] Implement: `Array(items ...interface{}) Term`

### Task 2: Core term builder

- [ ] Test: `DB("test")` -> `[14,["test"]]`
- [ ] Test: `DB("test").Table("users")` -> `[15,[[14,["test"]],"users"]]`
- [ ] Test: chained `.Filter({...})` -> correct nested structure
- [ ] Implement: `Term` struct with chainable methods, `MarshalJSON()`

### Task 3: Write operations

- [ ] Test: `.Insert(doc)` -> `[56,[<table_term>,<doc>]]`
- [ ] Test: `.Update(doc)` -> `[53,[<table_term>,<doc>]]`
- [ ] Test: `.Delete()` -> `[54,[<table_term>]]`
- [ ] Test: `.Replace(doc)` -> `[55,[<table_term>,<doc>]]`
- [ ] Implement: Insert, Update, Delete, Replace methods

### Task 4: Read operations

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

### Task 5: Comparison, logic operators and arithmetic

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
- [ ] Test: `.Add(value)` -> `[24,[<term>,<value>]]`
- [ ] Test: `.Sub(value)` -> `[25,[<term>,<value>]]`
- [ ] Test: `.Mul(value)` -> `[26,[<term>,<value>]]`
- [ ] Test: `.Div(value)` -> `[27,[<term>,<value>]]`
- [ ] Implement: arithmetic methods
