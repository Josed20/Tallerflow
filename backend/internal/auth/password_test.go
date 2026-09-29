package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

const testPassword = "Correct horse battery staple 7!"

func TestPasswordHasherRoundTrip(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())

	encoded, err := hasher.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("Hash() = %q, want Argon2id PHC string with the required parameters", encoded)
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		t.Fatalf("Hash() PHC field count = %d, want 6", len(parts))
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		t.Fatalf("Hash() salt is not valid unpadded base64: %v", err)
	}
	if len(salt) != 16 {
		t.Fatalf("Hash() salt length = %d bytes, want 16", len(salt))
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		t.Fatalf("Hash() key is not valid unpadded base64: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("Hash() key length = %d bytes, want 32", len(key))
	}

	ok, err := hasher.Verify(encoded, testPassword)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("Verify() = false, want true for the password that produced the hash")
	}
}

func TestPasswordHasherRejectsWrongPassword(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	encoded, err := hasher.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	ok, err := hasher.Verify(encoded, "This is not the password 8!")
	if err != nil {
		t.Fatalf("Verify() error = %v, want a clean mismatch", err)
	}
	if ok {
		t.Fatal("Verify() = true, want false for a wrong password")
	}
}

func TestPasswordHasherRejectsMalformedHash(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())

	ok, err := hasher.Verify("not-a-phc-password-hash", testPassword)
	if err == nil {
		t.Fatal("Verify() error = nil, want malformed hash error")
	}
	if ok {
		t.Fatal("Verify() = true, want false for a malformed hash")
	}
}

func TestPasswordHasherRejectsInputLargerThanOneMiB(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	tooLarge := strings.Repeat("a", (1<<20)+1)

	if _, err := hasher.Hash(tooLarge); err == nil {
		t.Fatal("Hash() error = nil, want input-size error for password larger than 1 MiB")
	}
}

func TestPasswordHasherRejectsResourceExhaustingPHCParameters(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	validSalt := base64.RawStdEncoding.EncodeToString(make([]byte, 16))
	validKey := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	oversizedSalt := base64.RawStdEncoding.EncodeToString(make([]byte, maxDecodedSaltBytes+1))
	oversizedKey := base64.RawStdEncoding.EncodeToString(make([]byte, maxDecodedKeyBytes+1))

	tests := []struct {
		name    string
		encoded string
	}{
		{name: "memory", encoded: "$argon2id$v=19$m=65537,t=3,p=2$" + validSalt + "$" + validKey},
		{name: "iterations", encoded: "$argon2id$v=19$m=65536,t=4,p=2$" + validSalt + "$" + validKey},
		{name: "parallelism", encoded: "$argon2id$v=19$m=65536,t=3,p=3$" + validSalt + "$" + validKey},
		{name: "encoded hash length", encoded: "$argon2id$v=19$m=65536,t=3,p=2$" + strings.Repeat("A", maxEncodedHashBytes)},
		{name: "decoded salt length", encoded: "$argon2id$v=19$m=65536,t=3,p=2$" + oversizedSalt + "$" + validKey},
		{name: "decoded key length", encoded: "$argon2id$v=19$m=65536,t=3,p=2$" + validSalt + "$" + oversizedKey},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := hasher.Verify(tc.encoded, testPassword)
			if !errors.Is(err, ErrMalformedPasswordHash) {
				t.Fatalf("Verify() error = %v, want ErrMalformedPasswordHash", err)
			}
			if ok {
				t.Fatal("Verify() = true, want false for resource-exhausting PHC input")
			}
		})
	}
}
