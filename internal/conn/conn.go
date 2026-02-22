package conn

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"

	"r-cli/internal/wire"
)

// ErrClosed is returned by Send when the connection is closed.
var ErrClosed = errors.New("conn: connection closed")

// stopPayload is the wire payload for a STOP query (proto.QueryStop = 3).
var stopPayload = []byte(`[3]`)

// result carries the outcome of a dispatched response.
type result struct {
	payload []byte
	err     error
}

// Config holds connection parameters.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"-"`
}

// String returns Config without the password.
func (c Config) String() string {
	return fmt.Sprintf("conn{%s:%d user=%s}", c.Host, c.Port, c.User)
}

// Conn manages a single RethinkDB connection with multiplexed query dispatch.
// A background readLoop goroutine dispatches responses to Send callers by token.
type Conn struct {
	token   atomic.Uint64
	nc      net.Conn
	mu      sync.Mutex
	waiters map[uint64]chan result
	writeMu sync.Mutex
	closed  bool
	done    chan struct{}
	debug   bool
}

// Dial connects to addr, performs the V1_0 handshake, and starts the readLoop.
// tlsCfg may be nil for a plain TCP connection.
func Dial(ctx context.Context, addr string, cfg Config, tlsCfg *tls.Config) (*Conn, error) {
	nc, err := dialNet(ctx, addr, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	// run handshake in a goroutine to respect context cancellation
	type hsResult struct{ err error }
	hsC := make(chan hsResult, 1)
	go func() {
		hsC <- hsResult{err: Handshake(nc, cfg.User, cfg.Password)}
	}()

	select {
	case <-ctx.Done():
		_ = nc.Close()
		<-hsC // drain to avoid goroutine leak
		return nil, fmt.Errorf("dial %s: %w", addr, ctx.Err())
	case res := <-hsC:
		if res.err != nil {
			_ = nc.Close()
			return nil, fmt.Errorf("dial %s: %w", addr, res.err)
		}
	}
	return newConn(nc), nil
}

// dialNet establishes a TCP or TLS connection.
func dialNet(ctx context.Context, addr string, tlsCfg *tls.Config) (net.Conn, error) {
	d := &net.Dialer{}
	if tlsCfg != nil {
		td := tls.Dialer{NetDialer: d, Config: tlsCfg}
		return td.DialContext(ctx, "tcp", addr)
	}
	return d.DialContext(ctx, "tcp", addr)
}

// newConn wraps nc in a Conn and starts the background readLoop.
func newConn(nc net.Conn) *Conn {
	c := &Conn{
		nc:      nc,
		waiters: make(map[uint64]chan result),
		done:    make(chan struct{}),
		debug:   os.Getenv("RCLI_DEBUG") == "wire",
	}
	go c.readLoop()
	return c
}

// IsClosed reports whether the connection is closed.
func (c *Conn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Close closes the underlying connection and waits for all pending Send calls to unblock.
func (c *Conn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	err := c.nc.Close()
	<-c.done // readLoop will notify all pending waiters
	return err
}

// Send registers a waiter for token, writes the query frame, and waits for the response.
// If ctx is cancelled, a STOP frame is sent and the waiter is cleaned up.
func (c *Conn) Send(ctx context.Context, token uint64, payload []byte) ([]byte, error) {
	ch := make(chan result, 1)

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrClosed
	}
	c.waiters[token] = ch
	c.mu.Unlock()

	c.writeMu.Lock()
	if c.debug {
		_, _ = fmt.Fprintf(os.Stderr, "wire out: token=%d len=%d\n%s", token, len(payload), hex.Dump(payload))
	}
	werr := wire.WriteQuery(c.nc, token, payload)
	c.writeMu.Unlock()

	if werr != nil {
		c.mu.Lock()
		delete(c.waiters, token)
		c.mu.Unlock()
		return nil, fmt.Errorf("send: %w", werr)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.waiters, token)
		c.mu.Unlock()
		c.sendStop(token)
		return nil, ctx.Err()
	case res := <-ch:
		return res.payload, res.err
	}
}

// sendStop sends a STOP query for the given token; write errors are silently ignored.
func (c *Conn) sendStop(token uint64) {
	c.writeMu.Lock()
	_ = wire.WriteQuery(c.nc, token, stopPayload)
	c.writeMu.Unlock()
}

// nextToken returns the next unique query token, incrementing atomically.
func (c *Conn) nextToken() uint64 {
	return c.token.Add(1)
}

// NextToken returns the next unique query token for external callers.
func (c *Conn) NextToken() uint64 {
	return c.nextToken()
}

// WriteFrame writes a wire frame to the connection without registering a
// response waiter. Used for noreply queries and STOP frames.
func (c *Conn) WriteFrame(token uint64, payload []byte) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClosed
	}
	c.mu.Unlock()
	c.writeMu.Lock()
	err := wire.WriteQuery(c.nc, token, payload)
	c.writeMu.Unlock()
	return err
}

// readLoop continuously reads wire frames and dispatches them to pending Send callers.
func (c *Conn) readLoop() {
	defer close(c.done)
	for {
		token, payload, err := wire.ReadResponse(c.nc)
		if err != nil {
			// close nc to release the fd when the connection dies unexpectedly
			// (user's Close() won't call nc.Close() once closed=true is set)
			_ = c.nc.Close()
			c.closeWaiters(fmt.Errorf("readLoop: %w", err))
			return
		}
		if c.debug {
			_, _ = fmt.Fprintf(os.Stderr, "wire in: token=%d len=%d\n%s", token, len(payload), hex.Dump(payload))
		}
		c.dispatch(token, payload)
	}
}

// dispatch sends payload to the waiter registered for token, if any.
// Responses for unknown or removed tokens are silently discarded.
func (c *Conn) dispatch(token uint64, payload []byte) {
	c.mu.Lock()
	ch, ok := c.waiters[token]
	if ok {
		delete(c.waiters, token)
	}
	c.mu.Unlock()

	if !ok {
		return
	}
	select {
	case ch <- result{payload: payload}:
	default:
		// buffer already full, discard
	}
}

// closeWaiters delivers err to all pending waiters and marks the connection closed.
func (c *Conn) closeWaiters(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	for token, ch := range c.waiters {
		select {
		case ch <- result{err: err}:
		default:
		}
		delete(c.waiters, token)
	}
}
