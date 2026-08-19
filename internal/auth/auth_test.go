package auth_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/momtazularefin/truebearing/internal/auth"
)

func TestGenerateAPIKeyFormatAndLength(t *testing.T) {
	plainKey, keyHash, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey returned unexpected error: %v", err)
	}

	if !strings.HasPrefix(plainKey, auth.KeyPrefix) {
		t.Errorf("expected key to start with prefix %q, got %q", auth.KeyPrefix, plainKey)
	}

	// 32 bytes in unpadded base32 = 52 characters + 3 ("tb_") = 55 characters
	if len(plainKey) != 55 {
		t.Errorf("expected key length of 55, got %d (key: %q)", len(plainKey), plainKey)
	}

	if err := auth.ValidateKeyFormat(plainKey); err != nil {
		t.Errorf("expected generated key to pass format validation, got error: %v", err)
	}

	// Verify SHA-256 hash
	expectedHashBytes := sha256.Sum256([]byte(plainKey))
	expectedHash := hex.EncodeToString(expectedHashBytes[:])
	if keyHash != expectedHash {
		t.Errorf("expected hash %q, got %q", expectedHash, keyHash)
	}
	if len(keyHash) != 64 {
		t.Errorf("expected SHA-256 hex hash length 64, got %d", len(keyHash))
	}
}

func TestGenerateAPIKeyUniqueness(t *testing.T) {
	const count = 1000
	seenKeys := make(map[string]bool, count)
	seenHashes := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		plain, hash, err := auth.GenerateAPIKey()
		if err != nil {
			t.Fatalf("iteration %d: GenerateAPIKey failed: %v", i, err)
		}
		if seenKeys[plain] {
			t.Fatalf("iteration %d: duplicate plain key generated: %s", i, plain)
		}
		if seenHashes[hash] {
			t.Fatalf("iteration %d: duplicate hash generated: %s", i, hash)
		}
		seenKeys[plain] = true
		seenHashes[hash] = true
	}
}

func TestValidateKeyFormatRejections(t *testing.T) {
	testCases := []struct {
		name string
		key  string
	}{
		{"empty string", ""},
		{"missing prefix", "MZXW6YTBOI========"},
		{"wrong prefix", "ak_MZXW6YTBOI"},
		{"too short", "tb_MZXW6YTBOI"},
		{"too long", "tb_MZXW6YTBOI" + strings.Repeat("A", 60)},
		{"invalid base32 chars", "tb_1890189018901890189018901890189018901890189018901890"},
		{"with padding", "tb_MZXW6YTBOI======"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := auth.ValidateKeyFormat(tc.key)
			if err == nil {
				t.Errorf("expected error for %s (%q), got nil", tc.name, tc.key)
			}
		})
	}
}
