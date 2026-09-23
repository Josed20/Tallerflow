package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const maxRandomBytes = 1 << 20

var ErrInvalidRandomLength = errors.New("random byte length is invalid")

func RandomBytes(source io.Reader, length int) ([]byte, error) {
	if source == nil || length <= 0 || length > maxRandomBytes {
		return nil, ErrInvalidRandomLength
	}
	value := make([]byte, length)
	if _, err := io.ReadFull(source, value); err != nil {
		return nil, fmt.Errorf("read cryptographic randomness: %w", err)
	}
	return value, nil
}

func RandomToken(source io.Reader, byteLength int) (string, error) {
	value, err := RandomBytes(source, byteLength)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func TokenDigest(pepper []byte, rawToken string) [32]byte {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(rawToken))
	var digest [32]byte
	copy(digest[:], mac.Sum(nil))
	return digest
}
