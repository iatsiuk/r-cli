# Add Environment Variables Section to Help Output

## Overview
- Add a dedicated "Environment Variables" section to `r-cli --help` output listing all supported env vars with descriptions
- Remove `(or RETHINKDB_PASSWORD env)` from `--password` flag description since env vars will have their own section
- Remove dead `NoColor()` function and its tests -- never called, no colors in the app
- Environment variables: `RETHINKDB_HOST`, `RETHINKDB_PORT`, `RETHINKDB_USER`, `RETHINKDB_PASSWORD`, `RETHINKDB_DATABASE`

## Context
- File: `cmd/r-cli/root.go` -- root command setup, flag definitions, `resolveEnvVars()`
- Tests: `cmd/r-cli/root_test.go`
- Cobra default usage template does not include env vars; need custom `SetUsageTemplate()`
- `NO_COLOR` exists in `internal/output` but `NoColor()` is never called -- no colors in the app, skip it
- `buildRootCmd()` currently 69/80 lines (funlen limit) -- no room for inline template, need helper function
- Cobra's `SetUsageTemplate()` inherits to subcommands via recursive parent lookup (`command.go:592-601`)
- Cobra's help template calls `{{.UsageString}}` which renders the usage template -- so `SetUsageTemplate()` affects `--help` output
- `cobra.Command.HasParent()` exists (`command.go:1676`) and is available in templates

## Development Approach
- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task**
- Run tests after each change

## Implementation Steps

### Task 1: Write tests for env vars section in help output
- [x] add table-driven `TestHelpEnvVarsSection` test in `root_test.go` with cases:
  - root `--help` -- output MUST contain "Environment Variables:" header and all 5 env var names
  - subcommand `db --help` -- output MUST NOT contain "Environment Variables:"
- [x] add `TestPasswordFlagUsageNoEnvMention` test asserting `--password` flag's `Usage` field does not contain `RETHINKDB_PASSWORD`
- [x] run tests -- expected to FAIL (tests written before implementation)

### Task 2: Implement env vars section in help output
- [ ] remove `(or RETHINKDB_PASSWORD env)` from `--password` flag description in `root.go`
- [ ] add helper function (e.g. `envVarsUsageTemplate`) that gets default usage template via `cmd.UsageTemplate()` and injects env vars section wrapped in `{{if not .HasParent}}...{{end}}` before the trailing "Use ... --help" line via `strings.Replace`
- [ ] call helper from `buildRootCmd()` and apply result via `cmd.SetUsageTemplate()`
- [ ] env vars section content:
  - `RETHINKDB_HOST` -- override default host
  - `RETHINKDB_PORT` -- override default port
  - `RETHINKDB_USER` -- override default user
  - `RETHINKDB_PASSWORD` -- set password
  - `RETHINKDB_DATABASE` -- set default database
- [ ] run tests -- must pass

### Task 3: Remove dead NoColor code
- [ ] remove `NoColor()` function from `internal/output/detect.go`
- [ ] remove `TestNoColor` and `TestNoColorUnset` from `internal/output/detect_test.go`
- [ ] run tests (`go test ./internal/output/... -race -count=1`) -- must pass

### Task 4: Verify acceptance criteria
- [ ] run full test suite (`go test ./cmd/r-cli/... -race -count=1`)
- [ ] run linter (`golangci-lint run`)
- [ ] build (`make build`)
- [ ] manually verify `go run ./cmd/r-cli --help` shows env vars section and clean `--password` description
- [ ] manually verify `go run ./cmd/r-cli db --help` does NOT show env vars section

## Technical Details
- Get default template via `cmd.UsageTemplate()`, inject env section via `strings.Replace` -- avoids hardcoding full cobra template
- Wrap env section in `{{if not .HasParent}}...{{end}}` so subcommands don't render it
- Include preceding blank line inside the `{{if}}` block to avoid extra whitespace in subcommand output
- Extract template logic into helper function to keep `buildRootCmd()` under 80-line funlen limit
