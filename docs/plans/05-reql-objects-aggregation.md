# Plan: ReQL Object Operations and Aggregation

## Overview

Extend ReQL term builder with object operations (GetField, HasFields, Merge, Distinct) and aggregation methods (Map, Reduce, Group, Sum, Avg, Min, Max).

Package: `internal/reql`

Depends on: `04-reql-core`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Object operations

- [ ] Test: `.GetField("name")` -> `[31,[<term>,"name"]]`
- [ ] Test: `.HasFields("a","b")` -> `[32,[<term>,"a","b"]]`
- [ ] Test: `.Merge(obj)` -> `[35,[<term>,<obj>]]`
- [ ] Test: `.Distinct()` -> `[42,[<term>]]`
- [ ] Implement: object operation methods

### Task 2: Aggregation

- [ ] Test: `.Map(func)` -> `[38,[<term>,<func>]]`
- [ ] Test: `.Reduce(func)` -> `[37,[<term>,<func>]]`
- [ ] Test: `.Group(field)` -> `[144,[<term>,<field>]]`
- [ ] Test: `.Ungroup()` -> `[150,[<term>]]`
- [ ] Test: `.Sum(field)` -> `[145,[<term>,<field>]]`
- [ ] Test: `.Avg(field)` -> `[146,[<term>,<field>]]`
- [ ] Test: `.Min(field)` -> `[147,[<term>,<field>]]`
- [ ] Test: `.Max(field)` -> `[148,[<term>,<field>]]`
- [ ] Implement: aggregation methods
