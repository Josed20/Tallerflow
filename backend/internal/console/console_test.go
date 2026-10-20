package console

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEncryptSecretRoundTrip(t *testing.T) {
	service := &Service{key: keyDigest("a sufficiently long console encryption key")}
	ciphertext, err := EncryptSecret("a sufficiently long console encryption key", "JBSWY3DPEHPK3PXP")
	require.NoError(t, err)
	plain, err := service.decrypt(ciphertext)
	require.NoError(t, err)
	require.Equal(t, "JBSWY3DPEHPK3PXP", plain)
}

func TestValidTOTPAcceptsCurrentCodeAndRejectsWrongCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Date(2026, 10, 20, 17, 0, 0, 0, time.UTC)
	require.True(t, validTOTP(secret, totpAt(secret, now), now))
	require.False(t, validTOTP(secret, "000000", now))
}

func keyDigest(value string) []byte   { digest := sha256Sum(value); return digest[:] }
func sha256Sum(value string) [32]byte { return sha256.Sum256([]byte(value)) }

func totpAt(secret string, now time.Time) string {
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	payload := make([]byte, 8)
	binary.BigEndian.PutUint64(payload, uint64(now.Unix()/30))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(payload)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	value := (uint32(sum[offset])&0x7f)<<24 | uint32(sum[offset+1])<<16 | uint32(sum[offset+2])<<8 | uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1000000)
}
