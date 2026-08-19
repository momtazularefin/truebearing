// Package auth provides cryptographic API key generation, validation, and hashing.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	// KeyPrefix is the standard prefix for TrueBearing API keys.
	KeyPrefix = "tb_"
	// KeyEntropyBytes is 32 bytes (256 bits of entropy).
	KeyEntropyBytes = 32
)

var (
	// ErrInvalidKeyFormat is returned when an API key does not match the expected format.
	ErrInvalidKeyFormat = errors.New("invalid api key format")
	// ErrKeyEntropySource is returned if the cryptographic entropy generator fails.
	ErrKeyEntropySource = errors.New("failed to read secure random bytes")

	// base32Raw is unpadded RFC 4648 base32 encoding.
	base32Raw = base32.StdEncoding.WithPadding(base32.NoPadding)
)

// GenerateAPIKey generates a cryptographically secure 256-bit API key.
// It returns the plaintext key (to be displayed to the user exactly once)
// and its SHA-256 hash (to be stored in the database).
//
// Per D009, keys are full-entropy 256-bit random tokens, so SHA-256 provides
// collision and preimage resistance without incurring the 50-100ms CPU latency
// penalty of a slow KDF like bcrypt or argon2 during high-throughput verification.
func GenerateAPIKey() (plainKey string, keyHash string, err error) {
	var raw [KeyEntropyBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrKeyEntropySource, err)
	}

	encoded := base32Raw.EncodeToString(raw[:])
	plainKey = KeyPrefix + encoded
	keyHash = HashAPIKey(plainKey)

	return plainKey, keyHash, nil
}

// HashAPIKey computes the hex-encoded SHA-256 hash of an API key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// ValidateKeyFormat verifies that the provided key conforms to the "tb_<base32>" format.
func ValidateKeyFormat(key string) error {
	if !strings.HasPrefix(key, KeyPrefix) {
		return ErrInvalidKeyFormat
	}
	payload := strings.TrimPrefix(key, KeyPrefix)
	decoded, err := base32Raw.DecodeString(payload)
	if err != nil || len(decoded) != KeyEntropyBytes {
		return ErrInvalidKeyFormat
	}
	return nil
}
