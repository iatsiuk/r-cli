package query

import (
	"context"
	"encoding/json"
	"fmt"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/cursor"
	"r-cli/internal/proto"
	"r-cli/internal/reql"
	"r-cli/internal/response"
)

// Executor executes ReQL queries via a managed connection.
type Executor struct {
	mgr *connmgr.ConnManager
}

// New creates an Executor backed by the given connection manager.
func New(mgr *connmgr.ConnManager) *Executor {
	return &Executor{mgr: mgr}
}

// Run executes a ReQL term and returns a cursor over the results.
// If opts contains "noreply": true, the query is sent without waiting for a
// response and Run returns (nil, nil).
func (e *Executor) Run(ctx context.Context, term reql.Term, opts reql.OptArgs) (cursor.Cursor, error) {
	c, err := e.mgr.Get(ctx)
	if err != nil {
		return nil, err
	}
	if opts == nil {
		opts = reql.OptArgs{}
	}
	payload, err := reql.BuildQuery(proto.QueryStart, term, opts)
	if err != nil {
		return nil, fmt.Errorf("query: build: %w", err)
	}
	if v, _ := opts["noreply"].(bool); v {
		return nil, c.WriteFrame(c.NextToken(), payload)
	}
	token := c.NextToken()
	raw, err := c.Send(ctx, token, payload)
	if err != nil {
		return nil, fmt.Errorf("query: send: %w", err)
	}
	resp, err := response.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("query: response: %w", err)
	}
	if err := response.MapError(resp); err != nil {
		return nil, err
	}
	return makeCursor(ctx, c, token, resp)
}

// makeCursor selects the appropriate cursor type for the response.
func makeCursor(ctx context.Context, c *conn.Conn, token uint64, resp *response.Response) (cursor.Cursor, error) {
	switch resp.Type {
	case proto.ResponseSuccessAtom:
		return cursor.NewAtom(resp), nil
	case proto.ResponseSuccessSequence:
		return cursor.NewSequence(resp), nil
	case proto.ResponseSuccessPartial:
		ch := make(chan *response.Response, 1)
		send := makeSend(ctx, c, token, ch)
		if isFeed(resp) {
			return cursor.NewChangefeed(ctx, resp, ch, send), nil
		}
		return cursor.NewStream(ctx, resp, ch, send), nil
	default:
		return nil, fmt.Errorf("query: unexpected response type %d", resp.Type)
	}
}

// makeSend builds the send function for streaming cursors.
// CONTINUE spawns a goroutine to fetch the next batch asynchronously.
// STOP writes the stop frame without waiting for a response.
func makeSend(ctx context.Context, c *conn.Conn, token uint64, ch chan<- *response.Response) func(proto.QueryType) error {
	return func(qt proto.QueryType) error {
		switch qt {
		case proto.QueryContinue:
			go continueStream(ctx, c, token, ch)
			return nil
		case proto.QueryStop:
			return c.WriteFrame(token, []byte(`[3]`))
		default:
			return fmt.Errorf("query: unsupported query type %d", qt)
		}
	}
}

// continueStream sends CONTINUE and delivers the next response batch on ch.
func continueStream(ctx context.Context, c *conn.Conn, token uint64, ch chan<- *response.Response) {
	raw, err := c.Send(ctx, token, []byte(`[2]`))
	if err != nil {
		select {
		case ch <- errResp(err):
		default:
		}
		return
	}
	resp, err := response.Parse(raw)
	if err != nil {
		select {
		case ch <- errResp(err):
		default:
		}
		return
	}
	select {
	case ch <- resp:
	default:
	}
}

// isFeed reports whether the response carries a changefeed note.
func isFeed(resp *response.Response) bool {
	for _, n := range resp.Notes {
		switch n {
		case proto.NoteSequenceFeed, proto.NoteAtomFeed,
			proto.NoteOrderByLimitFeed, proto.NoteUnionedFeed:
			return true
		}
	}
	return false
}

// ServerInfo holds information about the connected RethinkDB server.
type ServerInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ServerInfo returns information about the connected server.
func (e *Executor) ServerInfo(ctx context.Context) (*ServerInfo, error) {
	c, err := e.mgr.Get(ctx)
	if err != nil {
		return nil, err
	}
	token := c.NextToken()
	raw, err := c.Send(ctx, token, []byte(`[5]`))
	if err != nil {
		return nil, fmt.Errorf("query: server info: %w", err)
	}
	resp, err := response.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("query: server info response: %w", err)
	}
	if resp.Type != proto.ResponseServerInfo {
		return nil, fmt.Errorf("query: unexpected response type %d", resp.Type)
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("query: empty server info response")
	}
	var info ServerInfo
	if err := json.Unmarshal(resp.Results[0], &info); err != nil {
		return nil, fmt.Errorf("query: parse server info: %w", err)
	}
	return &info, nil
}

// errResp wraps a transport error into a CLIENT_ERROR response so streaming
// cursors can surface it through the normal response channel.
func errResp(err error) *response.Response {
	msg, _ := json.Marshal(err.Error())
	return &response.Response{
		Type:    proto.ResponseClientError,
		Results: []json.RawMessage{msg},
	}
}
