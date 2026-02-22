//go:build integration

package integration

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"r-cli/internal/conn"
	"r-cli/internal/proto"
	"r-cli/internal/scram"
)

func TestConnectionSuccess(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := defaultCfg()
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	c, err := conn.Dial(ctx, addr, cfg, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestConnectionServerVersion(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := defaultCfg()
	nc, err := (&net.Dialer{}).DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = nc.Close() }()

	// build step1 (magic) + step3 (client-first-message) pipelined
	conv := scram.NewConversation("admin", "")
	step3Payload, err := json.Marshal(struct {
		ProtocolVersion      int    `json:"protocol_version"`
		AuthenticationMethod string `json:"authentication_method"`
		Authentication       string `json:"authentication"`
	}{0, "SCRAM-SHA-256", conv.ClientFirst()})
	if err != nil {
		t.Fatalf("marshal step3: %v", err)
	}

	magic := make([]byte, 4)
	binary.LittleEndian.PutUint32(magic, uint32(proto.V1_0))

	var pipeline []byte
	pipeline = append(pipeline, magic...)
	pipeline = append(pipeline, step3Payload...)
	pipeline = append(pipeline, 0x00)

	if _, err := nc.Write(pipeline); err != nil {
		t.Fatalf("write pipeline: %v", err)
	}

	// read step2: null-terminated JSON with server_version
	data, err := readNullTerminated(nc)
	if err != nil {
		t.Fatalf("read step2: %v", err)
	}

	var step2 struct {
		Success       bool   `json:"success"`
		ServerVersion string `json:"server_version"`
		Error         string `json:"error"`
	}
	if err := json.Unmarshal(data, &step2); err != nil {
		t.Fatalf("parse step2: %v", err)
	}
	if !step2.Success {
		t.Fatalf("step2 failed: %s", step2.Error)
	}
	if step2.ServerVersion == "" {
		t.Fatal("server_version empty in handshake response")
	}
}

func TestConnectionDialError(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := conn.Config{Host: "127.0.0.1", Port: 1, User: "admin", Password: ""}
	_, err := conn.Dial(ctx, "127.0.0.1:1", cfg, nil)
	if err == nil {
		t.Fatal("expected dial error, got nil")
	}
}

func TestConnectionClose(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := defaultCfg()
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	c, err := conn.Dial(ctx, addr, cfg, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !c.IsClosed() {
		t.Fatal("IsClosed should be true after Close")
	}
	// verify TCP socket released: subsequent Send must return ErrClosed
	_, sendErr := c.Send(ctx, c.NextToken(), []byte(`[5]`))
	if !errors.Is(sendErr, conn.ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", sendErr)
	}
}

func TestConnectionConcurrent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const n = 10
	cfg := defaultCfg()
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	type result struct {
		c   *conn.Conn
		err error
	}
	results := make([]result, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func(i int) {
			defer wg.Done()
			c, err := conn.Dial(ctx, addr, cfg, nil)
			results[i] = result{c, err}
		}(i)
	}
	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Errorf("goroutine %d: %v", i, r.err)
		}
		if r.c != nil {
			_ = r.c.Close()
		}
	}
}

// readNullTerminated reads bytes from r until a null byte is encountered.
func readNullTerminated(r io.Reader) ([]byte, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		if b[0] == 0x00 {
			return buf, nil
		}
		buf = append(buf, b[0])
	}
}
