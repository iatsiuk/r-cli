# Plan: Output Formatting

## Overview

Result formatters accepting a `RowIterator` interface for streaming output without buffering entire result sets. Formats: JSON, JSONL, Raw, Table. Auto-detection of TTY for default format.

Package: `internal/output`

## Validation Commands
- `go test ./internal/output/... -race -count=1`
- `make build`

### Task 1: JSON output

- [x] Test: format single document as pretty JSON
- [x] Test: format array of documents as streaming JSON array
- [x] Test: format empty result
- [x] Implement: `JSON(w io.Writer, iter RowIterator) error`

### Task 2: Raw and JSONL output

- [x] Test: format single value as plain string
- [x] Test: format each row on separate line (streaming)
- [x] Implement: `Raw(w io.Writer, iter RowIterator) error`
- [x] Test: format single document as compact single-line JSON
- [x] Test: format sequence as one JSON object per line (no wrapping array)
- [x] Test: format streaming (changefeed) output as continuous JSONL
- [x] Implement: `JSONL(w io.Writer, iter RowIterator) error`

### Task 3: Table output

- [ ] Test: format array of objects as aligned ASCII table
- [ ] Test: handle missing fields (fill with empty)
- [ ] Test: truncate long values
- [ ] Test: handle non-object results (fallback to raw)
- [ ] Test: rows exceeding maxTableRows (10000) -> truncate with warning to stderr
- [ ] Implement: `Table(w io.Writer, iter RowIterator) error` (buffers up to maxTableRows=10000)

### Task 4: Non-TTY detection and auto-format

- [ ] Test: isatty(stdout) true -> default to pretty JSON
- [ ] Test: isatty(stdout) false -> default to JSONL
- [ ] Test: explicit --format flag overrides auto-detection
- [ ] Test: NO_COLOR env var disables colored output
- [ ] Implement: `DetectFormat(stdout *os.File, flagFormat string) string`
