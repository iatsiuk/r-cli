# Unsupported RethinkDB JS API Commands

Commands from the RethinkDB JavaScript API that are not implemented in r-cli and the reasons why.

## Connection and Driver Management

r-cli is a CLI tool, not a driver library. Connection lifecycle is managed internally.

- `r` - top-level namespace / driver import
- `connect` - programmatic connection creation
- `close` - programmatic connection close
- `reconnect` - programmatic reconnect
- `use` - change default database on connection object (r-cli uses `--db` flag and `.use` REPL command instead)
- `run` - execute query on connection object (r-cli executes queries implicitly)
- `noreplyWait` - wait for noreply queries to complete on a connection
- `server` - connection server info (r-cli has `status` subcommand instead)
- `EventEmitter (connection)` - Node.js event-driven connection state handling

## Cursor API

r-cli handles cursor iteration internally and outputs results in the requested format (json, jsonl, table, raw).

- `next` - get next cursor element
- `each` - iterate cursor elements with callback
- `eachAsync` - async cursor iteration with promises
- `toArray` - collect cursor into array
- `close (cursor)` - close cursor and free resources
- `EventEmitter (cursor)` - Node.js event-driven cursor data handling

## Server-Side JavaScript

- `js` - execute arbitrary JavaScript on the RethinkDB server; security risk, deprecated in practice

## Server-Side HTTP

- `http` - make HTTP requests from the RethinkDB server; not a query operation, limited use in CLI context

## Write Hooks

- `setWriteHook` - set a server-side write hook function on a table
- `getWriteHook` - get the current write hook of a table
