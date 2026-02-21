package scram

import (
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
