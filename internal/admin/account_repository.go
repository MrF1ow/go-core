package admin

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JedidiahDigital/go-core/internal/sqlcgen"
	"github.com/JedidiahDigital/go-core/pkg/models"
)

// AccountRepository handles database operations for admin accounts
type AccountRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewAccountRepository creates a new AccountRepository backed by pgx/SQLC.
func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// Create stores a new admin account in the database.
// The account pointer is updated with the generated ID and timestamps.
func (r *AccountRepository) Create(account *models.AdminAccount) error {
	row, err := r.queries.CreateAdminAccount(context.Background(), sqlcgen.CreateAdminAccountParams{
		Username:     account.Username,
		Email:        stringToPtr(account.Email),
		PasswordHash: account.PasswordHash,
	})
	if err != nil {
		return err
	}
	applyRowToModel(account, row)
	return nil
}

// GetByUsername retrieves an admin account by username
func (r *AccountRepository) GetByUsername(username string) (*models.AdminAccount, error) {
	row, err := r.queries.GetAdminAccountByUsername(context.Background(), username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m := rowToModel(row)
	return &m, nil
}

// GetByUsernameOrEmail retrieves an admin account by username or email.
// This allows admins to log in using either their username or email address.
func (r *AccountRepository) GetByUsernameOrEmail(identifier string) (*models.AdminAccount, error) {
	row, err := r.queries.GetAdminAccountByUsernameOrEmail(context.Background(), identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m := rowToModel(row)
	return &m, nil
}

// GetByID retrieves an admin account by ID
func (r *AccountRepository) GetByID(id string) (*models.AdminAccount, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetAdminAccountByID(context.Background(), parsed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m := rowToModel(row)
	return &m, nil
}

// UpdateLastLogin sets the LastLoginAt timestamp for an admin account
func (r *AccountRepository) UpdateLastLogin(id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.UpdateAdminAccountLastLogin(context.Background(), parsed)
}

// ListAll retrieves all admin accounts ordered by creation date
func (r *AccountRepository) ListAll() ([]models.AdminAccount, error) {
	rows, err := r.queries.ListAllAdminAccounts(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]models.AdminAccount, len(rows))
	for i, row := range rows {
		out[i] = rowToModel(row)
	}
	return out, nil
}

// Count returns the total number of admin accounts
func (r *AccountRepository) Count() (int64, error) {
	return r.queries.CountAdminAccounts(context.Background())
}

// DeleteByID removes an admin account by ID
func (r *AccountRepository) DeleteByID(id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.DeleteAdminAccountByID(context.Background(), parsed)
}

// UpdateEmail sets the email address for an admin account
func (r *AccountRepository) UpdateEmail(id, email string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.UpdateAdminAccountEmail(context.Background(), sqlcgen.UpdateAdminAccountEmailParams{
		ID:    parsed,
		Email: &email,
	})
}

// UpdatePassword updates the password hash for an admin account
func (r *AccountRepository) UpdatePassword(id, passwordHash string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.UpdateAdminAccountPassword(context.Background(), sqlcgen.UpdateAdminAccountPasswordParams{
		ID:           parsed,
		PasswordHash: passwordHash,
	})
}

// Enable2FA activates two-factor authentication for an admin account.
// It sets the method, secret, and recovery codes in a single update.
func (r *AccountRepository) Enable2FA(id, method, secret string, recoveryCodes []byte) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.EnableAdminAccount2FA(context.Background(), sqlcgen.EnableAdminAccount2FAParams{
		ID:                 parsed,
		TwoFaMethod:        &method,
		TwoFaSecret:        &secret,
		TwoFaRecoveryCodes: recoveryCodes,
	})
}

// Disable2FA deactivates two-factor authentication for an admin account,
// clearing all 2FA-related fields.
func (r *AccountRepository) Disable2FA(id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.DisableAdminAccount2FA(context.Background(), parsed)
}

// UpdateRecoveryCodes replaces the recovery codes for an admin account.
func (r *AccountRepository) UpdateRecoveryCodes(id string, codes []byte) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.UpdateAdminAccountRecoveryCodes(context.Background(), sqlcgen.UpdateAdminAccountRecoveryCodesParams{
		ID:                 parsed,
		TwoFaRecoveryCodes: codes,
	})
}

// UpdateMagicLinkEnabled sets the magic_link_enabled flag for an admin account.
func (r *AccountRepository) UpdateMagicLinkEnabled(id string, enabled bool) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.UpdateAdminAccountMagicLinkEnabled(context.Background(), sqlcgen.UpdateAdminAccountMagicLinkEnabledParams{
		ID:               parsed,
		MagicLinkEnabled: enabled,
	})
}

// GetByEmail retrieves an admin account by email address.
func (r *AccountRepository) GetByEmail(email string) (*models.AdminAccount, error) {
	row, err := r.queries.GetAdminAccountByEmail(context.Background(), &email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m := rowToModel(row)
	return &m, nil
}

// SetBackupEmail sets (or updates) the backup email for an admin account.
// It marks backup_email_verified = false since the new address hasn't been confirmed.
func (r *AccountRepository) SetBackupEmail(id, backupEmail string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.SetAdminAccountBackupEmail(context.Background(), sqlcgen.SetAdminAccountBackupEmailParams{
		ID:          parsed,
		BackupEmail: backupEmail,
	})
}

// ClearBackupEmail removes the backup email from an admin account.
func (r *AccountRepository) ClearBackupEmail(id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.ClearAdminAccountBackupEmail(context.Background(), parsed)
}

// ---------- helpers ----------

// stringToPtr converts a string to *string, returning nil for empty strings.
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ptrToString safely dereferences a *string, returning "" for nil.
func ptrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// rowToModel converts an SQLC-generated AdminAccount to the shared model type.
func rowToModel(row sqlcgen.AdminAccount) models.AdminAccount {
	var lastLogin *time.Time
	if row.LastLoginAt.Valid {
		lastLogin = &row.LastLoginAt.Time
	}

	return models.AdminAccount{
		ID:                  row.ID,
		Username:            row.Username,
		Email:               ptrToString(row.Email),
		PasswordHash:        row.PasswordHash,
		TwoFAEnabled:        row.TwoFaEnabled,
		TwoFAMethod:         ptrToString(row.TwoFaMethod),
		TwoFASecret:         ptrToString(row.TwoFaSecret),
		TwoFARecoveryCodes:  json.RawMessage(row.TwoFaRecoveryCodes),
		MagicLinkEnabled:    row.MagicLinkEnabled,
		BackupEmail:         row.BackupEmail,
		BackupEmailVerified: row.BackupEmailVerified,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
		LastLoginAt:         lastLogin,
	}
}

// applyRowToModel copies SQLC row fields onto an existing model pointer,
// used by Create to populate the caller's struct with generated values.
func applyRowToModel(m *models.AdminAccount, row sqlcgen.AdminAccount) {
	result := rowToModel(row)
	*m = result
}
