package passwordreset

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	rawTokenBytes = 32
	minTokenLen   = 32
	maxTokenLen   = 256
)

// GenerateToken produces a cryptographically secure random token and its SHA-256 hash.
// The raw token is meant to be sent to the user via delivery (email) and NEVER saved to PostgreSQL.
// Only the returned tokenHash ([]byte) is persisted in the database.
func GenerateToken() (rawToken string, tokenHash []byte, err error) {
	bytes := make([]byte, rawTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, fmt.Errorf("read secure random bytes: %w", err)
	}
	rawToken = hex.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(rawToken))
	return rawToken, hash[:], nil
}

// HashToken computes the SHA-256 hash of a raw token received from the client.
func HashToken(rawToken string) ([]byte, error) {
	trimmed := strings.TrimSpace(rawToken)
	if len(trimmed) < minTokenLen || len(trimmed) > maxTokenLen {
		return nil, errors.New("raw token length out of bounds")
	}
	hash := sha256.Sum256([]byte(trimmed))
	return hash[:], nil
}
