package security

import (
	"crypto/rand"
	"encoding/base64"
)

// GeneratePassword creates a random URL-safe password with the given length.
// If n is non-positive, a default length of 18 is used.
func GeneratePassword(n int) (string, error) {
	if n <= 0 {
		n = 18
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// URL-safe, no padding
	return base64.RawURLEncoding.EncodeToString(b)[:n], nil
}

