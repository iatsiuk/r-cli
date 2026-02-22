package output

import "encoding/json"

// RowIterator streams rows from a query result.
type RowIterator interface {
	Next() (json.RawMessage, error)
	Close() error
}
