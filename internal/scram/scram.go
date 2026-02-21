package scram

import (
	"crypto/rand"
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
