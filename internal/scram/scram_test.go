package scram

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

// RFC 7677 SCRAM-SHA-256 test vectors.
// https://www.rfc-editor.org/rfc/rfc7677 Section 3
const (
	rfc7677Password = "pencil"
	// AuthMessage = client-first-bare + "," + server-first + "," + client-final-without-proof
	rfc7677AuthMsg = "n=user,r=rOprNGfwEbeRWgbNEkqO," +
		"r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096," +
		"c=biws,r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0"
	rfc7677SaltB64     = "W22ZaJ0SNY7soEsUEjb6gQ=="
	rfc7677Iterations  = 4096
	rfc7677ClientProof = "dHzbZapWIk4jUhN+Ute9ytag9zjfMHgsqmmiz7AndVQ="
	rfc7677ServerSig   = "6rriTRBi23WpRR/wtup+mMhUZUn/dB5nLTJRsjl95G4="
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
		{
			name:        "field without equals separator",
			msg:         "r=noncecombined,ab,i=4096",
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

func TestComputeProofClientProof(t *testing.T) {
	t.Parallel()

	salt, err := base64.StdEncoding.DecodeString(rfc7677SaltB64)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}
	wantProof, err := base64.StdEncoding.DecodeString(rfc7677ClientProof)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}

	got, _ := ComputeProof(rfc7677Password, salt, rfc7677Iterations, rfc7677AuthMsg)
	if !bytes.Equal(got, wantProof) {
		t.Errorf("clientProof=%s, want %s",
			base64.StdEncoding.EncodeToString(got), rfc7677ClientProof)
	}
}

func TestComputeProofServerSig(t *testing.T) {
	t.Parallel()

	salt, err := base64.StdEncoding.DecodeString(rfc7677SaltB64)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}
	wantSig, err := base64.StdEncoding.DecodeString(rfc7677ServerSig)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}

	_, got := ComputeProof(rfc7677Password, salt, rfc7677Iterations, rfc7677AuthMsg)
	if !bytes.Equal(got, wantSig) {
		t.Errorf("serverSig=%s, want %s",
			base64.StdEncoding.EncodeToString(got), rfc7677ServerSig)
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

func TestClientFinalMessage(t *testing.T) {
	t.Parallel()

	// use RFC 7677 values: combined nonce and known proof bytes
	combinedNonce := "rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0"
	proof, err := base64.StdEncoding.DecodeString(rfc7677ClientProof)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}
	want := "c=biws,r=" + combinedNonce + ",p=" + rfc7677ClientProof

	got := ClientFinalMessage(combinedNonce, proof)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVerifyServerFinalSuccess(t *testing.T) {
	t.Parallel()

	sig, err := base64.StdEncoding.DecodeString(rfc7677ServerSig)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}
	msg := "v=" + rfc7677ServerSig

	if err := VerifyServerFinal(msg, sig); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVerifyServerFinalWrongSig(t *testing.T) {
	t.Parallel()

	sig, err := base64.StdEncoding.DecodeString(rfc7677ServerSig)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}
	// flip one byte to make it wrong
	wrong := make([]byte, len(sig))
	copy(wrong, sig)
	wrong[0] ^= 0xFF

	msg := "v=" + base64.StdEncoding.EncodeToString(wrong)
	if err := VerifyServerFinal(msg, sig); err == nil {
		t.Error("expected error for wrong signature, got nil")
	}
}

func TestVerifyServerFinalNilExpected(t *testing.T) {
	t.Parallel()

	// "v=" decodes to empty bytes; hmac.Equal([]byte{}, nil) would be true without the length check.
	if err := VerifyServerFinal("v=", nil); err == nil {
		t.Error("expected error when expectedSig is nil, got nil")
	}
}

func TestConversationServerFirstBeforeClientFirst(t *testing.T) {
	t.Parallel()

	c := NewConversation("user", "pencil")
	_, err := c.ServerFirst("r=nonce,s=QSXCR+Q6sek8bf92,i=4096")
	if err == nil {
		t.Error("expected error when ServerFirst called before ClientFirst, got nil")
	}
}

func TestConversationFullExchange(t *testing.T) {
	t.Parallel()

	// RFC 7677 test vectors with fixed client nonce for deterministic output.
	c := &Conversation{
		username:    "user",
		password:    "pencil",
		clientNonce: "rOprNGfwEbeRWgbNEkqO",
	}

	clientFirst := c.ClientFirst()
	wantClientFirst := "n,,n=user,r=rOprNGfwEbeRWgbNEkqO"
	if clientFirst != wantClientFirst {
		t.Errorf("client-first=%q, want %q", clientFirst, wantClientFirst)
	}

	serverFirstMsg := "r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096"
	clientFinal, err := c.ServerFirst(serverFirstMsg)
	if err != nil {
		t.Fatalf("ServerFirst: %v", err)
	}
	wantClientFinal := "c=biws,r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,p=" + rfc7677ClientProof
	if clientFinal != wantClientFinal {
		t.Errorf("client-final=%q, want %q", clientFinal, wantClientFinal)
	}

	serverFinalMsg := "v=" + rfc7677ServerSig
	if err := c.ServerFinal(serverFinalMsg); err != nil {
		t.Errorf("ServerFinal: %v", err)
	}
}

func TestVerifyServerFinalInvalid(t *testing.T) {
	t.Parallel()

	sig, err := base64.StdEncoding.DecodeString(rfc7677ServerSig)
	if err != nil {
		t.Fatalf("test setup: %v", err)
	}

	tests := []struct {
		name string
		msg  string
	}{
		{name: "missing v= prefix", msg: "notaserver"},
		{name: "invalid base64", msg: "v=!!!invalid!!!"},
		{name: "empty signature", msg: "v="},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := VerifyServerFinal(tc.msg, sig); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
