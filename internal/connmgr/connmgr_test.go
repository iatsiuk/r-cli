package connmgr

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"r-cli/internal/conn"
	"r-cli/internal/scram"
)

// startTestServer starts a TCP listener that performs the RethinkDB V1_0
// handshake server-side. Returns the listener address and a stop function.
func startTestServer(t *testing.T, password string) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			go serveHandshake(nc, password)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

// idleUntilClosed reads from nc discarding data until the connection closes.
func idleUntilClosed(nc net.Conn) {
	buf := make([]byte, 1)
	for {
		if _, err := nc.Read(buf); err != nil {
			_ = nc.Close()
			return
		}
	}
}

// readHandshakeInit reads magic + step3, sends step2, returns clientFirstMsg.
func readHandshakeInit(nc net.Conn) (string, error) {
	magic := make([]byte, 4)
	if _, err := io.ReadFull(nc, magic); err != nil {
		return "", err
	}
	if binary.LittleEndian.Uint32(magic) == 0 {
		return "", fmt.Errorf("invalid magic")
	}

	step3, err := readNull(nc)
	if err != nil {
		return "", err
	}
	var req3 struct {
		Authentication string `json:"authentication"`
	}
	if err := json.Unmarshal(step3, &req3); err != nil {
		return "", err
	}

	step2, _ := json.Marshal(map[string]interface{}{
		"success":              true,
		"min_protocol_version": 0,
		"max_protocol_version": 0,
		"server_version":       "2.3.0",
	})
	if err := writeNull(nc, step2); err != nil {
		return "", err
	}
	return req3.Authentication, nil
}

// completeSCRAM handles the SCRAM key exchange (steps 4-6).
func completeSCRAM(nc net.Conn, clientFirstMsg, password string) error {
	clientFirstBare := clientFirstMsg[len("n,,"):]
	clientNonce := extractNonce(clientFirstBare)

	salt := []byte("saltsaltsalt1234")
	iter := 4096
	combinedNonce := clientNonce + "SERVERNONCE"
	serverFirstMsg := fmt.Sprintf("r=%s,s=%s,i=%d",
		combinedNonce, base64.StdEncoding.EncodeToString(salt), iter)

	step4, _ := json.Marshal(map[string]interface{}{"success": true, "authentication": serverFirstMsg})
	if err := writeNull(nc, step4); err != nil {
		return err
	}

	step5, err := readNull(nc)
	if err != nil {
		return err
	}
	var req5 struct {
		Authentication string `json:"authentication"`
	}
	if err := json.Unmarshal(step5, &req5); err != nil {
		return err
	}

	clientFinalMsg := req5.Authentication
	pIdx := strings.LastIndex(clientFinalMsg, ",p=")
	if pIdx < 0 {
		return fmt.Errorf("missing proof")
	}
	authMsg := clientFirstBare + "," + serverFirstMsg + "," + clientFinalMsg[:pIdx]
	_, serverSig := scram.ComputeProof(password, salt, iter, authMsg)

	step6, _ := json.Marshal(map[string]interface{}{
		"success":        true,
		"authentication": "v=" + base64.StdEncoding.EncodeToString(serverSig),
	})
	return writeNull(nc, step6)
}

func extractNonce(clientFirstBare string) string {
	for _, part := range strings.Split(clientFirstBare, ",") {
		if strings.HasPrefix(part, "r=") {
			return part[2:]
		}
	}
	return ""
}

// serveHandshake runs the server-side RethinkDB V1_0 handshake then idles.
func serveHandshake(nc net.Conn, password string) {
	defer idleUntilClosed(nc)
	clientFirstMsg, err := readHandshakeInit(nc)
	if err != nil {
		return
	}
	_ = completeSCRAM(nc, clientFirstMsg, password)
}

// serveHandshakeThenClose completes the handshake then immediately closes the connection.
func serveHandshakeThenClose(nc net.Conn, password string) {
	defer func() { _ = nc.Close() }()
	clientFirstMsg, err := readHandshakeInit(nc)
	if err != nil {
		return
	}
	_ = completeSCRAM(nc, clientFirstMsg, password)
}

// startDropOnceServer starts a server that drops the first connection after
// handshake and serves subsequent connections normally.
func startDropOnceServer(t *testing.T, password string) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	var dropped atomic.Bool
	go func() {
		for {
			nc, err := ln.Accept()
			if err != nil {
				return
			}
			if !dropped.Swap(true) {
				go serveHandshakeThenClose(nc, password)
			} else {
				go serveHandshake(nc, password)
			}
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func readNull(r io.Reader) ([]byte, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		if b[0] == 0 {
			return buf, nil
		}
		buf = append(buf, b[0])
	}
}

func writeNull(w io.Writer, data []byte) error {
	_, err := w.Write(append(data, 0))
	return err
}

// testDialFunc returns a DialFunc that connects to addr with the given password.
func testDialFunc(addr, password string) DialFunc {
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	cfg := conn.Config{
		Host:     host,
		Port:     port,
		User:     "admin",
		Password: password,
	}
	return func(ctx context.Context) (*conn.Conn, error) {
		return conn.Dial(ctx, addr, cfg, nil)
	}
}

func TestGetCreatesConnectionOnFirstCall(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	addr, stop := startTestServer(t, pass)
	defer stop()

	dialCount := 0
	base := testDialFunc(addr, pass)
	dial := func(ctx context.Context) (*conn.Conn, error) {
		dialCount++
		return base(ctx)
	}

	mgr := New(dial)
	defer func() { _ = mgr.Close() }()

	c, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if c == nil {
		t.Fatal("Get returned nil conn")
	}
	if dialCount != 1 {
		t.Fatalf("dial called %d times, want 1", dialCount)
	}
}

func TestGetReturnsSameConnection(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	addr, stop := startTestServer(t, pass)
	defer stop()

	dialCount := 0
	base := testDialFunc(addr, pass)
	dial := func(ctx context.Context) (*conn.Conn, error) {
		dialCount++
		return base(ctx)
	}

	mgr := New(dial)
	defer func() { _ = mgr.Close() }()

	c1, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	c2, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if c1 != c2 {
		t.Fatal("second Get returned different connection")
	}
	if dialCount != 1 {
		t.Fatalf("dial called %d times, want 1", dialCount)
	}
}

func TestGetReconnectsAfterDrop(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	addr, stop := startDropOnceServer(t, pass)
	defer stop()

	mgr := New(testDialFunc(addr, pass))
	defer func() { _ = mgr.Close() }()

	c1, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}

	// wait for server to drop c1 and readLoop to detect it
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !c1.IsClosed() {
		time.Sleep(time.Millisecond)
	}
	if !c1.IsClosed() {
		t.Fatal("connection not marked closed after 2s")
	}

	c2, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("reconnect Get: %v", err)
	}
	if c1 == c2 {
		t.Fatal("expected a new connection after drop, got the same pointer")
	}
}

func TestGetDuringServerDowntime(t *testing.T) {
	t.Parallel()
	// get a free port then close listener so nothing is listening there
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	dialCount := 0
	mgr := New(func(ctx context.Context) (*conn.Conn, error) {
		dialCount++
		cfg := conn.Config{Host: "127.0.0.1", Port: 1, User: "admin"}
		return conn.Dial(ctx, addr, cfg, nil)
	})

	if _, err := mgr.Get(context.Background()); err == nil {
		t.Fatal("expected dial error, got nil")
	}
	if dialCount != 1 {
		t.Fatalf("dial called %d times on first failure, want 1", dialCount)
	}

	// second Get must also re-dial (failed conn must not be cached)
	if _, err := mgr.Get(context.Background()); err == nil {
		t.Fatal("expected dial error on second Get, got nil")
	}
	if dialCount != 2 {
		t.Fatalf("dial called %d times after two failures, want 2", dialCount)
	}
}

func TestReconnectPreservesConfig(t *testing.T) {
	t.Parallel()
	const pass = "secret"
	addr, stop := startDropOnceServer(t, pass)
	defer stop()

	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	cfg := conn.Config{
		Host:     host,
		Port:     port,
		User:     "admin",
		Password: pass,
	}

	mgr := NewFromConfig(cfg, nil)
	defer func() { _ = mgr.Close() }()

	c1, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}

	// wait for drop
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !c1.IsClosed() {
		time.Sleep(time.Millisecond)
	}
	if !c1.IsClosed() {
		t.Fatal("connection not marked closed after 2s")
	}

	// reconnect must succeed using same host/port/user/password
	c2, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("reconnect with preserved config failed: %v", err)
	}
	if c1 == c2 {
		t.Fatal("expected a new connection, got the same pointer")
	}
}

func TestCloseClosesConnection(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	addr, stop := startTestServer(t, pass)
	defer stop()

	mgr := New(testDialFunc(addr, pass))

	c, err := mgr.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if err := mgr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// conn should be closed now
	_, err = c.Send(context.Background(), 1, []byte(`[1,[39,[]],{}]`))
	if err == nil {
		t.Fatal("expected error after Close, got nil")
	}
	if !errors.Is(err, conn.ErrClosed) {
		t.Errorf("expected ErrClosed, got %v", err)
	}
}
