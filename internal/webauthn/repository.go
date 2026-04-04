package webauthn

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JedidiahDigital/go-core/internal/safeconv"
	"github.com/JedidiahDigital/go-core/internal/sqlcgen"
	"github.com/JedidiahDigital/go-core/pkg/models"
	"github.com/google/uuid"
)

// ErrNotFound is returned when a credential lookup or scoped update matches no rows.
var ErrNotFound = errors.New("record not found")

// Repository provides database access for WebAuthn credentials.
type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewRepository creates a new WebAuthn repository backed by pgx/SQLC.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// CreateCredential stores a new WebAuthn credential in the database.
func (r *Repository) CreateCredential(cred *models.WebAuthnCredential) error {
	if cred.ID == uuid.Nil {
		cred.ID = uuid.New()
	}
	if cred.CreatedAt.IsZero() {
		cred.CreatedAt = time.Now().UTC()
	}
	return r.queries.CreateWebAuthnCredential(context.Background(), sqlcgen.CreateWebAuthnCredentialParams{
		ID:              cred.ID,
		UserID:          uuidToPgtype(cred.UserID),
		AppID:           uuidToPgtype(cred.AppID),
		AdminID:         uuidToPgtype(cred.AdminID),
		CredentialID:    cred.CredentialID,
		PublicKey:       cred.PublicKey,
		AttestationType: strPtr(cred.AttestationType),
		Aaguid:          cred.AAGUID,
		SignCount:       safeconv.Uint32ToInt32(cred.SignCount),
		Name:            strPtr(cred.Name),
		Transports:      strPtr(cred.Transports),
		BackupEligible:  cred.BackupEligible,
		BackupState:     cred.BackupState,
		LastUsedAt:      timeToPgtypeTimestamptz(cred.LastUsedAt),
		CreatedAt:       cred.CreatedAt,
	})
}

// GetCredentialsByUserID returns all WebAuthn credentials for a user.
func (r *Repository) GetCredentialsByUserID(userID uuid.UUID) ([]models.WebAuthnCredential, error) {
	rows, err := r.queries.GetWebAuthnCredentialsByUserID(context.Background(), uuidToPgtypeVal(userID))
	if err != nil {
		return nil, err
	}
	return toModelCredentials(rows), nil
}

// GetCredentialsByUserAndApp returns all WebAuthn credentials for a user within a specific app.
func (r *Repository) GetCredentialsByUserAndApp(userID, appID uuid.UUID) ([]models.WebAuthnCredential, error) {
	rows, err := r.queries.GetWebAuthnCredentialsByUserAndApp(context.Background(), sqlcgen.GetWebAuthnCredentialsByUserAndAppParams{
		UserID: uuidToPgtypeVal(userID),
		AppID:  uuidToPgtypeVal(appID),
	})
	if err != nil {
		return nil, err
	}
	return toModelCredentials(rows), nil
}

// GetCredentialByCredentialID looks up a credential by its WebAuthn credential ID bytes.
func (r *Repository) GetCredentialByCredentialID(credentialID []byte) (*models.WebAuthnCredential, error) {
	row, err := r.queries.GetWebAuthnCredentialByCredentialID(context.Background(), credentialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	cred := toModelCredential(row)
	return &cred, nil
}

// GetCredentialByAppAndCredentialID looks up a credential by app ID and WebAuthn credential ID.
func (r *Repository) GetCredentialByAppAndCredentialID(appID uuid.UUID, credentialID []byte) (*models.WebAuthnCredential, error) {
	row, err := r.queries.GetWebAuthnCredentialByAppAndCredentialID(context.Background(), sqlcgen.GetWebAuthnCredentialByAppAndCredentialIDParams{
		AppID:        uuidToPgtypeVal(appID),
		CredentialID: credentialID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	cred := toModelCredential(row)
	return &cred, nil
}

// GetCredentialByID looks up a credential by its primary key UUID.
func (r *Repository) GetCredentialByID(id uuid.UUID) (*models.WebAuthnCredential, error) {
	row, err := r.queries.GetWebAuthnCredentialByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	cred := toModelCredential(row)
	return &cred, nil
}

// UpdateCredentialSignCount updates the sign count and last used timestamp.
func (r *Repository) UpdateCredentialSignCount(id uuid.UUID, signCount uint32) error {
	return r.queries.UpdateWebAuthnCredentialSignCount(context.Background(), sqlcgen.UpdateWebAuthnCredentialSignCountParams{
		ID:        id,
		SignCount: safeconv.Uint32ToInt32(signCount),
	})
}

// DeleteCredential removes a credential, scoped to the owning user for safety.
func (r *Repository) DeleteCredential(id, userID uuid.UUID) error {
	return r.queries.DeleteWebAuthnCredential(context.Background(), sqlcgen.DeleteWebAuthnCredentialParams{
		ID:     id,
		UserID: uuidToPgtypeVal(userID),
	})
}

// CountCredentialsByUserAndApp returns the number of passkeys a user has for an app.
func (r *Repository) CountCredentialsByUserAndApp(userID, appID uuid.UUID) (int64, error) {
	return r.queries.CountWebAuthnCredentialsByUserAndApp(context.Background(), sqlcgen.CountWebAuthnCredentialsByUserAndAppParams{
		UserID: uuidToPgtypeVal(userID),
		AppID:  uuidToPgtypeVal(appID),
	})
}

// RenameCredential updates the user-friendly name of a credential.
func (r *Repository) RenameCredential(id, userID uuid.UUID, name string) error {
	rowsAffected, err := r.queries.RenameWebAuthnCredential(context.Background(), sqlcgen.RenameWebAuthnCredentialParams{
		ID:     id,
		UserID: uuidToPgtypeVal(userID),
		Name:   &name,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ============================================================
// Admin Account Passkey Operations
// ============================================================

// GetCredentialsByAdminID returns all WebAuthn credentials for an admin account.
func (r *Repository) GetCredentialsByAdminID(adminID uuid.UUID) ([]models.WebAuthnCredential, error) {
	rows, err := r.queries.GetWebAuthnCredentialsByAdminID(context.Background(), uuidToPgtypeVal(adminID))
	if err != nil {
		return nil, err
	}
	return toModelCredentials(rows), nil
}

// DeleteAdminCredential removes a credential scoped to the owning admin for safety.
func (r *Repository) DeleteAdminCredential(id, adminID uuid.UUID) error {
	return r.queries.DeleteWebAuthnAdminCredential(context.Background(), sqlcgen.DeleteWebAuthnAdminCredentialParams{
		ID:      id,
		AdminID: uuidToPgtypeVal(adminID),
	})
}

// RenameAdminCredential updates the user-friendly name of an admin passkey.
func (r *Repository) RenameAdminCredential(id, adminID uuid.UUID, name string) error {
	rowsAffected, err := r.queries.RenameWebAuthnAdminCredential(context.Background(), sqlcgen.RenameWebAuthnAdminCredentialParams{
		ID:      id,
		AdminID: uuidToPgtypeVal(adminID),
		Name:    &name,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetCredentialByAdminAndCredentialID looks up a credential by admin ID and WebAuthn credential ID.
func (r *Repository) GetCredentialByAdminAndCredentialID(adminID uuid.UUID, credentialID []byte) (*models.WebAuthnCredential, error) {
	row, err := r.queries.GetWebAuthnCredentialByAdminAndCredentialID(context.Background(), sqlcgen.GetWebAuthnCredentialByAdminAndCredentialIDParams{
		AdminID:      uuidToPgtypeVal(adminID),
		CredentialID: credentialID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	cred := toModelCredential(row)
	return &cred, nil
}

// ============================================================
// Type conversion helpers
// ============================================================

// toModelCredentials converts a slice of SQLC-generated rows to the shared model type.
func toModelCredentials(rows []sqlcgen.WebauthnCredential) []models.WebAuthnCredential {
	out := make([]models.WebAuthnCredential, len(rows))
	for i, row := range rows {
		out[i] = toModelCredential(row)
	}
	return out
}

// toModelCredential converts a single SQLC-generated row to the shared model type.
func toModelCredential(row sqlcgen.WebauthnCredential) models.WebAuthnCredential {
	return models.WebAuthnCredential{
		ID:              row.ID,
		UserID:          pgtypeToUUIDPtr(row.UserID),
		AppID:           pgtypeToUUIDPtr(row.AppID),
		AdminID:         pgtypeToUUIDPtr(row.AdminID),
		CredentialID:    row.CredentialID,
		PublicKey:       row.PublicKey,
		AttestationType: derefStr(row.AttestationType),
		AAGUID:          row.Aaguid,
		SignCount:       safeconv.Int32ToUint32(row.SignCount),
		Name:            derefStr(row.Name),
		Transports:      derefStr(row.Transports),
		BackupEligible:  row.BackupEligible,
		BackupState:     row.BackupState,
		LastUsedAt:      pgtypeTimestamptzToTimePtr(row.LastUsedAt),
		CreatedAt:       row.CreatedAt,
	}
}

// uuidToPgtype converts a *uuid.UUID to a pgtype.UUID (nullable).
func uuidToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

// uuidToPgtypeVal converts a uuid.UUID value to a pgtype.UUID (always valid).
func uuidToPgtypeVal(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// pgtypeToUUIDPtr converts a pgtype.UUID to *uuid.UUID.
func pgtypeToUUIDPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

// timeToPgtypeTimestamptz converts a *time.Time to pgtype.Timestamptz.
func timeToPgtypeTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// pgtypeTimestamptzToTimePtr converts a pgtype.Timestamptz to *time.Time.
func pgtypeTimestamptzToTimePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	return &ts.Time
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
