package twofa

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JedidiahDigital/go-core/internal/sqlcgen"
	"github.com/JedidiahDigital/go-core/pkg/models"
	"github.com/google/uuid"
)

// TrustedDeviceRepository handles all database operations for TrustedDevice records.
type TrustedDeviceRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewTrustedDeviceRepository creates a new TrustedDeviceRepository backed by pgx/SQLC.
func NewTrustedDeviceRepository(pool *pgxpool.Pool) *TrustedDeviceRepository {
	return &TrustedDeviceRepository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// hashToken returns the SHA-256 hex hash of a plaintext device token.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}

// Create persists a new TrustedDevice record. The caller is responsible for hashing
// the device token before storing it (use hashToken).
func (r *TrustedDeviceRepository) Create(device *models.TrustedDevice) error {
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}
	now := time.Now().UTC()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	if device.LastUsedAt.IsZero() {
		device.LastUsedAt = now
	}
	return r.queries.CreateTrustedDevice(context.Background(), sqlcgen.CreateTrustedDeviceParams{
		ID:         device.ID,
		UserID:     device.UserID,
		AppID:      device.AppID,
		TokenHash:  device.TokenHash,
		Name:       strPtr(device.Name),
		UserAgent:  strPtr(device.UserAgent),
		IpAddress:  strPtr(device.IPAddress),
		LastUsedAt: device.LastUsedAt,
		ExpiresAt:  device.ExpiresAt,
		CreatedAt:  device.CreatedAt,
	})
}

// FindByTokenHash looks up a trusted device by its hashed token.
// Returns nil, nil when not found.
func (r *TrustedDeviceRepository) FindByTokenHash(tokenHash string) (*models.TrustedDevice, error) {
	row, err := r.queries.FindTrustedDeviceByTokenHash(context.Background(), tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d := toModelTrustedDevice(row)
	return &d, nil
}

// FindByUserAndApp returns all non-expired trusted devices for a given user + app.
func (r *TrustedDeviceRepository) FindByUserAndApp(userID, appID uuid.UUID) ([]models.TrustedDevice, error) {
	rows, err := r.queries.FindTrustedDevicesByUserAndApp(context.Background(), sqlcgen.FindTrustedDevicesByUserAndAppParams{
		UserID: userID,
		AppID:  appID,
	})
	if err != nil {
		return nil, err
	}
	return toModelTrustedDevices(rows), nil
}

// FindByID returns a single trusted device by its primary key.
// Returns nil, nil when not found.
func (r *TrustedDeviceRepository) FindByID(id uuid.UUID) (*models.TrustedDevice, error) {
	row, err := r.queries.FindTrustedDeviceByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d := toModelTrustedDevice(row)
	return &d, nil
}

// TouchLastUsed updates last_used_at to the current time for a device.
func (r *TrustedDeviceRepository) TouchLastUsed(id uuid.UUID) error {
	return r.queries.TouchTrustedDeviceLastUsed(context.Background(), id)
}

// DeleteByID removes a single trusted device by its primary key.
func (r *TrustedDeviceRepository) DeleteByID(id uuid.UUID) error {
	return r.queries.DeleteTrustedDeviceByID(context.Background(), id)
}

// DeleteAllForUser removes all trusted devices for a user in a given app.
func (r *TrustedDeviceRepository) DeleteAllForUser(userID, appID uuid.UUID) error {
	return r.queries.DeleteAllTrustedDevicesForUser(context.Background(), sqlcgen.DeleteAllTrustedDevicesForUserParams{
		UserID: userID,
		AppID:  appID,
	})
}

// DeleteByUserAppAndUserAgent removes all existing device records (expired or active)
// for a given user + app + user-agent combination. Used for deduplication before
// inserting a fresh trusted device row on repeated "Remember this device" logins.
func (r *TrustedDeviceRepository) DeleteByUserAppAndUserAgent(userID, appID uuid.UUID, userAgent string) error {
	return r.queries.DeleteTrustedDevicesByUserAppAndUserAgent(context.Background(), sqlcgen.DeleteTrustedDevicesByUserAppAndUserAgentParams{
		UserID:    userID,
		AppID:     appID,
		UserAgent: &userAgent,
	})
}

// DeleteExpired removes all trusted devices whose ExpiresAt is in the past.
// This can be called periodically as a cleanup job.
func (r *TrustedDeviceRepository) DeleteExpired() (int64, error) {
	return r.queries.DeleteExpiredTrustedDevices(context.Background())
}

// CountByUserAndApp returns the number of active (non-expired) trusted devices for a user.
func (r *TrustedDeviceRepository) CountByUserAndApp(userID, appID uuid.UUID) (int64, error) {
	return r.queries.CountTrustedDevicesByUserAndApp(context.Background(), sqlcgen.CountTrustedDevicesByUserAndAppParams{
		UserID: userID,
		AppID:  appID,
	})
}

// CountAllActive returns the total count of non-expired trusted devices across all apps/users.
// Used for dashboard stats.
func (r *TrustedDeviceRepository) CountAllActive() (int64, error) {
	return r.queries.CountAllActiveTrustedDevices(context.Background())
}

// DeleteAllForUserAllApps removes all trusted devices for a user across all apps.
// Used by admin to revoke all devices regardless of app scope.
func (r *TrustedDeviceRepository) DeleteAllForUserAllApps(userID uuid.UUID) error {
	return r.queries.DeleteAllTrustedDevicesForUserAllApps(context.Background(), userID)
}

// FindAllForUser returns all non-expired trusted devices for a user across all apps.
// Used by the admin panel to list and revoke devices.
func (r *TrustedDeviceRepository) FindAllForUser(userID uuid.UUID) ([]models.TrustedDevice, error) {
	rows, err := r.queries.FindAllTrustedDevicesForUser(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	return toModelTrustedDevices(rows), nil
}

// toModelTrustedDevices converts a slice of SQLC-generated rows to the shared model type.
func toModelTrustedDevices(rows []sqlcgen.TrustedDevice) []models.TrustedDevice {
	out := make([]models.TrustedDevice, len(rows))
	for i, row := range rows {
		out[i] = toModelTrustedDevice(row)
	}
	return out
}

// toModelTrustedDevice converts a single SQLC-generated row to the shared model type.
func toModelTrustedDevice(row sqlcgen.TrustedDevice) models.TrustedDevice {
	return models.TrustedDevice{
		ID:         row.ID,
		UserID:     row.UserID,
		AppID:      row.AppID,
		TokenHash:  row.TokenHash,
		Name:       derefStr(row.Name),
		UserAgent:  derefStr(row.UserAgent),
		IPAddress:  derefStr(row.IpAddress),
		LastUsedAt: row.LastUsedAt,
		ExpiresAt:  row.ExpiresAt,
		CreatedAt:  row.CreatedAt,
	}
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
