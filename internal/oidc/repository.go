package oidc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/google/uuid"
)

// Repository handles all OIDC-related database operations.
type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewRepository constructs a new OIDC Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// ─── Application ───────────────────────────────────────────────────────────────

// GetApplication fetches an Application by UUID.
func (r *Repository) GetApplication(appID uuid.UUID) (*models.Application, error) {
	row, err := r.queries.GetApplicationForOIDC(context.Background(), appID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	app := toModelApplication(row)
	return &app, nil
}

// SaveRSAPrivateKey persists the PEM-encoded RSA private key for an application.
func (r *Repository) SaveRSAPrivateKey(appID uuid.UUID, pemKey string) error {
	return r.queries.SaveOIDCRSAPrivateKey(context.Background(), sqlcgen.SaveOIDCRSAPrivateKeyParams{
		ID:                appID,
		OidcRsaPrivateKey: pemKey,
	})
}

// ─── OIDCClient CRUD ────────────────────────────────────────────────────────────

// CreateClient inserts a new OIDCClient.
func (r *Repository) CreateClient(client *models.OIDCClient) error {
	if client.ID == uuid.Nil {
		client.ID = uuid.New()
	}
	now := time.Now().UTC()
	if client.CreatedAt.IsZero() {
		client.CreatedAt = now
	}
	if client.UpdatedAt.IsZero() {
		client.UpdatedAt = now
	}
	return r.queries.CreateOIDCClient(context.Background(), sqlcgen.CreateOIDCClientParams{
		ID:                client.ID,
		AppID:             client.AppID,
		Name:              client.Name,
		Description:       client.Description,
		ClientID:          client.ClientID,
		ClientSecretHash:  client.ClientSecretHash,
		RedirectUris:      client.RedirectURIs,
		AllowedGrantTypes: client.AllowedGrantTypes,
		AllowedScopes:     client.AllowedScopes,
		RequireConsent:    client.RequireConsent,
		IsConfidential:    client.IsConfidential,
		PkceRequired:      client.PKCERequired,
		IsActive:          client.IsActive,
		LogoUrl:           client.LogoURL,
		LoginTheme:        client.LoginTheme,
		LoginPrimaryColor: client.LoginPrimaryColor,
		CreatedAt:         client.CreatedAt,
		UpdatedAt:         client.UpdatedAt,
	})
}

// GetClientByID fetches an OIDCClient by its primary key UUID.
func (r *Repository) GetClientByID(id uuid.UUID) (*models.OIDCClient, error) {
	row, err := r.queries.GetOIDCClientByID(context.Background(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	c := toModelOIDCClient(row)
	return &c, nil
}

// GetClientByClientID fetches an OIDCClient by its public client_id string.
func (r *Repository) GetClientByClientID(clientID string) (*models.OIDCClient, error) {
	row, err := r.queries.GetOIDCClientByClientID(context.Background(), clientID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	c := toModelOIDCClient(row)
	return &c, nil
}

// ListClientsByApp returns all OIDC clients registered for a given application.
func (r *Repository) ListClientsByApp(appID uuid.UUID) ([]models.OIDCClient, error) {
	rows, err := r.queries.ListOIDCClientsByApp(context.Background(), appID)
	if err != nil {
		return nil, err
	}
	clients := make([]models.OIDCClient, len(rows))
	for i, row := range rows {
		clients[i] = toModelOIDCClient(row)
	}
	return clients, nil
}

// UpdateClient persists changes to an OIDCClient.
func (r *Repository) UpdateClient(client *models.OIDCClient) error {
	return r.queries.UpdateOIDCClient(context.Background(), sqlcgen.UpdateOIDCClientParams{
		ID:                client.ID,
		AppID:             client.AppID,
		Name:              client.Name,
		Description:       client.Description,
		ClientID:          client.ClientID,
		ClientSecretHash:  client.ClientSecretHash,
		RedirectUris:      client.RedirectURIs,
		AllowedGrantTypes: client.AllowedGrantTypes,
		AllowedScopes:     client.AllowedScopes,
		RequireConsent:    client.RequireConsent,
		IsConfidential:    client.IsConfidential,
		PkceRequired:      client.PKCERequired,
		IsActive:          client.IsActive,
		LogoUrl:           client.LogoURL,
		LoginTheme:        client.LoginTheme,
		LoginPrimaryColor: client.LoginPrimaryColor,
	})
}

// DeleteClient hard-deletes an OIDCClient by UUID.
func (r *Repository) DeleteClient(id uuid.UUID) error {
	return r.queries.DeleteOIDCClient(context.Background(), id)
}

// ─── OIDCAuthCode ──────────────────────────────────────────────────────────────

// CreateAuthCode inserts a new authorization code record.
func (r *Repository) CreateAuthCode(code *models.OIDCAuthCode) error {
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = time.Now().UTC()
	}
	return r.queries.CreateOIDCAuthCode(context.Background(), sqlcgen.CreateOIDCAuthCodeParams{
		ID:                  code.ID,
		AppID:               code.AppID,
		ClientID:            code.ClientID,
		UserID:              code.UserID,
		Code:                code.Code,
		RedirectUri:         code.RedirectURI,
		Scopes:              code.Scopes,
		Nonce:               code.Nonce,
		CodeChallenge:       code.CodeChallenge,
		CodeChallengeMethod: code.CodeChallengeMethod,
		ExpiresAt:           code.ExpiresAt,
		Used:                code.Used,
		CreatedAt:           code.CreatedAt,
	})
}

// GetAuthCode fetches a single authorization code by its code string.
func (r *Repository) GetAuthCode(code string) (*models.OIDCAuthCode, error) {
	row, err := r.queries.GetOIDCAuthCode(context.Background(), code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	ac := toModelOIDCAuthCode(row)
	return &ac, nil
}

// MarkAuthCodeUsed marks an authorization code as used (prevents replay).
func (r *Repository) MarkAuthCodeUsed(id uuid.UUID) error {
	return r.queries.MarkOIDCAuthCodeUsed(context.Background(), id)
}

// DeleteExpiredAuthCodes removes all expired authorization codes.
// Call periodically to keep the table small.
func (r *Repository) DeleteExpiredAuthCodes() error {
	return r.queries.DeleteExpiredOIDCAuthCodes(context.Background())
}

// ─── User lookup (needed by service layer) ─────────────────────────────────────

// GetUserByID fetches a User by UUID string.
func (r *Repository) GetUserByID(id string) (*models.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetUserByIDForOIDC(context.Background(), uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	u := toModelUser(row)
	return &u, nil
}

// GetUserByEmail fetches a User by appID + email.
func (r *Repository) GetUserByEmail(appID, email string) (*models.User, error) {
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetUserByEmailForOIDC(context.Background(), sqlcgen.GetUserByEmailForOIDCParams{
		AppID: appUUID,
		Email: email,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	u := toModelUser(row)
	return &u, nil
}

// ── type conversions ────────────────────────────────────────────────────────

func toModelOIDCClient(row sqlcgen.OidcClient) models.OIDCClient {
	return models.OIDCClient{
		ID:                row.ID,
		AppID:             row.AppID,
		Name:              row.Name,
		Description:       row.Description,
		ClientID:          row.ClientID,
		ClientSecretHash:  row.ClientSecretHash,
		RedirectURIs:      row.RedirectUris,
		AllowedGrantTypes: row.AllowedGrantTypes,
		AllowedScopes:     row.AllowedScopes,
		RequireConsent:    row.RequireConsent,
		IsConfidential:    row.IsConfidential,
		PKCERequired:      row.PkceRequired,
		IsActive:          row.IsActive,
		LogoURL:           row.LogoUrl,
		LoginTheme:        row.LoginTheme,
		LoginPrimaryColor: row.LoginPrimaryColor,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func toModelOIDCAuthCode(row sqlcgen.OidcAuthCode) models.OIDCAuthCode {
	return models.OIDCAuthCode{
		ID:                  row.ID,
		AppID:               row.AppID,
		ClientID:            row.ClientID,
		UserID:              row.UserID,
		Code:                row.Code,
		RedirectURI:         row.RedirectUri,
		Scopes:              row.Scopes,
		Nonce:               row.Nonce,
		CodeChallenge:       row.CodeChallenge,
		CodeChallengeMethod: row.CodeChallengeMethod,
		ExpiresAt:           row.ExpiresAt,
		Used:                row.Used,
		CreatedAt:           row.CreatedAt,
	}
}

func toModelApplication(row sqlcgen.Application) models.Application {
	app := models.Application{
		ID:                        row.ID,
		TenantID:                  row.TenantID,
		Name:                      row.Name,
		TwoFAIssuerName:           row.TwoFaIssuerName,
		TwoFAEnabled:              row.TwoFaEnabled,
		TwoFARequired:             row.TwoFaRequired,
		Email2FAEnabled:           row.Email2faEnabled,
		TwoFAMethods:              row.TwoFaMethods,
		Passkey2FAEnabled:         row.Passkey2faEnabled,
		PasskeyLoginEnabled:       row.PasskeyLoginEnabled,
		MagicLinkEnabled:          row.MagicLinkEnabled,
		LoginNotificationsEnabled: row.LoginNotificationsEnabled,
		SuspiciousActivityAlerts:  row.SuspiciousActivityAlerts,
		SMS2FAEnabled:             row.Sms2faEnabled,
		TrustedDeviceEnabled:      row.TrustedDeviceEnabled,
		TrustedDeviceMaxDays:      int(row.TrustedDeviceMaxDays),
		BfLockoutEnabled:          row.BfLockoutEnabled,
		BfDelayEnabled:            row.BfDelayEnabled,
		BfCaptchaEnabled:          row.BfCaptchaEnabled,
		OIDCEnabled:               row.OidcEnabled,
		OIDCRSAPrivateKey:         row.OidcRsaPrivateKey,
		OIDCIDTokenTTL:            int(row.OidcIDTokenTtl),
		OIDCIssuerURL:             row.OidcIssuerUrl,
		FrontendURL:               row.FrontendUrl,
		LoginLogoURL:              row.LoginLogoUrl,
		LoginTheme:                row.LoginTheme,
		LoginPrimaryColor:         row.LoginPrimaryColor,
		LoginSecondaryColor:       row.LoginSecondaryColor,
		LoginDisplayName:          row.LoginDisplayName,
		PwMinLength:               int(row.PwMinLength),
		PwMaxLength:               int(row.PwMaxLength),
		PwRequireUpper:            row.PwRequireUpper,
		PwRequireLower:            row.PwRequireLower,
		PwRequireDigit:            row.PwRequireDigit,
		PwRequireSymbol:           row.PwRequireSymbol,
		PwHistoryCount:            int(row.PwHistoryCount),
		PwMaxAgeDays:              int(row.PwMaxAgeDays),
		AccessTokenTTLMinutes:     int(row.AccessTokenTtlMinutes),
		RefreshTokenTTLHours:      int(row.RefreshTokenTtlHours),
		ResetPasswordPath:         row.ResetPasswordPath,
		MagicLinkPath:             row.MagicLinkPath,
		VerifyEmailPath:           row.VerifyEmailPath,
		CreatedAt:                 row.CreatedAt,
		UpdatedAt:                 row.UpdatedAt,
	}

	// Handle nullable description
	if row.Description != nil {
		app.Description = *row.Description
	}

	// Handle nullable brute-force int fields (int32* -> int*)
	if row.BfLockoutThreshold != nil {
		v := int(*row.BfLockoutThreshold)
		app.BfLockoutThreshold = &v
	}
	if row.BfLockoutDurations != nil {
		app.BfLockoutDurations = row.BfLockoutDurations
	}
	if row.BfLockoutWindow != nil {
		app.BfLockoutWindow = row.BfLockoutWindow
	}
	if row.BfLockoutTierTtl != nil {
		app.BfLockoutTierTTL = row.BfLockoutTierTtl
	}
	if row.BfDelayStartAfter != nil {
		v := int(*row.BfDelayStartAfter)
		app.BfDelayStartAfter = &v
	}
	if row.BfDelayMaxSeconds != nil {
		v := int(*row.BfDelayMaxSeconds)
		app.BfDelayMaxSeconds = &v
	}
	if row.BfDelayTierTtl != nil {
		app.BfDelayTierTTL = row.BfDelayTierTtl
	}
	if row.BfCaptchaSiteKey != nil {
		app.BfCaptchaSiteKey = row.BfCaptchaSiteKey
	}
	if row.BfCaptchaSecretKey != nil {
		app.BfCaptchaSecretKey = row.BfCaptchaSecretKey
	}
	if row.BfCaptchaThreshold != nil {
		v := int(*row.BfCaptchaThreshold)
		app.BfCaptchaThreshold = &v
	}

	return app
}

func toModelUser(row sqlcgen.User) models.User {
	u := models.User{
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
		LockReason:          row.LockReason,
		PasswordHistory:     json.RawMessage(row.PasswordHistory),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	if row.LockedAt.Valid {
		t := row.LockedAt.Time
		u.LockedAt = &t
	}
	if row.LockExpiresAt.Valid {
		t := row.LockExpiresAt.Time
		u.LockExpiresAt = &t
	}
	if row.PasswordChangedAt.Valid {
		t := row.PasswordChangedAt.Time
		u.PasswordChangedAt = &t
	}

	return u
}
