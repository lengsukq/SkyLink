package drive

import (
	"testing"
	"time"
)

func TestJWT_RoundTrip(t *testing.T) {
	secret := []byte("test-secret")
	expires := time.Now().Add(2 * time.Minute)
	token, err := NewToken(secret, 123, expires)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.AccountID != 123 {
		t.Fatalf("account_id=%d want=123", claims.AccountID)
	}
}

