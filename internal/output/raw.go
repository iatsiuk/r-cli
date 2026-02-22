package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Raw formats results as plain strings.
// String values are unquoted; other types are printed as compact JSON.
func Raw(w io.Writer, iter RowIterator) error {
	for {
		row, err := iter.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		var s string
		if jsonErr := json.Unmarshal(row, &s); jsonErr == nil {
			_, err = fmt.Fprintln(w, s)
		} else {
			_, err = fmt.Fprintln(w, string(row))
		}
		if err != nil {
			return err
		}
	}
}
