# Plan: ReQL Object Operations and Aggregation

## Overview

Extend ReQL term builder with object operations (GetField, HasFields, Merge, Distinct) and aggregation methods (Map, Reduce, Group, Sum, Avg, Min, Max).

Package: `internal/reql`

Depends on: `04-reql-core`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Object operations

- [x] Test: `.GetField("name")` -> `[31,[<term>,"name"]]`
- [x] Test: `.HasFields("a","b")` -> `[32,[<term>,"a","b"]]`
- [x] Test: `.Merge(obj)` -> `[35,[<term>,<obj>]]`
- [x] Test: `.Distinct()` -> `[42,[<term>]]`
- [x] Implement: object operation methods

### Task 2: Aggregation

- [x] Test: `.Map(func)` -> `[38,[<term>,<func>]]`
- [x] Test: `.Reduce(func)` -> `[37,[<term>,<func>]]`
- [x] Test: `.Group(field)` -> `[144,[<term>,<field>]]`
- [x] Test: `.Ungroup()` -> `[150,[<term>]]`
- [x] Test: `.Sum(field)` -> `[145,[<term>,<field>]]`
- [x] Test: `.Avg(field)` -> `[146,[<term>,<field>]]`
- [x] Test: `.Min(field)` -> `[147,[<term>,<field>]]`
- [x] Test: `.Max(field)` -> `[148,[<term>,<field>]]`
- [x] Implement: aggregation methods
