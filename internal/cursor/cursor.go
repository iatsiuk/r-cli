package cursor

import (
	"encoding/json"
	"io"

	"r-cli/internal/response"
)

// Cursor iterates over query results.
type Cursor interface {
	Next() (json.RawMessage, error)
	All() ([]json.RawMessage, error)
	Close() error
}

// atomCursor returns a single value from a SUCCESS_ATOM response.
type atomCursor struct {
	item    json.RawMessage
	hasItem bool
	done    bool
}

// NewAtom creates a cursor from a SUCCESS_ATOM response.
func NewAtom(resp *response.Response) Cursor {
	if len(resp.Results) > 0 {
		return &atomCursor{item: resp.Results[0], hasItem: true}
	}
	return &atomCursor{}
}

func (c *atomCursor) Next() (json.RawMessage, error) {
	if c.done || !c.hasItem {
		return nil, io.EOF
	}
	c.done = true
	return c.item, nil
}

func (c *atomCursor) All() ([]json.RawMessage, error) {
	if !c.hasItem {
		return nil, nil
	}
	return []json.RawMessage{c.item}, nil
}

func (c *atomCursor) Close() error { return nil }

// seqCursor iterates over all items in a SUCCESS_SEQUENCE response.
type seqCursor struct {
	items []json.RawMessage
	pos   int
}

// NewSequence creates a cursor from a SUCCESS_SEQUENCE response.
func NewSequence(resp *response.Response) Cursor {
	return &seqCursor{items: resp.Results}
}

func (c *seqCursor) Next() (json.RawMessage, error) {
	if c.pos >= len(c.items) {
		return nil, io.EOF
	}
	item := c.items[c.pos]
	c.pos++
	return item, nil
}

func (c *seqCursor) All() ([]json.RawMessage, error) {
	return c.items, nil
}

func (c *seqCursor) Close() error { return nil }
