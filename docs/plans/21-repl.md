# Plan: Interactive REPL

## Overview

Interactive REPL with readline, history, multiline input, tab completion, and dot-commands. Uses `github.com/chzyer/readline`.

Package: `internal/repl`

Depends on: `14-parser`, `12-cli-core`

## Validation Commands
- `go test ./internal/repl/... ./cmd/... -race -count=1`
- `make build`

### Task 1: Basic REPL loop

- [x] Test: start REPL, send query, receive output, prompt reappears
- [x] Test: empty input (just Enter) -> no query executed, new prompt
- [x] Test: Ctrl+D (EOF) -> clean exit
- [x] Test: Ctrl+C during input -> cancel current line, new prompt
- [x] Test: Ctrl+C during query execution -> cancel query (send STOP), new prompt
- [x] Implement: REPL loop with github.com/chzyer/readline

### Task 2: History

- [x] Test: query is saved to history file (~/.r-cli_history)
- [x] Test: up/down arrows navigate history
- [x] Test: history persists between sessions
- [x] Implement: readline history integration

### Task 3: Multiline input

- [x] Test: unclosed parenthesis -> continuation prompt, wait for closing
- [x] Test: unclosed brace -> continuation prompt
- [x] Test: complete multiline query executes correctly
- [x] Implement: paren/brace/bracket counting for continuation detection

### Task 4: Tab completion

- [x] Test: `r.` + TAB -> list top-level r.* methods
- [x] Test: `.` + TAB after table -> list chainable methods
- [x] Test: `r.db("` + TAB -> list database names (query server)
- [x] Test: `.table("` + TAB -> list table names in current db (query server)
- [x] Implement: completer with static methods + dynamic db/table names

### Task 5: REPL-specific commands

- [ ] Test: `.exit` or `.quit` -> exit REPL
- [ ] Test: `.use <db>` -> change default database
- [ ] Test: `.format <fmt>` -> change output format for session
- [ ] Test: `.help` -> show available commands
- [ ] Implement: dot-command dispatcher

### Task 6: CLI integration

- [ ] Test: `r-cli` (no args, TTY) -> start REPL
- [ ] Test: `r-cli` (no args, not TTY, stdin has data) -> read from stdin
- [ ] Test: `r-cli repl` -> force REPL mode
- [ ] Test: REPL respects --host/--port/--db/--user flags
- [ ] Implement: REPL command in cobra, auto-detect mode
