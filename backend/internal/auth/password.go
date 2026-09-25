package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	maxPasswordBytes    = 1 << 20
	maxEncodedHashBytes = 512
	maxArgonMemoryKiB   = 64 * 1024
	maxArgonIterations  = 3
	maxArgonParallelism = 2
	minDecodedSaltBytes = 8
	maxDecodedSaltBytes = 64
	minDecodedKeyBytes  = 16
	maxDecodedKeyBytes  = 64
	defaultArgonVersion = argon2.Version
)

var (
	ErrPasswordTooLong       = errors.New("password exceeds 1 MiB")
	ErrMalformedPasswordHash = errors.New("malformed password hash")
	ErrInvalidPasswordParams = errors.New("invalid password parameters")
)

type PasswordParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func DefaultPasswordParams() PasswordParams {
	return PasswordParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

type PasswordHasher struct {
	params PasswordParams
}

func NewPasswordHasher(params PasswordParams) PasswordHasher {
	return PasswordHasher{params: params}
}

func (h PasswordHasher) Hash(password string) (string, error) {
	if len(password) > maxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	if err := validatePasswordParams(h.params); err != nil {
		return "", err
	}

	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey(
		[]byte(password),
		salt,
		h.params.Iterations,
		h.params.Memory,
		h.params.Parallelism,
		h.params.KeyLength,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		defaultArgonVersion,
		h.params.Memory,
		h.params.Iterations,
		h.params.Parallelism,
		encodedSalt,
		encodedKey,
	), nil
}

func (h PasswordHasher) Verify(encoded, password string) (bool, error) {
	if len(password) > maxPasswordBytes {
		return false, ErrPasswordTooLong
	}

	params, salt, expected, err := parsePasswordHash(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parsePasswordHash(encoded string) (PasswordParams, []byte, []byte, error) {
	if len(encoded) == 0 || len(encoded) > maxEncodedHashBytes {
		return PasswordParams{}, nil, nil, ErrMalformedPasswordHash
	}

	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return PasswordParams{}, nil, nil, ErrMalformedPasswordHash
	}

	parameterParts := strings.Split(parts[3], ",")
	if len(parameterParts) != 3 {
		return PasswordParams{}, nil, nil, ErrMalformedPasswordHash
	}
	memory, err := parseUintParameter(parameterParts[0], "m=", 32)
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}
	iterations, err := parseUintParameter(parameterParts[1], "t=", 32)
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}
	parallelism, err := parseUintParameter(parameterParts[2], "p=", 8)
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}

	if len(parts[4]) > base64.RawStdEncoding.EncodedLen(maxDecodedSaltBytes) ||
		len(parts[5]) > base64.RawStdEncoding.EncodedLen(maxDecodedKeyBytes) {
		return PasswordParams{}, nil, nil, ErrMalformedPasswordHash
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("%w: invalid salt", ErrMalformedPasswordHash)
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("%w: invalid key", ErrMalformedPasswordHash)
	}

	params := PasswordParams{
		Memory:      uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
		SaltLength:  uint32(len(salt)),
		KeyLength:   uint32(len(key)),
	}
	if err := validatePasswordParams(params); err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("%w: %v", ErrMalformedPasswordHash, err)
	}
	return params, salt, key, nil
}

func parseUintParameter(value, prefix string, bitSize int) (uint64, error) {
	if !strings.HasPrefix(value, prefix) || len(value) == len(prefix) {
		return 0, ErrMalformedPasswordHash
	}
	parsed, err := strconv.ParseUint(strings.TrimPrefix(value, prefix), 10, bitSize)
	if err != nil {
		return 0, ErrMalformedPasswordHash
	}
	return parsed, nil
}

func validatePasswordParams(params PasswordParams) error {
	if params.Memory == 0 || params.Memory > maxArgonMemoryKiB ||
		params.Iterations == 0 || params.Iterations > maxArgonIterations ||
		params.Parallelism == 0 || params.Parallelism > maxArgonParallelism ||
		params.Memory < 8*uint32(params.Parallelism) ||
		params.SaltLength < minDecodedSaltBytes || params.SaltLength > maxDecodedSaltBytes ||
		params.KeyLength < minDecodedKeyBytes || params.KeyLength > maxDecodedKeyBytes {
		return ErrInvalidPasswordParams
	}
	return nil
}
