package conn

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"

	"r-cli/internal/proto"
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
