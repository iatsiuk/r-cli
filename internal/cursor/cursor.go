package cursor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"r-cli/internal/proto"
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
	if c.done || !c.hasItem {
		return nil, nil
	}
	c.done = true
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

// streamCursor handles paginated SUCCESS_PARTIAL responses by sending CONTINUE.
type streamCursor struct {
	ch     <-chan *response.Response
	send   func(qt proto.QueryType) error
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	cond     *sync.Cond
	buf      []json.RawMessage
	pos      int
	partial  bool // last response was PARTIAL; CONTINUE needed when buf exhausted
	done     bool
	err      error
	fetching bool

	closeOnce sync.Once
	stopErr   error
}

// NewStream creates a streaming cursor for SUCCESS_PARTIAL responses.
// initial is the first response; ch receives subsequent batches.
// send transmits CONTINUE or STOP queries back to the server.
func NewStream(ctx context.Context, initial *response.Response, ch <-chan *response.Response, send func(proto.QueryType) error) Cursor {
	ctx2, cancel := context.WithCancel(ctx)
	c := &streamCursor{
		ch:     ch,
		send:   send,
		ctx:    ctx2,
		cancel: cancel,
		buf:    initial.Results,
		pos:    0,
	}
	c.cond = sync.NewCond(&c.mu)
	switch initial.Type {
	case proto.ResponseSuccessSequence:
		c.done = true
	case proto.ResponseSuccessPartial:
		c.partial = true
	}
	return c
}

func (c *streamCursor) Next() (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		if c.err != nil {
			return nil, c.err
		}
		if c.pos < len(c.buf) {
			item := c.buf[c.pos]
			c.pos++
			return item, nil
		}
		if c.done {
			return nil, io.EOF
		}
		if c.fetching {
			c.cond.Wait()
			continue
		}
		if err := c.fetchBatch(); err != nil {
			return nil, err
		}
	}
}

// fetchBatch is called with mu held; it releases and reacquires mu around I/O.
func (c *streamCursor) fetchBatch() error {
	c.fetching = true
	needContinue := c.partial
	c.mu.Unlock()

	var fetchErr error
	if needContinue {
		fetchErr = c.send(proto.QueryContinue)
	}
	var resp *response.Response
	if fetchErr == nil {
		resp, fetchErr = c.waitForResponse()
	}

	c.mu.Lock()
	c.fetching = false

	if fetchErr != nil {
		c.err = fetchErr
		c.cond.Broadcast()
		return fetchErr
	}

	c.buf = resp.Results
	c.pos = 0
	c.partial = false

	switch {
	case resp.Type == proto.ResponseSuccessSequence:
		c.done = true
	case resp.Type == proto.ResponseSuccessPartial:
		c.partial = true
	case resp.Type.IsError():
		c.err = response.MapError(resp)
	default:
		c.err = fmt.Errorf("cursor: unexpected response type %d", resp.Type)
	}
	c.cond.Broadcast()
	return c.err
}

func (c *streamCursor) waitForResponse() (*response.Response, error) {
	select {
	case resp, ok := <-c.ch:
		if !ok {
			return nil, io.EOF
		}
		return resp, nil
	case <-c.ctx.Done():
		// send STOP exactly once (guards against concurrent Close())
		c.closeOnce.Do(func() {
			c.stopErr = c.send(proto.QueryStop)
		})
		return nil, c.ctx.Err()
	}
}

func (c *streamCursor) All() ([]json.RawMessage, error) {
	var all []json.RawMessage
	for {
		item, err := c.Next()
		if errors.Is(err, io.EOF) {
			return all, nil
		}
		if err != nil {
			return all, err
		}
		all = append(all, item)
	}
}

func (c *streamCursor) Close() error {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		needStop := !c.done && c.err == nil
		c.mu.Unlock()
		c.cancel()
		if needStop {
			c.stopErr = c.send(proto.QueryStop)
		}
	})
	return c.stopErr
}

// changefeedCursor handles infinite SUCCESS_PARTIAL streams (changefeeds).
// It never auto-completes; only Close() or a connection drop terminates it.
type changefeedCursor struct {
	ch     <-chan *response.Response
	send   func(qt proto.QueryType) error
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	cond     *sync.Cond
	buf      []json.RawMessage
	pos      int
	err      error
	fetching bool

	closeOnce sync.Once
	stopErr   error
}

// NewChangefeed creates a cursor for infinite changefeed streams.
// It always sends CONTINUE after each batch and never terminates automatically.
func NewChangefeed(ctx context.Context, initial *response.Response, ch <-chan *response.Response, send func(proto.QueryType) error) Cursor {
	ctx2, cancel := context.WithCancel(ctx)
	c := &changefeedCursor{
		ch:     ch,
		send:   send,
		ctx:    ctx2,
		cancel: cancel,
		buf:    initial.Results,
		pos:    0,
	}
	c.cond = sync.NewCond(&c.mu)
	return c
}

func (c *changefeedCursor) Next() (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for {
		if c.err != nil {
			return nil, c.err
		}
		if c.pos < len(c.buf) {
			item := c.buf[c.pos]
			c.pos++
			return item, nil
		}
		if c.fetching {
			c.cond.Wait()
			continue
		}
		if err := c.fetchNextBatch(); err != nil {
			return nil, err
		}
	}
}

// fetchNextBatch is called with mu held; releases and reacquires mu around I/O.
func (c *changefeedCursor) fetchNextBatch() error {
	c.fetching = true
	c.mu.Unlock()

	fetchErr := c.send(proto.QueryContinue)
	var resp *response.Response
	if fetchErr == nil {
		resp, fetchErr = c.waitForChangefeedResponse()
	}

	c.mu.Lock()
	c.fetching = false

	if fetchErr != nil {
		c.err = fetchErr
		c.cond.Broadcast()
		return fetchErr
	}

	c.buf = resp.Results
	c.pos = 0

	if resp.Type.IsError() {
		c.err = response.MapError(resp)
	} else if resp.Type != proto.ResponseSuccessPartial {
		c.err = fmt.Errorf("cursor: unexpected response type %d", resp.Type)
	}
	c.cond.Broadcast()
	return c.err
}

func (c *changefeedCursor) waitForChangefeedResponse() (*response.Response, error) {
	select {
	case resp, ok := <-c.ch:
		if !ok {
			return nil, fmt.Errorf("cursor: connection closed")
		}
		return resp, nil
	case <-c.ctx.Done():
		c.closeOnce.Do(func() {
			c.stopErr = c.send(proto.QueryStop)
		})
		return nil, c.ctx.Err()
	}
}

func (c *changefeedCursor) All() ([]json.RawMessage, error) {
	return nil, fmt.Errorf("cursor: All() not supported for changefeed; use Next()")
}

func (c *changefeedCursor) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		c.stopErr = c.send(proto.QueryStop)
	})
	return c.stopErr
}
