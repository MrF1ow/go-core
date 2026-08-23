package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// GenerateApiKey tests
// ---------------------------------------------------------------------------

func TestGenerateApiKeyAdmin(t *testing.T) {
	rawKey, keyHash, keyPrefix, keySuffix, err := GenerateApiKey(KeyTypeAdmin)
	if err != nil {
		t.Fatalf("GenerateApiKey(admin) error: %v", err)
	}

	// Raw key should start with "ak_".
	if !strings.HasPrefix(rawKey, adminKeyPrefix) {
		t.Errorf("admin key should start with %q, got %q", adminKeyPrefix, rawKey[:4])
	}

	// Raw key length: "ak_" (3) + 48 hex chars = 51.
	expectedLen := len(adminKeyPrefix) + keyRandomBytes*2
	if len(rawKey) != expectedLen {
		t.Errorf("admin key length = %d, want %d", len(rawKey), expectedLen)
	}

	// Hash should be a valid SHA-256 hex digest.
	if len(keyHash) != 64 {
		t.Errorf("hash length = %d, want 64", len(keyHash))
	}

	// Hash should match SHA-256 of raw key.
	h := sha256.Sum256([]byte(rawKey))
	expected := hex.EncodeToString(h[:])
	if keyHash != expected {
		t.Error("hash does not match SHA-256 of raw key")
	}

	// Prefix should be first 12 chars.
	if keyPrefix != rawKey[:12] {
		t.Errorf("keyPrefix = %q, want %q", keyPrefix, rawKey[:12])
	}

	// Suffix should be last 4 chars.
	if keySuffix != rawKey[len(rawKey)-4:] {
		t.Errorf("keySuffix = %q, want %q", keySuffix, rawKey[len(rawKey)-4:])
	}
}

func TestGenerateApiKeyApp(t *testing.T) {
	rawKey, _, _, _, err := GenerateApiKey(KeyTypeApp)
	if err != nil {
		t.Fatalf("GenerateApiKey(app) error: %v", err)
	}

	// App key should start with "apk_".
	if !strings.HasPrefix(rawKey, appKeyPrefix) {
		t.Errorf("app key should start with %q, got %q", appKeyPrefix, rawKey[:4])
	}

	// Raw key length: "apk_" (4) + 48 hex chars = 52.
	expectedLen := len(appKeyPrefix) + keyRandomBytes*2
	if len(rawKey) != expectedLen {
		t.Errorf("app key length = %d, want %d", len(rawKey), expectedLen)
	}
}

func TestGenerateApiKeyUniqueness(t *testing.T) {
	// Generate two keys and verify they are different.
	raw1, hash1, _, _, _ := GenerateApiKey(KeyTypeAdmin)
	raw2, hash2, _, _, _ := GenerateApiKey(KeyTypeAdmin)

	if raw1 == raw2 {
		t.Error("two generated keys should not be identical")
	}
	if hash1 == hash2 {
		t.Error("two generated hashes should not be identical")
	}
}

// ---------------------------------------------------------------------------
// HashApiKey tests
// ---------------------------------------------------------------------------

func TestHashApiKey(t *testing.T) {
	input := "ak_1234567890abcdef1234567890abcdef1234567890abcdef"
	hash := HashApiKey(input)

	// Should be 64 hex chars (SHA-256).
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Should be deterministic.
	hash2 := HashApiKey(input)
	if hash != hash2 {
		t.Error("HashApiKey should be deterministic")
	}

	// Different input should produce different hash.
	hash3 := HashApiKey(input + "x")
	if hash == hash3 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestHashApiKeyEmpty(t *testing.T) {
	hash := HashApiKey("")
	// SHA-256 of empty string is a known value.
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Errorf("hash of empty string = %q, want %q", hash, expected)
	}
}

// ---------------------------------------------------------------------------
// Constants tests
// ---------------------------------------------------------------------------

func TestParseOptionalExpiresAt_AppEmptyIsForever(t *testing.T) {
	got, err := parseOptionalExpiresAt("", time.Now(), false)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("empty app expiry must be forever")
	}
}

func TestParseOptionalExpiresAt_AdminRequiredEmptyCreateErrors(t *testing.T) {
	_, err := parseOptionalExpiresAt("", time.Now(), true)
	if err == nil {
		t.Fatal("empty admin create expiry must fail")
	}
}

func TestParseOptionalExpiresAt_Future(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	got, err := parseOptionalExpiresAt("2026-09-01T15:04", now, false)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Year() != 2026 || got.Month() != 9 {
		t.Fatalf("got %v", got)
	}
}

func TestParseOptionalExpiresAt_PastRejected(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	_, err := parseOptionalExpiresAt("2026-01-01T00:00", now, false)
	if err == nil {
		t.Fatal("past expiry must fail")
	}
}

func TestParseOptionalExpiresAtKeeping_UnchangedPastKept(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	current := time.Date(2026, 1, 1, 15, 4, 30, 0, time.UTC)
	got, err := parseOptionalExpiresAtKeeping("2026-01-01T15:04", now, &current, true)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.Equal(current) {
		t.Fatalf("got %v, want stored timestamp %v", got, current)
	}
}

func TestParseOptionalExpiresAtKeeping_AppEmptyIsForever(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	current := time.Date(2026, 1, 1, 15, 4, 0, 0, time.UTC)
	got, err := parseOptionalExpiresAtKeeping("", now, &current, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("empty app expiry must be forever")
	}
}

func TestParseOptionalExpiresAtKeeping_AdminRequiredEmptyEditKeepsCurrent(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	current := time.Date(2026, 1, 1, 15, 4, 0, 0, time.UTC)
	got, err := parseOptionalExpiresAtKeeping("", now, &current, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != &current {
		t.Fatalf("got %v, want stored pointer", got)
	}
}

func TestParseOptionalExpiresAtKeeping_NewPastRejected(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	current := time.Date(2026, 1, 1, 15, 4, 0, 0, time.UTC)
	_, err := parseOptionalExpiresAtKeeping("2026-02-01T00:00", now, &current, true)
	if err == nil {
		t.Fatal("a different past expiry must fail")
	}
}

func TestParseOptionalExpiresAtKeeping_FutureChange(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	current := time.Date(2026, 1, 1, 15, 4, 0, 0, time.UTC)
	got, err := parseOptionalExpiresAtKeeping("2026-09-01T15:04", now, &current, true)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Year() != 2026 || got.Month() != 9 {
		t.Fatalf("got %v", got)
	}
}

func TestKeyTypeConstants(t *testing.T) {
	if KeyTypeAdmin != "admin" {
		t.Errorf("KeyTypeAdmin = %q, want %q", KeyTypeAdmin, "admin")
	}
	if KeyTypeApp != "app" {
		t.Errorf("KeyTypeApp = %q, want %q", KeyTypeApp, "app")
	}
}
