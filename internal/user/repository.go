package user

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/JedidiahDigital/go-core/internal/sqlcgen"
	"github.com/JedidiahDigital/go-core/pkg/models"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

func (r *Repository) CreateUser(user *models.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	row, err := r.queries.CreateUser(context.Background(), sqlcgen.CreateUserParams{
		ID:                  user.ID,
		AppID:               user.AppID,
		Email:               user.Email,
		PasswordHash:        user.PasswordHash,
		EmailVerified:       user.EmailVerified,
		IsActive:            user.IsActive,
		Name:                user.Name,
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		ProfilePicture:      user.ProfilePicture,
		Locale:              user.Locale,
		TwoFaEnabled:        user.TwoFAEnabled,
		TwoFaMethod:         user.TwoFAMethod,
		TwoFaSecret:         user.TwoFASecret,
		TwoFaRecoveryCodes:  json.RawMessage(user.TwoFARecoveryCodes),
		BackupEmail:         user.BackupEmail,
		BackupEmailVerified: user.BackupEmailVerified,
		TwoFaPreviousMethod: user.TwoFAPreviousMethod,
		TwoFaPreviousSecret: user.TwoFAPreviousSecret,
		PhoneNumber:         user.PhoneNumber,
		PhoneVerified:       user.PhoneVerified,
		LockedAt:            timePtrToTimestamptz(user.LockedAt),
		LockReason:          user.LockReason,
		LockExpiresAt:       timePtrToTimestamptz(user.LockExpiresAt),
		PasswordHistory:     []byte(user.PasswordHistory),
		PasswordChangedAt:   timePtrToTimestamptz(user.PasswordChangedAt),
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	})
	if err != nil {
		return err
	}
	applyRowToUser(user, row)
	return nil
}

func (r *Repository) GetUserByEmail(appID, email string) (*models.User, error) {
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetUserByEmail(context.Background(), sqlcgen.GetUserByEmailParams{
		AppID: appUUID,
		Email: email,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	m := rowToUser(row)
	return &m, nil
}

func (r *Repository) GetUserByID(id string) (*models.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetUserByID(context.Background(), uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	m := rowToUser(row)

	// Load social accounts (replaces GORM Preload)
	saRows, err := r.queries.GetSocialAccountsByUserID(context.Background(), uid)
	if err != nil {
		return &m, nil // non-fatal: return user without social accounts
	}
	m.SocialAccounts = toModelSocialAccounts(saRows)

	return &m, nil
}

func (r *Repository) UpdateUser(user *models.User) error {
	return r.queries.UpdateUser(context.Background(), sqlcgen.UpdateUserParams{
		ID:                  user.ID,
		AppID:               user.AppID,
		Email:               user.Email,
		PasswordHash:        user.PasswordHash,
		EmailVerified:       user.EmailVerified,
		IsActive:            user.IsActive,
		Name:                user.Name,
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		ProfilePicture:      user.ProfilePicture,
		Locale:              user.Locale,
		TwoFaEnabled:        user.TwoFAEnabled,
		TwoFaMethod:         user.TwoFAMethod,
		TwoFaSecret:         user.TwoFASecret,
		TwoFaRecoveryCodes:  json.RawMessage(user.TwoFARecoveryCodes),
		BackupEmail:         user.BackupEmail,
		BackupEmailVerified: user.BackupEmailVerified,
		TwoFaPreviousMethod: user.TwoFAPreviousMethod,
		TwoFaPreviousSecret: user.TwoFAPreviousSecret,
		PhoneNumber:         user.PhoneNumber,
		PhoneVerified:       user.PhoneVerified,
		LockedAt:            timePtrToTimestamptz(user.LockedAt),
		LockReason:          user.LockReason,
		LockExpiresAt:       timePtrToTimestamptz(user.LockExpiresAt),
		PasswordHistory:     []byte(user.PasswordHistory),
		PasswordChangedAt:   timePtrToTimestamptz(user.PasswordChangedAt),
	})
}

func (r *Repository) UpdateUserPassword(userID, hashedPassword string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateUserPassword(context.Background(), sqlcgen.UpdateUserPasswordParams{
		ID:           uid,
		PasswordHash: hashedPassword,
	})
}

// UpdateUserPasswordWithHistory sets password_hash, password_history, and password_changed_at atomically.
func (r *Repository) UpdateUserPasswordWithHistory(userID, hashedPassword string, history []byte) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	now := time.Now()
	return r.queries.UpdateUserPasswordWithHistory(context.Background(), sqlcgen.UpdateUserPasswordWithHistoryParams{
		ID:                uid,
		PasswordHash:      hashedPassword,
		PasswordHistory:   history,
		PasswordChangedAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
}

func (r *Repository) UpdateUserEmailVerified(userID string, verified bool) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateUserEmailVerified(context.Background(), sqlcgen.UpdateUserEmailVerifiedParams{
		ID:            uid,
		EmailVerified: verified,
	})
}

// 2FA related methods

// Enable2FA enables 2FA for a user and stores the secret and recovery codes.
// Defaults to TOTP method for backward compatibility.
func (r *Repository) Enable2FA(userID, secret, recoveryCodes string) error {
	return r.Enable2FAWithMethod(userID, secret, recoveryCodes, "totp")
}

// Enable2FAWithMethod enables 2FA for a user with a specific method ("totp" or "email").
func (r *Repository) Enable2FAWithMethod(userID, secret, recoveryCodes, method string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.Enable2FAWithMethod(context.Background(), sqlcgen.Enable2FAWithMethodParams{
		ID:                 uid,
		TwoFaSecret:        secret,
		TwoFaRecoveryCodes: json.RawMessage(recoveryCodes),
		TwoFaMethod:        method,
	})
}

// Disable2FA disables 2FA for a user
func (r *Repository) Disable2FA(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.Disable2FA(context.Background(), uid)
}

// UpdateRecoveryCodes updates the recovery codes for a user
func (r *Repository) UpdateRecoveryCodes(userID, recoveryCodes string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateRecoveryCodes(context.Background(), sqlcgen.UpdateRecoveryCodesParams{
		ID:                 uid,
		TwoFaRecoveryCodes: json.RawMessage(recoveryCodes),
	})
}

// DeleteUser deletes a user and all related data within a transaction.
// FK-constrained tables (social_accounts, user_roles) are deleted first to avoid
// "update or delete violates foreign key constraint" errors from NO ACTION constraints.
func (r *Repository) DeleteUser(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. social_accounts.user_id → users.id (NOT NULL, NO ACTION) — must delete first
	if _, err := tx.Exec(ctx, "DELETE FROM social_accounts WHERE user_id = $1", uid); err != nil {
		return err
	}
	// 2. user_roles.user_id → users.id (NOT NULL, NO ACTION) — must delete first
	if _, err := tx.Exec(ctx, "DELETE FROM user_roles WHERE user_id = $1", uid); err != nil {
		return err
	}
	// 3. trusted_devices — no FK constraint, but clean up
	if _, err := tx.Exec(ctx, "DELETE FROM trusted_devices WHERE user_id = $1", uid); err != nil {
		return err
	}
	// 4. web_authn_credentials — no FK constraint, but clean up
	if _, err := tx.Exec(ctx, "DELETE FROM web_authn_credentials WHERE user_id = $1", uid); err != nil {
		return err
	}
	// 5. activity_logs — no FK constraint, but clean up
	if _, err := tx.Exec(ctx, "DELETE FROM activity_logs WHERE user_id = $1", uid); err != nil {
		return err
	}
	// 6. Finally hard-delete the user row
	qtx := r.queries.WithTx(tx)
	if err := qtx.DeleteUserByID(ctx, uid); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateUserProfile updates user profile fields (name, first_name, last_name, profile_picture, locale).
// Accepts a map for backward compatibility with the service layer's partial-update logic.
// Reads current values first, merges provided updates, then writes all profile fields.
func (r *Repository) UpdateUserProfile(userID string, updates map[string]interface{}) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	// Read current user to get existing profile values
	row, err := r.queries.GetUserByID(context.Background(), uid)
	if err != nil {
		return err
	}

	// Start with current values
	name := row.Name
	firstName := row.FirstName
	lastName := row.LastName
	profilePicture := row.ProfilePicture
	locale := row.Locale

	// Merge provided updates
	if v, ok := updates["name"]; ok {
		name = v.(string)
	}
	if v, ok := updates["first_name"]; ok {
		firstName = v.(string)
	}
	if v, ok := updates["last_name"]; ok {
		lastName = v.(string)
	}
	if v, ok := updates["profile_picture"]; ok {
		profilePicture = v.(string)
	}
	if v, ok := updates["locale"]; ok {
		locale = v.(string)
	}

	return r.queries.UpdateUserProfile(context.Background(), sqlcgen.UpdateUserProfileParams{
		ID:             uid,
		Name:           name,
		FirstName:      firstName,
		LastName:       lastName,
		ProfilePicture: profilePicture,
		Locale:         locale,
	})
}

// UpdateUserEmail updates user email and sets email_verified to false
func (r *Repository) UpdateUserEmail(userID, newEmail string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.UpdateUserEmail(context.Background(), sqlcgen.UpdateUserEmailParams{
		ID:    uid,
		Email: newEmail,
	})
}

// ClearLockout clears the lockout fields for a user (auto-unlock on expired lockout).
func (r *Repository) ClearLockout(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.ClearLockout(context.Background(), uid)
}

// SetBackupEmail sets the pending backup email for a user (not yet verified).
func (r *Repository) SetBackupEmail(userID, backupEmail string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.SetBackupEmail(context.Background(), sqlcgen.SetBackupEmailParams{
		ID:          uid,
		BackupEmail: backupEmail,
	})
}

// VerifyBackupEmail marks the backup email as verified.
func (r *Repository) VerifyBackupEmail(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.VerifyBackupEmail(context.Background(), uid)
}

// ClearBackupEmail removes the backup email and its verified status.
func (r *Repository) ClearBackupEmail(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.ClearBackupEmail(context.Background(), uid)
}

// SaveAndSwitchToBackupEmail2FA atomically saves the user's current 2FA method/secret as
// "previous" fields and switches the active method to backup_email.
// This allows DisableBackupEmail2FAMethod to fully restore the prior configuration.
func (r *Repository) SaveAndSwitchToBackupEmail2FA(userID, previousMethod, previousSecret, recoveryCodes string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.SaveAndSwitchToBackupEmail2FA(context.Background(), sqlcgen.SaveAndSwitchToBackupEmail2FAParams{
		ID:                  uid,
		TwoFaPreviousMethod: previousMethod,
		TwoFaPreviousSecret: previousSecret,
		TwoFaRecoveryCodes:  json.RawMessage(recoveryCodes),
	})
}

// RestorePreviousTwoFAMethod reverts a user from backup_email 2FA back to their prior method.
// It reads the previously saved method/secret, restores them, and clears the "previous" fields.
// If no prior method was saved the user ends up with 2FA disabled.
func (r *Repository) RestorePreviousTwoFAMethod(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	ctx := context.Background()

	prev, err := r.queries.GetUserTwoFAPreviousFields(ctx, uid)
	if err != nil {
		return err
	}

	enabled := prev.TwoFaPreviousMethod != ""
	return r.queries.RestorePreviousTwoFAMethod(ctx, sqlcgen.RestorePreviousTwoFAMethodParams{
		ID:           uid,
		TwoFaMethod:  prev.TwoFaPreviousMethod,
		TwoFaSecret:  prev.TwoFaPreviousSecret,
		TwoFaEnabled: enabled,
	})
}

// SetPhoneNumber sets the phone number for a user (not yet verified).
func (r *Repository) SetPhoneNumber(userID, phone string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.SetPhoneNumber(context.Background(), sqlcgen.SetPhoneNumberParams{
		ID:          uid,
		PhoneNumber: phone,
	})
}

// VerifyPhoneNumber marks the phone number as verified.
func (r *Repository) VerifyPhoneNumber(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.VerifyPhoneNumber(context.Background(), uid)
}

// ClearPhone removes the phone number and its verified status.
func (r *Repository) ClearPhone(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries.ClearPhone(context.Background(), uid)
}

// ── helpers ────────────────────────────────────────────────────────────────────

func rowToUser(row sqlcgen.User) models.User {
	return models.User{
		ID:                  row.ID,
		AppID:               row.AppID,
		Email:               row.Email,
		PasswordHash:        row.PasswordHash,
		EmailVerified:       row.EmailVerified,
		IsActive:            row.IsActive,
		Name:                row.Name,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		ProfilePicture:      row.ProfilePicture,
		Locale:              row.Locale,
		TwoFAEnabled:        row.TwoFaEnabled,
		TwoFAMethod:         row.TwoFaMethod,
		TwoFASecret:         row.TwoFaSecret,
		TwoFARecoveryCodes:  json.RawMessage(row.TwoFaRecoveryCodes),
		BackupEmail:         row.BackupEmail,
		BackupEmailVerified: row.BackupEmailVerified,
		TwoFAPreviousMethod: row.TwoFaPreviousMethod,
		TwoFAPreviousSecret: row.TwoFaPreviousSecret,
		PhoneNumber:         row.PhoneNumber,
		PhoneVerified:       row.PhoneVerified,
		LockedAt:            timestamptzToTimePtr(row.LockedAt),
		LockReason:          row.LockReason,
		LockExpiresAt:       timestamptzToTimePtr(row.LockExpiresAt),
		PasswordHistory:     json.RawMessage(row.PasswordHistory),
		PasswordChangedAt:   timestamptzToTimePtr(row.PasswordChangedAt),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func applyRowToUser(user *models.User, row sqlcgen.User) {
	user.ID = row.ID
	user.AppID = row.AppID
	user.Email = row.Email
	user.PasswordHash = row.PasswordHash
	user.EmailVerified = row.EmailVerified
	user.IsActive = row.IsActive
	user.Name = row.Name
	user.FirstName = row.FirstName
	user.LastName = row.LastName
	user.ProfilePicture = row.ProfilePicture
	user.Locale = row.Locale
	user.TwoFAEnabled = row.TwoFaEnabled
	user.TwoFAMethod = row.TwoFaMethod
	user.TwoFASecret = row.TwoFaSecret
	user.TwoFARecoveryCodes = json.RawMessage(row.TwoFaRecoveryCodes)
	user.BackupEmail = row.BackupEmail
	user.BackupEmailVerified = row.BackupEmailVerified
	user.TwoFAPreviousMethod = row.TwoFaPreviousMethod
	user.TwoFAPreviousSecret = row.TwoFaPreviousSecret
	user.PhoneNumber = row.PhoneNumber
	user.PhoneVerified = row.PhoneVerified
	user.LockedAt = timestamptzToTimePtr(row.LockedAt)
	user.LockReason = row.LockReason
	user.LockExpiresAt = timestamptzToTimePtr(row.LockExpiresAt)
	user.PasswordHistory = json.RawMessage(row.PasswordHistory)
	user.PasswordChangedAt = timestamptzToTimePtr(row.PasswordChangedAt)
	user.CreatedAt = row.CreatedAt
	user.UpdatedAt = row.UpdatedAt
}

func toModelSocialAccounts(rows []sqlcgen.SocialAccount) []models.SocialAccount {
	out := make([]models.SocialAccount, len(rows))
	for i, row := range rows {
		out[i] = models.SocialAccount{
			ID:             row.ID,
			AppID:          row.AppID,
			UserID:         row.UserID,
			Provider:       row.Provider,
			ProviderUserID: row.ProviderUserID,
			Email:          row.Email,
			Name:           row.Name,
			FirstName:      row.FirstName,
			LastName:       row.LastName,
			ProfilePicture: row.ProfilePicture,
			Username:       row.Username,
			Locale:         row.Locale,
			RawData:        json.RawMessage(row.RawData),
			AccessToken:    row.AccessToken,
			RefreshToken:   row.RefreshToken,
			ExpiresAt:      timestamptzToTimePtr(row.ExpiresAt),
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		}
	}
	return out
}

func timePtrToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timestamptzToTimePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
