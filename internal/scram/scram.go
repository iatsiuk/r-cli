package scram

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// GenerateNonce returns a cryptographically random base64-encoded nonce of at least 18 bytes.
func GenerateNonce() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		panic("scram: failed to generate nonce: " + err.Error())
	}
	return base64.RawStdEncoding.EncodeToString(b)
}

// ClientFirstMessage returns the SCRAM client-first-message per RFC 5802:
// "n,,n=<user>,r=<nonce>" where user has = and , escaped.
func ClientFirstMessage(user, nonce string) string {
	return "n,,n=" + escapeUsername(user) + ",r=" + nonce
}

// escapeUsername replaces = with =3D and , with =2C as required by RFC 5802.
func escapeUsername(user string) string {
	user = strings.ReplaceAll(user, "=", "=3D")
	user = strings.ReplaceAll(user, ",", "=2C")
	return user
}

// ServerFirst holds parsed fields from the server-first-message.
type ServerFirst struct {
	Nonce      string
	Salt       []byte
	Iterations int
}

// parseServerFields splits a server-first-message into a key->value map.
func parseServerFields(msg string) (map[string]string, error) {
	fields := make(map[string]string)
	for _, part := range strings.Split(msg, ",") {
		if len(part) < 2 || part[1] != '=' {
			return nil, fmt.Errorf("scram: malformed field %q", part)
		}
		key := string(part[0])
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("scram: duplicate field %q", key)
		}
		fields[key] = part[2:]
	}
	return fields, nil
}

// ComputeProof derives the ClientProof and ServerSignature per RFC 5802 using SCRAM-SHA-256.
// authMsg is the concatenation: client-first-bare + "," + server-first + "," + client-final-without-proof.
func ComputeProof(password string, salt []byte, iter int, authMsg string) (clientProof, serverSig []byte) {
	saltedPassword := pbkdf2SHA256([]byte(password), salt, iter)

	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKeyArr := sha256.Sum256(clientKey)
	storedKey := storedKeyArr[:]

	clientSig := hmacSHA256(storedKey, []byte(authMsg))
	proof := make([]byte, len(clientKey))
	for i := range clientKey {
		proof[i] = clientKey[i] ^ clientSig[i]
	}

	serverKey := hmacSHA256(saltedPassword, []byte("Server Key"))
	sig := hmacSHA256(serverKey, []byte(authMsg))
	return proof, sig
}

// pbkdf2SHA256 implements PBKDF2-HMAC-SHA256 with a 32-byte output per RFC 2898.
// iterations must be >= 1.
func pbkdf2SHA256(password, salt []byte, iterations int) []byte {
	if iterations < 1 {
		panic("scram: pbkdf2SHA256: iterations must be >= 1")
	}
	mac := hmac.New(sha256.New, password)
	mac.Write(salt)
	mac.Write([]byte{0, 0, 0, 1})
	u := mac.Sum(nil)
	result := make([]byte, len(u))
	copy(result, u)
	for i := 1; i < iterations; i++ {
		mac.Reset()
		mac.Write(u)
		u = mac.Sum(nil)
		for j := range result {
			result[j] ^= u[j]
		}
	}
	return result
}

// hmacSHA256 returns HMAC-SHA256(key, data).
func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// ClientFinalMessage builds the SCRAM client-final-message:
// "c=biws,r=<combinedNonce>,p=<base64proof>"
// where "biws" is base64("n,,") - the GS2 header with no channel binding.
func ClientFinalMessage(combinedNonce string, proof []byte) string {
	return "c=biws,r=" + combinedNonce + ",p=" + base64.StdEncoding.EncodeToString(proof)
}

// VerifyServerFinal parses the server-final-message "v=<base64sig>" or "e=<error>" and checks
// the signature against expectedSig using constant-time comparison.
func VerifyServerFinal(msg string, expectedSig []byte) error {
	const errPrefix = "e="
	if strings.HasPrefix(msg, errPrefix) {
		return fmt.Errorf("scram: server authentication error: %q", msg[len(errPrefix):])
	}
	const prefix = "v="
	if !strings.HasPrefix(msg, prefix) {
		return fmt.Errorf("scram: invalid server-final-message %q", msg)
	}
	sig, err := base64.StdEncoding.DecodeString(msg[len(prefix):])
	if err != nil {
		return fmt.Errorf("scram: invalid server signature encoding: %w", err)
	}
	if len(sig) == 0 || len(expectedSig) == 0 || !hmac.Equal(sig, expectedSig) {
		return fmt.Errorf("scram: server signature mismatch")
	}
	return nil
}

// Conversation tracks SCRAM-SHA-256 state across a 3-message exchange.
type Conversation struct {
	username        string
	password        string
	clientNonce     string
	clientFirstBare string
	serverFirstMsg  string
	serverSig       []byte
}

// NewConversation creates a new SCRAM conversation for the given credentials.
func NewConversation(username, password string) *Conversation {
	return &Conversation{username: username, password: password}
}

// ClientFirst generates the client-first-message.
func (c *Conversation) ClientFirst() string {
	if c.clientNonce == "" {
		c.clientNonce = GenerateNonce()
	}
	msg := ClientFirstMessage(c.username, c.clientNonce)
	c.clientFirstBare = msg[len("n,,"):]
	return msg
}

// ServerFirst processes the server-first-message and returns the client-final-message.
func (c *Conversation) ServerFirst(msg string) (string, error) {
	if c.clientFirstBare == "" {
		return "", fmt.Errorf("scram: ClientFirst must be called before ServerFirst")
	}
	sf, err := ParseServerFirst(msg, c.clientNonce)
	if err != nil {
		return "", err
	}
	c.serverFirstMsg = msg

	finalWithoutProof := "c=biws,r=" + sf.Nonce
	authMsg := c.clientFirstBare + "," + c.serverFirstMsg + "," + finalWithoutProof

	proof, serverSig := ComputeProof(c.password, sf.Salt, sf.Iterations, authMsg)
	c.serverSig = serverSig

	return ClientFinalMessage(sf.Nonce, proof), nil
}

// ServerFinal verifies the server-final-message against the expected server signature.
func (c *Conversation) ServerFinal(msg string) error {
	if c.serverSig == nil {
		return fmt.Errorf("scram: ServerFirst must be called before ServerFinal")
	}
	return VerifyServerFinal(msg, c.serverSig)
}

// ParseServerFirst parses a SCRAM server-first-message and validates the nonce prefix.
func ParseServerFirst(msg, clientNonce string) (*ServerFirst, error) {
	if clientNonce == "" {
		return nil, fmt.Errorf("scram: empty client nonce")
	}
	fields, err := parseServerFields(msg)
	if err != nil {
		return nil, err
	}
	return buildServerFirst(fields, clientNonce)
}

// buildServerFirst validates the parsed fields and constructs a ServerFirst.
func buildServerFirst(fields map[string]string, clientNonce string) (*ServerFirst, error) {
	if _, ok := fields["m"]; ok {
		return nil, fmt.Errorf("scram: unsupported mandatory extension")
	}

	nonce, ok := fields["r"]
	if !ok {
		return nil, fmt.Errorf("scram: missing nonce field")
	}
	if !strings.HasPrefix(nonce, clientNonce) || len(nonce) == len(clientNonce) {
		return nil, fmt.Errorf("scram: server nonce does not extend client nonce")
	}

	salt, err := decodeSalt(fields)
	if err != nil {
		return nil, err
	}

	iter, err := decodeIter(fields)
	if err != nil {
		return nil, err
	}

	return &ServerFirst{Nonce: nonce, Salt: salt, Iterations: iter}, nil
}

func decodeSalt(fields map[string]string) ([]byte, error) {
	saltB64, ok := fields["s"]
	if !ok {
		return nil, fmt.Errorf("scram: missing salt field")
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("scram: invalid salt: %w", err)
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("scram: empty salt")
	}
	return salt, nil
}

func decodeIter(fields map[string]string) (int, error) {
	iterStr, ok := fields["i"]
	if !ok {
		return 0, fmt.Errorf("scram: missing iteration count field")
	}
	iter, err := strconv.Atoi(iterStr)
	if err != nil || iter < 1 {
		return 0, fmt.Errorf("scram: invalid iteration count %q", iterStr)
	}
	return iter, nil
}
