package query

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"r-cli/internal/conn"
	"r-cli/internal/connmgr"
	"r-cli/internal/reql"
	"r-cli/internal/scram"
	"r-cli/internal/wire"
)

// queryHandler is called for each query received by the mock server.
type queryHandler func(nc net.Conn, token uint64, payload []byte)

// startQueryServer starts a mock RethinkDB server that completes the V1_0
// handshake and then calls handler for each received query frame.
func startQueryServer(t *testing.T, password string, handler queryHandler) (addr string, stop func()) {
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
			go serveQueryConn(nc, password, handler)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func serveQueryConn(nc net.Conn, password string, handler queryHandler) {
	defer func() { _ = nc.Close() }()
	if err := serverHandshake(nc, password); err != nil {
		return
	}
	for {
		token, payload, err := wire.ReadResponse(nc)
		if err != nil {
			return
		}
		handler(nc, token, payload)
	}
}

// sendResponse writes a wire frame response to nc.
func sendResponse(nc net.Conn, token uint64, resp interface{}) {
	raw, _ := json.Marshal(resp)
	_ = wire.WriteQuery(nc, token, raw)
}

// seqResp builds a SUCCESS_SEQUENCE response body.
func seqResp(items []interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": 2, // ResponseSuccessSequence
		"r": items,
	}
}

// newTestExecutor builds an Executor connected to addr using the given password.
func newTestExecutor(t *testing.T, addr, password string) *Executor {
	t.Helper()
	host, portStr, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portStr)
	cfg := conn.Config{Host: host, Port: port, User: "admin", Password: password}
	mgr := connmgr.NewFromConfig(cfg, nil)
	t.Cleanup(func() { _ = mgr.Close() })
	return New(mgr)
}

// --- handshake helpers (mirror of connmgr_test internals) ---

func serverHandshake(nc net.Conn, password string) error {
	clientFirstMsg, err := readHandshakeInit(nc)
	if err != nil {
		return err
	}
	return completeSCRAM(nc, clientFirstMsg, password)
}

func readHandshakeInit(nc net.Conn) (string, error) {
	magic := make([]byte, 4)
	if _, err := io.ReadFull(nc, magic); err != nil {
		return "", err
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
	return req3.Authentication, writeNull(nc, step2)
}

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

// --- tests ---

func TestExecutorRunGetsSequence(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	handler := func(nc net.Conn, token uint64, _ []byte) {
		sendResponse(nc, token, seqResp([]interface{}{
			map[string]interface{}{"id": 1, "name": "Alice"},
			map[string]interface{}{"id": 2, "name": "Bob"},
		}))
	}
	addr, stop := startQueryServer(t, pass, handler)
	defer stop()

	ex := newTestExecutor(t, addr, pass)
	cur, err := ex.Run(context.Background(), reql.DB("test").Table("users"), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer func() { _ = cur.Close() }()

	items, err := cur.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
}

func TestExecutorRunWithDBOption(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	received := make(chan []byte, 1)
	handler := func(nc net.Conn, token uint64, payload []byte) {
		received <- payload
		sendResponse(nc, token, seqResp(nil))
	}
	addr, stop := startQueryServer(t, pass, handler)
	defer stop()

	ex := newTestExecutor(t, addr, pass)
	_, err := ex.Run(context.Background(), reql.DB("test").Table("users"), reql.OptArgs{"db": "mydb"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	select {
	case payload := <-received:
		if !strings.Contains(string(payload), "mydb") {
			t.Errorf("payload does not contain db option: %s", payload)
		}
	case <-time.After(time.Second):
		t.Error("server did not receive query within 1s")
	}
}

func TestExecutorRunWithTimeout(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	handler := func(_ net.Conn, _ uint64, _ []byte) {
		// never respond; let ctx expire
	}
	addr, stop := startQueryServer(t, pass, handler)
	defer stop()

	ex := newTestExecutor(t, addr, pass)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := ex.Run(ctx, reql.DB("test").Table("users"), nil)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestExecutorServerInfo(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	handler := func(nc net.Conn, token uint64, _ []byte) {
		sendResponse(nc, token, map[string]interface{}{
			"t": 5, // ResponseServerInfo
			"r": []interface{}{
				map[string]interface{}{
					"id":   "some-uuid-1234",
					"name": "test-server",
				},
			},
		})
	}
	addr, stop := startQueryServer(t, pass, handler)
	defer stop()

	ex := newTestExecutor(t, addr, pass)
	info, err := ex.ServerInfo(context.Background())
	if err != nil {
		t.Fatalf("ServerInfo: %v", err)
	}
	if info.ID != "some-uuid-1234" {
		t.Errorf("ID: got %q, want %q", info.ID, "some-uuid-1234")
	}
	if info.Name != "test-server" {
		t.Errorf("Name: got %q, want %q", info.Name, "test-server")
	}
}

func TestExecutorRunWithNoreply(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	received := make(chan []byte, 1)
	handler := func(_ net.Conn, _ uint64, payload []byte) {
		received <- payload
		// no response for noreply
	}
	addr, stop := startQueryServer(t, pass, handler)
	defer stop()

	ex := newTestExecutor(t, addr, pass)
	cur, err := ex.Run(context.Background(), reql.DB("test").Table("users"), reql.OptArgs{"noreply": true})
	if err != nil {
		t.Fatalf("Run with noreply: %v", err)
	}
	if cur != nil {
		t.Error("expected nil cursor for noreply query")
	}

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Error("server did not receive noreply query within 1s")
	}
}

func TestExecutorRunGetsAtom(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	handler := func(nc net.Conn, token uint64, _ []byte) {
		sendResponse(nc, token, map[string]interface{}{
			"t": 1, // ResponseSuccessAtom
			"r": []interface{}{map[string]interface{}{"value": 42}},
		})
	}
	addr, stop := startQueryServer(t, pass, handler)
	defer stop()

	ex := newTestExecutor(t, addr, pass)
	cur, err := ex.Run(context.Background(), reql.DB("test").Table("users").Count(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer func() { _ = cur.Close() }()

	item, err := cur.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if string(item) == "" {
		t.Fatal("got empty item from atom cursor")
	}
}

func TestExecutorRunServerError(t *testing.T) {
	t.Parallel()
	const pass = "testpass"
	handler := func(nc net.Conn, token uint64, _ []byte) {
		sendResponse(nc, token, map[string]interface{}{
			"t": 18, // ResponseRuntimeError
			"r": []interface{}{"table `users` does not exist"},
			"e": 4, // ErrorType runtime
		})
	}
	addr, stop := startQueryServer(t, pass, handler)
	defer stop()

	ex := newTestExecutor(t, addr, pass)
	_, err := ex.Run(context.Background(), reql.DB("test").Table("users"), nil)
	if err == nil {
		t.Fatal("expected error from server, got nil")
	}
}
