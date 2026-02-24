# r-cli

RethinkDB command-line client. Execute ReQL queries, manage databases, tables, indexes, users, and permissions -- all from the terminal. Includes an interactive REPL with tab completion and multiline input.

## Installation

```bash
# from source
go install r-cli/cmd/r-cli@latest

# from releases (macOS Apple Silicon example)
curl -L https://github.com/iatsiuk/r-cli/releases/latest/download/r-cli_VERSION_darwin_arm64.tar.gz | tar xz
chmod +x r-cli && mv r-cli /usr/local/bin/
```

## Quick Usage

```bash
# interactive REPL (starts automatically when no args given)
r-cli

# execute a ReQL expression
r-cli 'r.db("test").tableList()'

# pipe-friendly
echo 'r.db("test").table("users").count()' | r-cli

# database management
r-cli db list
r-cli db create mydb
r-cli db drop mydb --yes

# table management
r-cli table list --db mydb
r-cli table create users --db mydb

# query with output format
r-cli -d mydb -f table 'r.table("users")'

# bulk insert from JSONL
cat data.jsonl | r-cli insert mydb.users
r-cli insert mydb.users -F data.json

# server status
r-cli status

# connect to remote host with auth
r-cli -H db.example.com -P 28015 -u admin -p secret 'r.dbList()'

# TLS connection
r-cli --tls-cert ca.pem -H db.example.com 'r.dbList()'
```

## Commands

| Command | Description |
|---------|-------------|
| *(default)* | Execute expression (arg or stdin) or start REPL on TTY |
| `query [expr]` | Execute a ReQL expression |
| `run [term]` | Execute a raw ReQL JSON term |
| `repl` | Start interactive REPL |
| `db list\|create\|drop` | Database management |
| `table list\|create\|drop\|info\|reconfigure\|rebalance\|wait\|sync` | Table management (requires `--db`) |
| `index list\|create\|drop\|rename\|status\|wait` | Index management (requires `--db`) |
| `user list\|create\|delete\|set-password` | User management |
| `grant <user>` | Grant/revoke permissions |
| `insert <db.table>` | Bulk insert documents |
| `status` | Show server info |
| `completion bash\|zsh\|fish` | Generate shell completions |

### query

```bash
r-cli query 'r.db("test").table("users").filter(r.row("age").gt(21))'

# from file (multiple queries separated by ---)
r-cli query -F queries.reql
r-cli query -F queries.reql --stop-on-error
```

### insert

```bash
# JSONL from stdin (default)
cat users.jsonl | r-cli insert mydb.users

# JSON array from file
r-cli insert mydb.users -F users.json

# options
r-cli insert mydb.users -F data.jsonl --batch-size 500 --conflict replace
```

Conflict strategies: `error` (default), `replace`, `update`.

### grant

```bash
# global permission
r-cli grant alice --read

# database-level
r-cli grant alice --read --write --db mydb

# table-level
r-cli grant alice --read --db mydb --table users

# revoke
r-cli grant alice --read=false --db mydb
```

### table reconfigure

```bash
r-cli table reconfigure users --db mydb --shards 2 --replicas 3
r-cli table reconfigure users --db mydb --dry-run
```

### index create

```bash
r-cli index create users email --db mydb
r-cli index create users location --db mydb --geo
r-cli index create users tags --db mydb --multi
```

## Interactive REPL

When invoked with no arguments on a TTY, r-cli starts an interactive REPL:

```
r> r.dbList()
["test", "mydb"]
r> .use mydb
r> r.tableList()
["users", "posts"]
r> r.table("users").count()
42
```

Features:
- Tab completion for databases, tables, and ReQL methods
- Multiline input (auto-detected by unbalanced brackets/parens)
- History saved to `~/.r-cli_history`

Dot-commands:
- `.use <db>` -- switch default database
- `.format <fmt>` -- switch output format (json, jsonl, raw, table)
- `.help` -- list commands
- `.exit` / `.quit` -- exit REPL

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--host` | `-H` | localhost | RethinkDB host |
| `--port` | `-P` | 28015 | RethinkDB port |
| `--db` | `-d` | | Default database |
| `--user` | `-u` | admin | RethinkDB user |
| `--password` | `-p` | | Password |
| `--password-file` | | | Read password from file |
| `--timeout` | `-t` | 30s | Connection timeout |
| `--format` | `-f` | *(auto)* | Output: json, jsonl, raw, table |
| `--profile` | | false | Enable query profiling |
| `--time-format` | | native | `native` converts TIME pseudo-types, `raw` passes through |
| `--binary-format` | | native | `native` converts BINARY pseudo-types, `raw` passes through |
| `--quiet` | | false | Suppress non-data stderr output |
| `--verbose` | | false | Show connection info and query timing |
| `--tls-cert` | | | CA certificate PEM file |
| `--tls-client-cert` | | | Client certificate PEM file |
| `--tls-key` | | | Client private key PEM file |
| `--insecure-skip-verify` | | false | Skip TLS certificate verification |

## Output Formats

Format is auto-detected: `json` (pretty-printed) on TTY, `jsonl` (one JSON per line) when piped. Override with `-f`:

- **json** -- pretty-printed JSON; single value as-is, multiple values wrapped in an array
- **jsonl** -- one compact JSON document per line
- **raw** -- strings unquoted, other values as compact JSON
- **table** -- aligned ASCII table (for object results)

## Environment Variables

| Variable | Overrides |
|----------|-----------|
| `RETHINKDB_HOST` | `--host` |
| `RETHINKDB_PORT` | `--port` |
| `RETHINKDB_USER` | `--user` |
| `RETHINKDB_PASSWORD` | `--password` |
| `RETHINKDB_DATABASE` | `--db` |
| `NO_COLOR` | Disables colored output |

CLI flags always take precedence over environment variables.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Connection error |
| 2 | Query error |
| 3 | Authentication error |
| 130 | Interrupted (SIGINT/SIGTERM) |
