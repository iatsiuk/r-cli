package conn

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"r-cli/internal/proto"
	"r-cli/internal/scram"
)

func TestBuildStep1(t *testing.T) {
	t.Parallel()

	got := buildStep1()
	want := make([]byte, 4)
	binary.LittleEndian.PutUint32(want, uint32(proto.V1_0))

	if !bytes.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// 0x34c2bdc3 in LE: c3 bd c2 34
	if got[0] != 0xc3 || got[1] != 0xbd || got[2] != 0xc2 || got[3] != 0x34 {
		t.Fatalf("wrong magic bytes: %02x %02x %02x %02x", got[0], got[1], got[2], got[3])
	}
}

func TestBuildStep3(t *testing.T) {
	t.Parallel()

	clientFirst := "n,,n=user,r=nonce123"
	got, err := buildStep3(clientFirst)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[len(got)-1] != 0x00 {
		t.Fatal("missing null terminator")
	}

	var msg struct {
		ProtocolVersion      int    `json:"protocol_version"`
		AuthenticationMethod string `json:"authentication_method"`
		Authentication       string `json:"authentication"`
	}
	if err := json.Unmarshal(got[:len(got)-1], &msg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if msg.ProtocolVersion != 0 {
		t.Errorf("protocol_version: got %d, want 0", msg.ProtocolVersion)
	}
	if msg.AuthenticationMethod != "SCRAM-SHA-256" {
		t.Errorf("authentication_method: got %q, want SCRAM-SHA-256", msg.AuthenticationMethod)
	}
	if msg.Authentication != clientFirst {
		t.Errorf("authentication: got %q, want %q", msg.Authentication, clientFirst)
	}
}

func TestBuildStep5(t *testing.T) {
	t.Parallel()

	clientFinal := "c=biws,r=nonce123server,p=proof"
	got, err := buildStep5(clientFinal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[len(got)-1] != 0x00 {
		t.Fatal("missing null terminator")
	}

	var msg struct {
		Authentication string `json:"authentication"`
	}
	if err := json.Unmarshal(got[:len(got)-1], &msg); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if msg.Authentication != clientFinal {
		t.Errorf("authentication: got %q, want %q", msg.Authentication, clientFinal)
	}
}

func TestParseStep2(t *testing.T) {
	t.Parallel()

	t.Run("success parses version fields", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"success":true,"min_protocol_version":0,"max_protocol_version":0,"server_version":"2.3.0"}`)
		resp, err := parseStep2(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.MinProtocolVersion != 0 {
			t.Errorf("min: got %d, want 0", resp.MinProtocolVersion)
		}
		if resp.MaxProtocolVersion != 0 {
			t.Errorf("max: got %d, want 0", resp.MaxProtocolVersion)
		}
		if resp.ServerVersion != "2.3.0" {
			t.Errorf("server_version: got %q, want 2.3.0", resp.ServerVersion)
		}
	})

	t.Run("non-JSON error string returns error", func(t *testing.T) {
		t.Parallel()
		_, err := parseStep2([]byte("ERROR: connection refused"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("success false returns error", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"success":false,"error":"unsupported version"}`)
		_, err := parseStep2(data)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestParseStep4(t *testing.T) {
	t.Parallel()

	t.Run("success returns authentication field", func(t *testing.T) {
		t.Parallel()
		auth := "r=nonce123server,s=c2FsdA==,i=4096"
		data, _ := json.Marshal(map[string]interface{}{
			"success":        true,
			"authentication": auth,
		})
		got, err := parseStep4(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != auth {
			t.Errorf("got %q, want %q", got, auth)
		}
	})

	t.Run("error_code 10-20 wraps ErrReqlAuth", func(t *testing.T) {
		t.Parallel()
		data, _ := json.Marshal(map[string]interface{}{
			"success":    false,
			"error":      "wrong password",
			"error_code": 12,
		})
		_, err := parseStep4(data)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrReqlAuth) {
			t.Errorf("expected ErrReqlAuth, got %v", err)
		}
	})

	t.Run("error_code outside 10-20 is not ErrReqlAuth", func(t *testing.T) {
		t.Parallel()
		data, _ := json.Marshal(map[string]interface{}{
			"success":    false,
			"error":      "other error",
			"error_code": 5,
		})
		_, err := parseStep4(data)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, ErrReqlAuth) {
			t.Error("should not be ErrReqlAuth for code outside 10-20")
		}
	})
}

func TestParseStep6(t *testing.T) {
	t.Parallel()

	t.Run("success returns authentication field", func(t *testing.T) {
		t.Parallel()
		serverSig := "v=serverSignatureBase64=="
		data, _ := json.Marshal(map[string]interface{}{
			"success":        true,
			"authentication": serverSig,
		})
		got, err := parseStep6(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != serverSig {
			t.Errorf("got %q, want %q", got, serverSig)
		}
	})

	t.Run("failure returns error", func(t *testing.T) {
		t.Parallel()
		data, _ := json.Marshal(map[string]interface{}{
			"success": false,
			"error":   "bad signature",
		})
		_, err := parseStep6(data)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// mockSCRAMServer simulates the server side of a RethinkDB V1_0 handshake.
// It reads step 3 BEFORE sending step 2 to require pipelined client behavior.
type mockSCRAMServer struct {
	password      string
	step2JSON     string // empty = use default success response
	step4ErrorMsg string // if non-empty, send step 4 error instead of server-first
	step4ErrCode  int
}

// setup reads and validates step 1 (magic) and step 3 (client-first-message).
// Returns clientFirstMsg and true on success.
func (m *mockSCRAMServer) setup(t *testing.T, rw io.ReadWriter) (string, bool) {
	t.Helper()

	magic := make([]byte, 4)
	if _, err := io.ReadFull(rw, magic); err != nil {
		t.Errorf("mock: read magic: %v", err)
		return "", false
	}
	if !bytes.Equal(magic, buildStep1()) {
		t.Errorf("mock: wrong magic bytes: %x", magic)
		return "", false
	}

	// read step 3 BEFORE step 2 to enforce pipelining
	data, err := readNullTerminated(rw)
	if err != nil {
		t.Errorf("mock: read step 3: %v", err)
		return "", false
	}
	var req step3Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Errorf("mock: parse step 3: %v", err)
		return "", false
	}
	if req.ProtocolVersion != 0 || req.AuthenticationMethod != "SCRAM-SHA-256" {
		t.Errorf("mock: step 3: protocol_version=%d method=%q", req.ProtocolVersion, req.AuthenticationMethod)
		return "", false
	}
	return req.Authentication, true
}

// serve runs the mock server-side handshake. Must be called from a goroutine.
func (m *mockSCRAMServer) serve(t *testing.T, rw io.ReadWriter) {
	t.Helper()

	clientFirstMsg, ok := m.setup(t, rw)
	if !ok {
		return
	}

	step2 := m.step2JSON
	if step2 == "" {
		step2 = `{"success":true,"min_protocol_version":0,"max_protocol_version":0,"server_version":"2.3.0"}`
	}
	if err := writeNullTerminated(rw, []byte(step2)); err != nil {
		t.Errorf("mock: write step 2: %v", err)
		return
	}

	var s2 step2Response
	if err := json.Unmarshal([]byte(step2), &s2); err != nil {
		return
	}
	if !s2.Success || s2.MinProtocolVersion > 0 {
		return
	}

	if m.step4ErrorMsg != "" {
		resp, _ := json.Marshal(map[string]interface{}{
			"success":    false,
			"error":      m.step4ErrorMsg,
			"error_code": m.step4ErrCode,
		})
		if err := writeNullTerminated(rw, resp); err != nil {
			t.Errorf("mock: write step 4 error: %v", err)
		}
		return
	}

	m.completeSCRAM(t, rw, clientFirstMsg)
}

// completeSCRAM handles the SCRAM key exchange (steps 4-6).
func (m *mockSCRAMServer) completeSCRAM(t *testing.T, rw io.ReadWriter, clientFirstMsg string) {
	t.Helper()

	const serverNonceExt = "SERVERNONCE"
	salt := []byte("saltsaltsalt1234")
	iter := 4096

	// extract client nonce from "n,,n=user,r=<nonce>"
	clientFirstBare := clientFirstMsg[len("n,,"):]
	var clientNonce string
	for _, part := range strings.Split(clientFirstBare, ",") {
		if strings.HasPrefix(part, "r=") {
			clientNonce = part[len("r="):]
		}
	}
	combinedNonce := clientNonce + serverNonceExt
	serverFirstMsg := fmt.Sprintf("r=%s,s=%s,i=%d",
		combinedNonce, base64.StdEncoding.EncodeToString(salt), iter)

	// send step 4
	step4, _ := json.Marshal(map[string]interface{}{
		"success":        true,
		"authentication": serverFirstMsg,
	})
	if err := writeNullTerminated(rw, step4); err != nil {
		t.Errorf("mock: write step 4: %v", err)
		return
	}

	// read step 5
	step5Data, err := readNullTerminated(rw)
	if err != nil {
		t.Errorf("mock: read step 5: %v", err)
		return
	}
	var req5 step5Request
	if err := json.Unmarshal(step5Data, &req5); err != nil {
		t.Errorf("mock: parse step 5: %v", err)
		return
	}
	clientFinalMsg := req5.Authentication

	// extract client-final-without-proof for authMsg
	pIdx := strings.LastIndex(clientFinalMsg, ",p=")
	if pIdx < 0 {
		t.Error("mock: missing proof in client-final")
		return
	}
	clientFinalWithoutProof := clientFinalMsg[:pIdx]
	authMsg := clientFirstBare + "," + serverFirstMsg + "," + clientFinalWithoutProof

	// verify client proof
	expectedProof, serverSig := scram.ComputeProof(m.password, salt, iter, authMsg)
	proofB64 := clientFinalMsg[pIdx+len(",p="):]
	actualProof, err := base64.StdEncoding.DecodeString(proofB64)
	if err != nil {
		t.Errorf("mock: invalid proof encoding: %v", err)
		return
	}
	if !bytes.Equal(actualProof, expectedProof) {
		t.Errorf("mock: client proof mismatch")
		return
	}

	// send step 6
	step6, _ := json.Marshal(map[string]interface{}{
		"success":        true,
		"authentication": "v=" + base64.StdEncoding.EncodeToString(serverSig),
	})
	if err := writeNullTerminated(rw, step6); err != nil {
		t.Errorf("mock: write step 6: %v", err)
	}
}

func TestHandshakeFullSuccess(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	srv := &mockSCRAMServer{password: "testpass"}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = server.Close() }()
		srv.serve(t, server)
	}()

	if err := Handshake(client, "testuser", "testpass"); err != nil {
		t.Fatalf("Handshake error: %v", err)
	}
	<-done
}

func TestHandshakeWrongPassword(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	srv := &mockSCRAMServer{
		password:      "correctpass",
		step4ErrorMsg: "wrong password",
		step4ErrCode:  12,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = server.Close() }()
		srv.serve(t, server)
	}()

	err := Handshake(client, "testuser", "wrongpass")
	<-done
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	if !errors.Is(err, ErrReqlAuth) {
		t.Errorf("expected ErrReqlAuth, got %v", err)
	}
}

func TestHandshakeIncompatibleVersion(t *testing.T) {
	t.Parallel()

	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	srv := &mockSCRAMServer{
		step2JSON: `{"success":true,"min_protocol_version":1,"max_protocol_version":2,"server_version":"3.0.0"}`,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = server.Close() }()
		srv.serve(t, server)
	}()

	err := Handshake(client, "user", "pass")
	<-done
	if err == nil {
		t.Fatal("expected incompatible version error, got nil")
	}
	if !strings.Contains(err.Error(), "protocol_version") {
		t.Errorf("expected protocol_version in error, got %v", err)
	}
}

func TestHandshakePipelined(t *testing.T) {
	t.Parallel()
	// the mock server reads step 3 before sending step 2, so a non-pipelining
	// client would deadlock. The timeout catches that case.

	client, server := net.Pipe()

	srv := &mockSCRAMServer{password: "pass"}
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		defer func() { _ = server.Close() }()
		srv.serve(t, server)
	}()

	clientDone := make(chan error, 1)
	go func() {
		defer func() { _ = client.Close() }()
		clientDone <- Handshake(client, "user", "pass")
	}()

	select {
	case err := <-clientDone:
		if err != nil {
			t.Fatalf("Handshake error: %v", err)
		}
	case <-time.After(3 * time.Second):
		_ = client.Close()
		t.Fatal("Handshake deadlocked - steps 1+3 not pipelined")
	}
	<-serverDone
}
