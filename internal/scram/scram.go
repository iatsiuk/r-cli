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
	return base64.StdEncoding.EncodeToString(b)
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
		fields[string(part[0])] = part[2:]
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
func pbkdf2SHA256(password, salt []byte, iterations int) []byte {
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

// ParseServerFirst parses a SCRAM server-first-message and validates the nonce prefix.
func ParseServerFirst(msg, clientNonce string) (*ServerFirst, error) {
	fields, err := parseServerFields(msg)
	if err != nil {
		return nil, err
	}

	nonce, ok := fields["r"]
	if !ok {
		return nil, fmt.Errorf("scram: missing nonce field")
	}
	if !strings.HasPrefix(nonce, clientNonce) {
		return nil, fmt.Errorf("scram: server nonce does not start with client nonce")
	}

	saltB64, ok := fields["s"]
	if !ok {
		return nil, fmt.Errorf("scram: missing salt field")
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("scram: invalid salt: %w", err)
	}

	iterStr, ok := fields["i"]
	if !ok {
		return nil, fmt.Errorf("scram: missing iteration count field")
	}
	iter, err := strconv.Atoi(iterStr)
	if err != nil || iter < 1 {
		return nil, fmt.Errorf("scram: invalid iteration count %q", iterStr)
	}

	return &ServerFirst{
		Nonce:      nonce,
		Salt:       salt,
		Iterations: iter,
	}, nil
}
