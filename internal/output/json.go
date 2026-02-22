package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// JSON formats results as pretty-printed JSON.
// A single row is printed directly; multiple rows are wrapped in an array.
// Empty results print as [].
func JSON(w io.Writer, iter RowIterator) error {
	first, err := iter.Next()
	if errors.Is(err, io.EOF) {
		_, err = fmt.Fprintln(w, "[]")
		return err
	}
	if err != nil {
		return err
	}

	second, err := iter.Next()
	if errors.Is(err, io.EOF) {
		return writeIndented(w, first)
	}
	if err != nil {
		return err
	}

	return writeJSONArray(w, first, second, iter)
}

func writeIndented(w io.Writer, data json.RawMessage) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		buf.Reset()
		buf.Write(data)
	}
	_, err := fmt.Fprintln(w, buf.String())
	return err
}

func writeJSONArray(w io.Writer, first, second json.RawMessage, iter RowIterator) error {
	if _, err := fmt.Fprintln(w, "["); err != nil {
		return err
	}

	cur := first
	peek := second
	var peekErr error

	for {
		var buf bytes.Buffer
		if err := json.Indent(&buf, cur, "  ", "  "); err != nil {
			buf.Reset()
			buf.Write(cur)
		}

		suffix := ""
		if peekErr == nil {
			suffix = ","
		}

		if _, err := fmt.Fprintf(w, "  %s%s\n", buf.String(), suffix); err != nil {
			return err
		}

		if errors.Is(peekErr, io.EOF) {
			break
		}
		if peekErr != nil {
			return peekErr
		}

		cur = peek
		peek, peekErr = iter.Next()
	}

	_, err := fmt.Fprintln(w, "]")
	return err
}
