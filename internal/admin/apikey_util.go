package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/operator"
)

const (
	// KeyTypeAdmin is the key type for admin API keys (/admin/* routes).
	KeyTypeAdmin = "admin"
	// KeyTypeApp is the key type for per-application API keys.
	KeyTypeApp = "app"

	// adminKeyPrefix is prepended to admin keys for visual identification.
	adminKeyPrefix = "ak_"
	// appKeyPrefix is prepended to app keys for visual identification.
	appKeyPrefix = "apk_"

	// keyRandomBytes is the number of random bytes (24 bytes = 48 hex chars = 192 bits entropy).
	keyRandomBytes = 24
)

// GenerateApiKey creates a new random API key for the given type.
// Returns (rawKey, hash, prefix, suffix).
// The rawKey should be shown to the user once and never stored.
func GenerateApiKey(keyType string) (rawKey, keyHash, keyPrefix, keySuffix string, err error) {
	// Generate random bytes
	randomBytes := make([]byte, keyRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Build the raw key with prefix
	prefix := adminKeyPrefix
	if keyType == KeyTypeApp {
		prefix = appKeyPrefix
	}
	rawKey = prefix + hex.EncodeToString(randomBytes)

	// SHA-256 hash for storage
	keyHash = HashApiKey(rawKey)

	// Extract display prefix (first 12 chars including the type prefix)
	keyPrefix = rawKey[:12]

	// Extract display suffix (last 4 chars)
	keySuffix = rawKey[len(rawKey)-4:]

	return rawKey, keyHash, keyPrefix, keySuffix, nil
}

// HashApiKey computes the SHA-256 hex digest of a raw API key.
func HashApiKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

func parseOptionalExpiresAt(raw string, now time.Time) (*time.Time, error) {
	return parseOptionalExpiresAtKeeping(raw, now, nil)
}

// parseOptionalExpiresAtKeeping treats a posted datetime-local value that still
// matches the stored instant (minute precision, same format as the edit form)
// as "leave expiry alone". That lets operators change name, description, or
// role on a key whose expiry has already passed. Clearing the field is forever.
// Any other posted value must be strictly after now, same as create.
func parseOptionalExpiresAtKeeping(raw string, now time.Time, current *time.Time) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if current != nil && raw == formatExpiresAtLocal(current) {
		return current, nil
	}
	t, err := time.Parse("2006-01-02T15:04", raw)
	if err != nil {
		return nil, err
	}
	if !t.After(now) {
		return nil, errExpiresInPast
	}
	return &t, nil
}

func formatExpiresAtLocal(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04")
}

func uuidPtrString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

type operatorRoleOption struct {
	ID   uuid.UUID
	Name string
}

func operatorRoleOptions() []operatorRoleOption {
	return []operatorRoleOption{
		{ID: operator.RoleIDViewer, Name: operator.RoleViewer},
		{ID: operator.RoleIDSupport, Name: operator.RoleSupport},
		{ID: operator.RoleIDAdmin, Name: operator.RoleAdmin},
		{ID: operator.RoleIDSuperadmin, Name: operator.RoleSuperadmin},
	}
}

var (
	errExpiresInPast = errors.New("expiration date must be in the future")
)
