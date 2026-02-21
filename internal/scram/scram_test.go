package scram

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateNonce(t *testing.T) {
	t.Parallel()

	nonce := GenerateNonce()

	// must be valid base64
	decoded, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		t.Fatalf("nonce is not valid base64: %v", err)
	}

	// must be at least 18 bytes of entropy
	if len(decoded) < 18 {
		t.Errorf("nonce decoded length=%d, want >= 18", len(decoded))
	}

	// must contain no commas (comma is a SCRAM field separator)
	if strings.Contains(nonce, ",") {
		t.Errorf("nonce contains comma: %q", nonce)
	}
}

func TestGenerateNonceUniqueness(t *testing.T) {
	t.Parallel()

	a := GenerateNonce()
	b := GenerateNonce()
	if a == b {
		t.Error("two consecutive nonces are identical")
	}
}

func TestClientFirstMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		user  string
		nonce string
		want  string
	}{
		{
			name:  "plain username",
			user:  "user",
			nonce: "fyko+d2lbbFgONRv9qkxdawL",
			want:  "n,,n=user,r=fyko+d2lbbFgONRv9qkxdawL",
		},
		{
			name:  "username with equals sign",
			user:  "us=er",
			nonce: "abc",
			want:  "n,,n=us=3Der,r=abc",
		},
		{
			name:  "username with comma",
			user:  "us,er",
			nonce: "abc",
			want:  "n,,n=us=2Cer,r=abc",
		},
		{
			name:  "username with both special chars",
			user:  "a=b,c",
			nonce: "xyz",
			want:  "n,,n=a=3Db=2Cc,r=xyz",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ClientFirstMessage(tc.user, tc.nonce)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseServerFirst(t *testing.T) {
	t.Parallel()

	clientNonce := "fyko+d2lbbFgONRv9qkxdawL"
	saltB64 := "QSXCR+Q6sek8bf92"
	wantSalt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		t.Fatalf("test setup: invalid base64 salt: %v", err)
	}
	wantNonce := "fyko+d2lbbFgONRv9qkxdawL3rfcNHYJY1ZVvWVs7j"
	msg := "r=" + wantNonce + ",s=" + saltB64 + ",i=4096"

	sf, err := ParseServerFirst(msg, clientNonce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sf.Nonce != wantNonce {
		t.Errorf("nonce=%q, want %q", sf.Nonce, wantNonce)
	}
	if !bytes.Equal(sf.Salt, wantSalt) {
		t.Errorf("salt=%x, want %x", sf.Salt, wantSalt)
	}
	if sf.Iterations != 4096 {
		t.Errorf("iterations=%d, want 4096", sf.Iterations)
	}
}

func TestParseServerFirstMalformed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		msg         string
		clientNonce string
	}{
		{
			name:        "empty message",
			msg:         "",
			clientNonce: "nonce",
		},
		{
			name:        "missing nonce field",
			msg:         "s=QSXCR+Q6sek8bf92,i=4096",
			clientNonce: "nonce",
		},
		{
			name:        "missing salt field",
			msg:         "r=noncecombined,i=4096",
			clientNonce: "nonce",
		},
		{
			name:        "missing iteration field",
			msg:         "r=noncecombined,s=QSXCR+Q6sek8bf92",
			clientNonce: "nonce",
		},
		{
			name:        "invalid base64 salt",
			msg:         "r=noncecombined,s=!!!invalid!!!,i=4096",
			clientNonce: "nonce",
		},
		{
			name:        "non-numeric iteration count",
			msg:         "r=noncecombined,s=QSXCR+Q6sek8bf92,i=abc",
			clientNonce: "nonce",
		},
		{
			name:        "zero iteration count",
			msg:         "r=noncecombined,s=QSXCR+Q6sek8bf92,i=0",
			clientNonce: "nonce",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseServerFirst(tc.msg, tc.clientNonce)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseServerFirstWrongNoncePrefix(t *testing.T) {
	t.Parallel()

	msg := "r=servernonceXXX,s=QSXCR+Q6sek8bf92,i=4096"
	_, err := ParseServerFirst(msg, "clientnonce")
	if err == nil {
		t.Error("expected error for wrong nonce prefix, got nil")
	}
}
