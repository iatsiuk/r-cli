# Plan: ReQL Functions, Index Operations and Changefeeds

## Overview

Implement function serialization (FUNC/VAR), IMPLICIT_VAR auto-wrapping, FUNCALL argument reordering, index operations, and changefeed/misc terms.

Package: `internal/reql`

Depends on: `04-reql-core`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Index operations

- [x] Test: `.IndexCreate("name")` -> `[75,[<table_term>,"name"]]`
- [x] Test: `.IndexDrop("name")` -> `[76,[<table_term>,"name"]]`
- [x] Test: `.IndexList()` -> `[77,[<table_term>]]`
- [x] Test: `.IndexWait("name")` -> `[140,[<table_term>,"name"]]`
- [x] Test: `.IndexStatus("name")` -> `[139,[<table_term>,"name"]]`
- [x] Test: `.IndexRename("old","new")` -> `[156,[<table_term>,"old","new"]]`
- [x] Implement: index operation methods

### Task 2: Changefeed and misc terms

- [x] Test: `.Changes()` -> `[152,[<term>]]`
- [x] Test: `.Changes()` with optarg `include_initial=true`
- [x] Test: `Now()` -> `[103,[]]`
- [x] Test: `UUID()` -> `[169,[]]`
- [x] Test: `Binary(data)` -> `[155,[<data>]]`
- [x] Test: `.Config()` -> `[174,[<term>]]`
- [x] Test: `.Status()` -> `[175,[<term>]]`
- [x] Test: `Grant("user", perms)` -> `[188,[<scope>,"user",<perms>]]`
- [x] Implement: changefeed, time, binary, admin term methods

### Task 3: Function serialization

- [x] Test: single-arg function -> `[69,[[2,[1]],<body>]]`
- [x] Test: multi-arg function -> correct param IDs
- [x] Test: VAR reference -> `[10,[<id>]]`
- [x] Implement: `Func` builder with VAR references

### Task 4: IMPLICIT_VAR auto-wrapping

The driver must detect IMPLICIT_VAR (term 13) in term arguments, replace it with VAR(1), and wrap the argument in FUNC(69). See docs/protocol-spec.md section 6.

- [x] Test: term containing `[13,[]]` is wrapped -> `[69,[[2,[1]],<body_with_var_1>]]`
- [x] Test: nested IMPLICIT_VAR in deeply nested term -> correctly replaced at all levels
- [x] Test: IMPLICIT_VAR in nested function context -> error (ambiguous per spec)
- [x] Test: term without IMPLICIT_VAR -> no wrapping applied
- [x] Implement: `wrapImplicitVar(term Term) Term` tree traversal

### Task 5: FUNCALL (r.do) argument reordering

API order: `Do(arg1, arg2, func)`. Wire order: `[64, [func, arg1, arg2]]`. Function goes first on the wire. See docs/protocol-spec.md section 7.

- [x] Test: `Do(10, 20, func)` -> `[64,[<func>,10,20]]`
- [x] Test: `Do(func)` with no extra args -> `[64,[<func>]]`
- [x] Implement: `Do` builder with argument reordering
