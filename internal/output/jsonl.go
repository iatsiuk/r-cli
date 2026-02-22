package output

import (
	"errors"
	"fmt"
	"io"
)

// JSONL formats results as newline-delimited JSON (one compact JSON per line).
func JSONL(w io.Writer, iter RowIterator) error {
	for {
		row, err := iter.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, string(row)); err != nil {
			return err
		}
	}
}
