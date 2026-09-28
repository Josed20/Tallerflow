package passwordreset

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateTokenProducesEntropyAndValidHash(t *testing.T) {
	rawToken1, hash1, err1 := GenerateToken()
	require.NoError(t, err1)
	require.Len(t, rawToken1, 64) // 32 bytes hex encoded
	require.Len(t, hash1, 32)     // sha-256 output

	rawToken2, hash2, err2 := GenerateToken()
	require.NoError(t, err2)
	require.NotEqual(t, rawToken1, rawToken2)
	require.NotEqual(t, hash1, hash2)

	// Verify that hash1 matches SHA-256 of rawToken1
	expectedHash := sha256.Sum256([]byte(rawToken1))
	require.Equal(t, expectedHash[:], hash1)
}

func TestHashToken(t *testing.T) {
	t.Run("computes expected sha256", func(t *testing.T) {
		raw := hex.EncodeToString([]byte("01234567890123456789012345678901"))
		hash, err := HashToken(raw)
		require.NoError(t, err)
		expected := sha256.Sum256([]byte(raw))
		require.Equal(t, expected[:], hash)
	})

	t.Run("rejects token that is too short", func(t *testing.T) {
		_, err := HashToken("too-short")
		require.Error(t, err)
	})

	t.Run("trims leading and trailing spaces", func(t *testing.T) {
		raw := hex.EncodeToString([]byte("01234567890123456789012345678901"))
		hash1, err1 := HashToken("  " + raw + "  ")
		require.NoError(t, err1)
		hash2, err2 := HashToken(raw)
		require.NoError(t, err2)
		require.Equal(t, hash1, hash2)
	})
}
