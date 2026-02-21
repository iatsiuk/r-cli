package scram

import (
	"crypto/rand"
	"encoding/base64"
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
