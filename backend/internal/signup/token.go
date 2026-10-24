package signup

import (
	"crypto/rand"
	"crypto/sha256"

	"github.com/Josed20/Tallerflow/backend/platform/security"
)

const (
	tokenBytes = 32
	// tokenLength is the base64url length of 32 random bytes, without padding.
	tokenLength = 43
)

// GenerateToken returns the raw token for the emailed link and the SHA-256
// digest that is the only form stored in PostgreSQL.
func GenerateToken() (string, []byte, error) {
	raw, err := security.RandomToken(rand.Reader, tokenBytes)
	if err != nil {
		return "", nil, err
	}
	digest := sha256.Sum256([]byte(raw))
	return raw, digest[:], nil
}

// HashToken digests a token received from the client, rejecting values that
// cannot have come from GenerateToken before any database lookup.
func HashToken(raw string) ([]byte, bool) {
	if len(raw) != tokenLength {
		return nil, false
	}
	for _, r := range raw {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return nil, false
		}
	}
	digest := sha256.Sum256([]byte(raw))
	return digest[:], true
}
