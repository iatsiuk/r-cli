# Plan: Query Language Parser

## Overview

Parse human-readable ReQL string into term tree. Lexer + recursive descent parser. Supports chained method calls, `r.row`, comparisons, nested `r.*` calls, literals, objects, arrays. No JS lambdas or infix operators.

Package: `internal/reql/parser`

Depends on: `04-reql-core`, `06-reql-functions-index`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Lexer

- [x] Test: tokenize `r.db("test")` -> [IDENT:r, DOT, IDENT:db, LPAREN, STRING:"test", RPAREN]
- [x] Test: tokenize numbers, bools, null
- [x] Test: tokenize object literals `{name: "foo", age: 42}`
- [x] Test: tokenize array literals `[1, 2, 3]`
- [x] Test: tokenize chained methods `.table("x").filter({...})`
- [x] Test: tokenize single-quoted strings `'foo'` (in addition to double-quoted)
- [x] Test: tokenize `r.minval` / `r.maxval` as IDENT (no parens)
- [x] Implement: lexer producing token stream

### Task 2: Parser - basic expressions

- [x] Test: parse `r.db("test")` -> DB("test") term
- [x] Test: parse `r.db("test").table("users")` -> chained terms
- [x] Test: parse `.filter({name: "foo"})` -> FILTER with object arg
- [x] Test: parse `.get("id")` -> GET term
- [x] Test: parse `.insert({...})` -> INSERT term
- [x] Test: parse `.orderBy(r.desc("name"))` -> ORDER_BY with DESC
- [x] Test: parse `.limit(10)` -> LIMIT term
- [x] Test: parse `r.row("field").gt(21)` -> IMPLICIT_VAR with GT comparison
- [x] Test: parse nested `r.row` in filter -> auto-wrapped via IMPLICIT_VAR
- [x] Implement: recursive descent parser core

### Task 3: Parser - advanced expressions

- [ ] Test: parse bracket notation `row("field")("subfield")` -> nested BRACKET terms
- [ ] Test: parse `r.expr([1,2,3])` -> MAKE_ARRAY wrapped
- [ ] Test: parse `r.minval` (no parens) -> MINVAL term
- [ ] Test: parse `r.maxval` (no parens) -> MAXVAL term
- [ ] Test: parse `r.branch(cond, trueVal, falseVal)` -> BRANCH term
- [ ] Test: parse `r.error("msg")` -> ERROR term
- [ ] Test: parse `r.args([...])` -> ARGS term
- [ ] Implement: extended expression parsing

### Task 4: Parser - all method names and extended operations

- [ ] Test: parse all new method names -> correct term types (mapping table test)
- [ ] Test: parse `.eqJoin("field", r.table("other"))` -> EQ_JOIN with table arg
- [ ] Test: parse `.match("^foo")` -> MATCH with string arg
- [ ] Test: parse `r.point(-122.4, 37.7)` -> POINT term
- [ ] Test: parse `r.epochTime(1234567890)` -> EPOCH_TIME term
- [ ] Test: parse `.coerceTo("string")` -> COERCE_TO term
- [ ] Test: parse `.default(0)` -> DEFAULT term
- [ ] Implement: complete method name -> term type mapping

### Task 5: Error handling and fuzz testing

- [ ] Test: syntax error -> descriptive error with position
- [ ] Test: deeply nested expression (depth > 256) -> error (prevent stack overflow)
- [ ] Implement: maxDepth=256 guard in parser
- [ ] Fuzz: lexer does not panic on arbitrary input
- [ ] Fuzz: parser does not panic on arbitrary token sequences
- [ ] Implement: `func FuzzParse(f *testing.F)` with seed corpus from test cases
