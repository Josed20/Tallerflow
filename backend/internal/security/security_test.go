package security

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	h, err := HashPassword("Una-clave-segura-2026")
	if err != nil || !VerifyPassword(h, "Una-clave-segura-2026") || VerifyPassword(h, "incorrecta") {
		t.Fatal("argon2id round trip failed")
	}
}

func TestTokenStoresOnlyHash(t *testing.T) {
	raw, hash, err := Token()
	if err != nil || raw == "" || len(hash) != 32 || string(hash) == raw {
		t.Fatal("invalid token")
	}
}
