# Plan: ReQL Admin, Optargs and Query Serialization

## Overview

Database/table admin terms, optarg support on all terms that need it, and full query serialization (START/CONTINUE/STOP wrapping).

Package: `internal/reql`

Depends on: `04-reql-core`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Database/table admin terms

- [ ] Test: `DBCreate("name")` -> `[57,["name"]]`
- [ ] Test: `DBDrop("name")` -> `[58,["name"]]`
- [ ] Test: `DBList()` -> `[59,[]]`
- [ ] Test: `TableCreate("name")` -> `[60,[<db_term>,"name"]]`
- [ ] Test: `TableDrop("name")` -> `[61,[<db_term>,"name"]]`
- [ ] Test: `TableList()` -> `[62,[<db_term>]]`
- [ ] Implement: admin term builders

### Task 2: Term optargs

- [ ] Test: `.Insert(doc)` with `conflict` optarg -> `[56,[<table>,<doc>],{"conflict":"replace"}]`
- [ ] Test: `.Insert(doc)` with `return_changes` optarg
- [ ] Test: `.Changes()` with `include_initial` optarg
- [ ] Test: `.TableCreate("name")` with `primary_key` optarg
- [ ] Test: `.OrderBy()` with `index` optarg
- [ ] Implement: optarg support on all term methods that need it

### Task 3: Full query serialization

- [ ] Test: wrap term in START query -> `[1,<term>,<optargs>]`
- [ ] Test: query with `db` optarg -> db value wrapped as DB term
- [ ] Test: CONTINUE query -> `[2]`
- [ ] Test: STOP query -> `[3]`
- [ ] Implement: `BuildQuery(queryType, term, opts) []byte`
