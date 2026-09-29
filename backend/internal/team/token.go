package team

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"

	"github.com/Josed20/Tallerflow/backend/platform/security"
)

const invitationTokenBytes = 32

func newInvitationToken(random io.Reader) (string, error) {
	return security.RandomToken(random, invitationTokenBytes)
}

func invitationDigest(pepper []byte, token string) [32]byte {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte("team-invitation:"))
	_, _ = mac.Write([]byte(token))
	var out [32]byte
	copy(out[:], mac.Sum(nil))
	return out
}

func encodeDigest(digest [32]byte) []byte {
	return digest[:]
}

func publicToken(token string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(token))
}
