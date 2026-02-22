package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const (
	maxTableRows = 10000
	maxColWidth  = 50
)

// Table formats results as an aligned ASCII table.
// Buffers up to maxTableRows rows; if exceeded, truncates with warning to stderr.
// Non-object rows fall back to raw output.
func Table(w io.Writer, iter RowIterator) error {
	return tableWriter(w, os.Stderr, iter, maxTableRows)
}

func tableWriter(w, errOut io.Writer, iter RowIterator, maxRows int) error {
	rows, truncated, err := collectRows(iter, maxRows)
	if err != nil {
		return err
	}

	if truncated {
		_, _ = fmt.Fprintf(errOut, "warning: result truncated at %d rows\n", maxRows)
	}

	if len(rows) == 0 {
		return nil
	}

	var probe map[string]json.RawMessage
	if json.Unmarshal(rows[0], &probe) != nil {
		return rawSlice(w, rows)
	}

	cols := extractColumns(rows)
	widths := computeWidths(cols, rows)

	if err := printTableHeader(w, cols, widths); err != nil {
		return err
	}
	for _, row := range rows {
		if err := printTableRow(w, cols, widths, row); err != nil {
			return err
		}
	}
	return nil
}

func drainIter(iter RowIterator) {
	for {
		if _, err := iter.Next(); err != nil {
			return
		}
	}
}

func collectRows(iter RowIterator, maxRows int) ([]json.RawMessage, bool, error) {
	var rows []json.RawMessage
	for {
		row, err := iter.Next()
		if errors.Is(err, io.EOF) {
			return rows, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		if len(rows) >= maxRows {
			drainIter(iter)
			return rows, true, nil
		}
		rows = append(rows, row)
	}
}

func extractColumns(rows []json.RawMessage) []string {
	seen := map[string]bool{}
	var cols []string
	for _, row := range rows {
		keys, err := objectKeysInOrder(row)
		if err != nil {
			continue
		}
		for _, k := range keys {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	return cols
}

func objectKeysInOrder(data json.RawMessage) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, fmt.Errorf("not an object")
	}
	var keys []string
	for dec.More() {
		tok, err = dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected token type")
		}
		keys = append(keys, key)
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func computeWidths(cols []string, rows []json.RawMessage) []int {
	widths := make([]int, len(cols))
	for i, col := range cols {
		widths[i] = utf8.RuneCountInString(col)
	}
	for _, row := range rows {
		var obj map[string]json.RawMessage
		if json.Unmarshal(row, &obj) != nil {
			continue
		}
		for i, col := range cols {
			v := cellValue(obj[col])
			if n := utf8.RuneCountInString(v); n > widths[i] {
				widths[i] = n
			}
		}
	}
	for i := range widths {
		if widths[i] > maxColWidth {
			widths[i] = maxColWidth
		}
	}
	return widths
}

func cellValue(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return string(raw)
}

func printTableHeader(w io.Writer, cols []string, widths []int) error {
	parts := make([]string, len(cols))
	for i, col := range cols {
		parts[i] = padRight(col, widths[i])
	}
	if _, err := fmt.Fprintln(w, strings.Join(parts, " | ")); err != nil {
		return err
	}
	seps := make([]string, len(cols))
	for i, width := range widths {
		seps[i] = strings.Repeat("-", width)
	}
	_, err := fmt.Fprintln(w, strings.Join(seps, "-+-"))
	return err
}

func printTableRow(w io.Writer, cols []string, widths []int, row json.RawMessage) error {
	var obj map[string]json.RawMessage
	if json.Unmarshal(row, &obj) != nil {
		_, err := fmt.Fprintln(w, string(row))
		return err
	}
	parts := make([]string, len(cols))
	for i, col := range cols {
		v := cellValue(obj[col])
		if runes := []rune(v); widths[i] > 0 && len(runes) > widths[i] {
			v = string(runes[:widths[i]-1]) + "~"
		}
		parts[i] = padRight(v, widths[i])
	}
	_, err := fmt.Fprintln(w, strings.Join(parts, " | "))
	return err
}

func padRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func rawSlice(w io.Writer, rows []json.RawMessage) error {
	for _, row := range rows {
		var s string
		if json.Unmarshal(row, &s) == nil {
			if _, err := fmt.Fprintln(w, s); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w, string(row)); err != nil {
				return err
			}
		}
	}
	return nil
}
