package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/safeconv"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/internal/sso"
	"github.com/MrF1ow/go-core/pkg/dto"
	"github.com/MrF1ow/go-core/pkg/models"
)

// errNotFound is returned when a record is not found.
// errNotFound is the sentinel used when a record lookup returns no rows.
var errNotFound = pgx.ErrNoRows

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

// ============================================================
// Tenant Operations
// ============================================================

func (r *Repository) CreateTenant(tenant *models.Tenant) error {
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}
	now := time.Now().UTC()
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = now
	}
	if tenant.UpdatedAt.IsZero() {
		tenant.UpdatedAt = now
	}

	row, err := r.queries.AdminCreateTenant(context.Background(), sqlcgen.AdminCreateTenantParams{
		ID:        tenant.ID,
		Name:      tenant.Name,
		CreatedAt: tenant.CreatedAt,
		UpdatedAt: tenant.UpdatedAt,
	})
	if err != nil {
		return err
	}
	tenant.ID = row.ID
	tenant.Name = row.Name
	tenant.CreatedAt = row.CreatedAt
	tenant.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) GetTenantByID(id string) (*models.Tenant, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetTenantByID(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	// Load apps
	appRows, err := r.queries.AdminGetTenantApps(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	apps := make([]models.Application, len(appRows))
	for i, a := range appRows {
		apps[i] = sqlcAppToModel(a)
	}
	return &models.Tenant{
		ID:        row.ID,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Apps:      apps,
	}, nil
}

func (r *Repository) ListTenants(page, pageSize int) ([]models.Tenant, int64, error) {
	ctx := context.Background()
	total, err := r.queries.AdminCountTenants(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListTenants(ctx, sqlcgen.AdminListTenantsParams{
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	tenants := make([]models.Tenant, len(rows))
	for i, row := range rows {
		tenants[i] = models.Tenant{
			ID:        row.ID,
			Name:      row.Name,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return tenants, total, nil
}

// TenantListItem holds a tenant with its app count for list views.
type TenantListItem struct {
	ID        uuid.UUID
	Name      string
	AppCount  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListTenantsWithAppCount returns paginated tenants with their application counts.
func (r *Repository) ListTenantsWithAppCount(page, pageSize int) ([]TenantListItem, int64, error) {
	ctx := context.Background()
	total, err := r.queries.AdminCountTenants(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListTenantsWithAppCount(ctx, sqlcgen.AdminListTenantsWithAppCountParams{
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]TenantListItem, len(rows))
	for i, row := range rows {
		items[i] = TenantListItem{
			ID:        row.ID,
			Name:      row.Name,
			AppCount:  row.AppCount,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}
	return items, total, nil
}

func (r *Repository) UpdateTenant(id string, name string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminUpdateTenant(context.Background(), sqlcgen.AdminUpdateTenantParams{
		ID:   uid,
		Name: name,
	})
}

func (r *Repository) DeleteTenant(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminDeleteTenant(context.Background(), uid)
}

// ListAllTenants returns all tenants (ID and Name only), ordered by name.
// Used for populating dropdown selects in forms and filters.
func (r *Repository) ListAllTenants() ([]models.Tenant, error) {
	rows, err := r.queries.AdminListAllTenants(context.Background())
	if err != nil {
		return nil, err
	}
	tenants := make([]models.Tenant, len(rows))
	for i, row := range rows {
		tenants[i] = models.Tenant{
			ID:   row.ID,
			Name: row.Name,
		}
	}
	return tenants, nil
}

// Count Operations (used by Dashboard)

func (r *Repository) CountTenants() (int64, error) {
	return r.queries.AdminCountTenants(context.Background())
}

func (r *Repository) CountApps() (int64, error) {
	return r.queries.AdminCountApps(context.Background())
}

// ============================================================
// App Operations
// ============================================================

func (r *Repository) CreateApp(app *models.Application) error {
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	now := time.Now().UTC()
	if app.CreatedAt.IsZero() {
		app.CreatedAt = now
	}
	if app.UpdatedAt.IsZero() {
		app.UpdatedAt = now
	}

	row, err := r.queries.AdminCreateApp(context.Background(), sqlcgen.AdminCreateAppParams{
		ID:                        app.ID,
		TenantID:                  app.TenantID,
		Name:                      app.Name,
		Description:               strToPtr(app.Description),
		TwoFaIssuerName:           app.TwoFAIssuerName,
		TwoFaEnabled:              app.TwoFAEnabled,
		TwoFaRequired:             app.TwoFARequired,
		Email2faEnabled:           app.Email2FAEnabled,
		TwoFaMethods:              app.TwoFAMethods,
		Passkey2faEnabled:         app.Passkey2FAEnabled,
		PasskeyLoginEnabled:       app.PasskeyLoginEnabled,
		MagicLinkEnabled:          app.MagicLinkEnabled,
		LoginNotificationsEnabled: app.LoginNotificationsEnabled,
		SuspiciousActivityAlerts:  app.SuspiciousActivityAlerts,
		Sms2faEnabled:             app.SMS2FAEnabled,
		TrustedDeviceEnabled:      app.TrustedDeviceEnabled,
		TrustedDeviceMaxDays:      safeconv.ToInt32(app.TrustedDeviceMaxDays),
		OidcEnabled:               app.OIDCEnabled,
		OidcRsaPrivateKey:         app.OIDCRSAPrivateKey,
		OidcIDTokenTtl:            safeconv.ToInt32(app.OIDCIDTokenTTL),
		OidcIssuerUrl:             app.OIDCIssuerURL,
		FrontendUrl:               app.FrontendURL,
		LoginLogoUrl:              app.LoginLogoURL,
		LoginTheme:                app.LoginTheme,
		LoginPrimaryColor:         app.LoginPrimaryColor,
		LoginSecondaryColor:       app.LoginSecondaryColor,
		LoginDisplayName:          app.LoginDisplayName,
		PwMinLength:               safeconv.ToInt32(app.PwMinLength),
		PwMaxLength:               safeconv.ToInt32(app.PwMaxLength),
		PwRequireUpper:            app.PwRequireUpper,
		PwRequireLower:            app.PwRequireLower,
		PwRequireDigit:            app.PwRequireDigit,
		PwRequireSymbol:           app.PwRequireSymbol,
		PwHistoryCount:            safeconv.ToInt32(app.PwHistoryCount),
		PwMaxAgeDays:              safeconv.ToInt32(app.PwMaxAgeDays),
		AccessTokenTtlMinutes:     safeconv.ToInt32(app.AccessTokenTTLMinutes),
		RefreshTokenTtlHours:      safeconv.ToInt32(app.RefreshTokenTTLHours),
		ResetPasswordPath:         app.ResetPasswordPath,
		MagicLinkPath:             app.MagicLinkPath,
		VerifyEmailPath:           app.VerifyEmailPath,
		CreatedAt:                 app.CreatedAt,
		UpdatedAt:                 app.UpdatedAt,
	})
	if err != nil {
		return err
	}
	applySqlcAppToModel(app, row)
	return nil
}

// SeedDefaultRolesForApp creates the default system roles (admin, member, viewer) for an application
// and assigns them the standard permissions. Should be called after creating a new application.
func (r *Repository) SeedDefaultRolesForApp(appID uuid.UUID) error {
	ctx := context.Background()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := r.queries.WithTx(tx)

	// Load all permissions
	permissions, err := qtx.AdminListAllPermissions(ctx)
	if err != nil {
		return err
	}
	if len(permissions) == 0 {
		return tx.Commit(ctx) // No permissions seeded yet, skip
	}

	// Build permission lookup by "resource:action"
	permByKey := make(map[string]sqlcgen.Permission)
	for _, p := range permissions {
		permByKey[p.Resource+":"+p.Action] = p
	}

	// Define default roles and their permission keys
	type roleDef struct {
		Name        string
		Description string
		PermKeys    []string // nil means ALL permissions
	}

	roleDefs := []roleDef{
		{
			Name:        "admin",
			Description: "Full access to all resources within the application",
			PermKeys:    nil, // all
		},
		{
			Name:        "member",
			Description: "Standard user with read and limited write access",
			PermKeys:    []string{"user:read", "user:write", "log:read", "role:read", "settings:read", "settings:write"},
		},
		{
			Name:        "viewer",
			Description: "Read-only access to resources",
			PermKeys:    []string{"user:read", "log:read", "role:read"},
		},
	}

	for _, rd := range roleDefs {
		// Check if role already exists for this app
		_, err := qtx.AdminGetRoleByAppAndName(ctx, sqlcgen.AdminGetRoleByAppAndNameParams{
			AppID: appID,
			Name:  rd.Name,
		})
		if err == nil {
			continue // Already exists, skip
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		role, err := qtx.AdminCreateRole(ctx, sqlcgen.AdminCreateRoleParams{
			ID:          uuid.New(),
			AppID:       appID,
			Name:        rd.Name,
			Description: &rd.Description,
			IsSystem:    true,
		})
		if err != nil {
			return err
		}

		// Delete any existing permissions for this role (clean slate)
		if err := qtx.AdminDeleteRolePermissions(ctx, role.ID); err != nil {
			return err
		}

		// Assign permissions
		var permsToAssign []sqlcgen.Permission
		if rd.PermKeys == nil {
			permsToAssign = permissions // all
		} else {
			for _, key := range rd.PermKeys {
				if p, ok := permByKey[key]; ok {
					permsToAssign = append(permsToAssign, p)
				}
			}
		}

		for _, p := range permsToAssign {
			if err := qtx.AdminAddRolePermission(ctx, sqlcgen.AdminAddRolePermissionParams{
				RoleID:       role.ID,
				PermissionID: p.ID,
			}); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetAppByID(id string) (*models.Application, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	row, err := r.queries.AdminGetAppByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	app := sqlcAppToModel(row)

	// Load OAuth configs (Preload equivalent)
	oauthRows, err := r.queries.AdminGetAppOAuthConfigs(ctx, uid)
	if err != nil {
		return nil, err
	}
	app.OAuthProviderConfigs = make([]models.OAuthProviderConfig, len(oauthRows))
	for i, o := range oauthRows {
		app.OAuthProviderConfigs[i] = sqlcOAuthToModel(o)
	}
	return &app, nil
}

func (r *Repository) ListAppsByTenant(tenantID string) ([]models.Application, error) {
	uid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.AdminListAppsByTenant(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	apps := make([]models.Application, len(rows))
	for i, row := range rows {
		apps[i] = sqlcAppToModel(row)
	}
	return apps, nil
}

// AppListItem holds an application with its tenant name and OAuth config count for list views.
type AppListItem struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	Name                string
	Description         string
	TenantName          string
	OAuthConfigCount    int64
	TwoFAEnabled        bool
	TwoFARequired       bool
	Passkey2FAEnabled   bool
	PasskeyLoginEnabled bool
	MagicLinkEnabled    bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ListAppsWithDetails returns paginated applications with tenant name and OAuth config count.
// If tenantID is non-empty, results are filtered to that tenant.
func (r *Repository) ListAppsWithDetails(page, pageSize int, tenantID string) ([]AppListItem, int64, error) {
	ctx := context.Background()

	var tenantFilter pgtype.UUID
	if tenantID != "" {
		uid, err := uuid.Parse(tenantID)
		if err != nil {
			return nil, 0, err
		}
		tenantFilter = pgtype.UUID{Bytes: uid, Valid: true}
	}

	total, err := r.queries.AdminCountAppsWithTenantFilter(ctx, tenantFilter)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListAppsWithDetails(ctx, sqlcgen.AdminListAppsWithDetailsParams{
		Limit:    safeconv.ToInt32(pageSize),
		Offset:   safeconv.ToInt32(offset),
		TenantID: tenantFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]AppListItem, len(rows))
	for i, row := range rows {
		items[i] = AppListItem{
			ID:                  row.ID,
			TenantID:            row.TenantID,
			Name:                row.Name,
			Description:         ptrToStr(row.Description),
			TenantName:          ptrToStr(row.TenantName),
			OAuthConfigCount:    row.OauthConfigCount,
			TwoFAEnabled:        row.TwoFaEnabled,
			TwoFARequired:       row.TwoFaRequired,
			Passkey2FAEnabled:   row.Passkey2faEnabled,
			PasskeyLoginEnabled: row.PasskeyLoginEnabled,
			MagicLinkEnabled:    row.MagicLinkEnabled,
			CreatedAt:           row.CreatedAt,
			UpdatedAt:           row.UpdatedAt,
		}
	}
	return items, total, nil
}

// BruteForceAppSettings holds per-application brute-force override values.
// Nil pointers mean "clear override, use global default".
type BruteForceAppSettings struct {
	LockoutEnabled   *bool
	LockoutThreshold *int
	LockoutDurations *string
	LockoutWindow    *string
	LockoutTierTTL   *string
	DelayEnabled     *bool
	DelayStartAfter  *int
	DelayMaxSeconds  *int
	DelayTierTTL     *string
	CaptchaEnabled   *bool
	CaptchaSiteKey   *string
	CaptchaSecretKey *string // empty string in form = keep existing; nil = clear override
	CaptchaThreshold *int
}

// AppCustomizationSettings holds per-application branding, password policy, token TTL, and email link path fields.
type AppCustomizationSettings struct {
	// Login Page Branding
	LoginLogoURL        string
	LoginPrimaryColor   string
	LoginSecondaryColor string
	LoginDisplayName    string
	// Password Policy
	PwMinLength     int
	PwMaxLength     int
	PwRequireUpper  bool
	PwRequireLower  bool
	PwRequireDigit  bool
	PwRequireSymbol bool
	PwHistoryCount  int
	PwMaxAgeDays    int
	// Token TTL overrides (0 = use global defaults)
	AccessTokenTTLMinutes int
	RefreshTokenTTLHours  int
	// Email Action Link Paths (empty = use system defaults)
	ResetPasswordPath string
	MagicLinkPath     string
	VerifyEmailPath   string
}

func (r *Repository) UpdateApp(id string, name string, description string, frontendURL string, twoFAIssuerName string, twoFAEnabled bool, twoFARequired bool, passkey2FAEnabled bool, passkeyLoginEnabled bool, magicLinkEnabled bool, oidcEnabled bool, bf BruteForceAppSettings, custom AppCustomizationSettings) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	err = r.queries.AdminUpdateApp(context.Background(), sqlcgen.AdminUpdateAppParams{
		ID:                    uid,
		Name:                  name,
		Description:           strToPtr(description),
		FrontendUrl:           frontendURL,
		TwoFaIssuerName:       twoFAIssuerName,
		TwoFaEnabled:          twoFAEnabled,
		TwoFaRequired:         twoFARequired,
		Passkey2faEnabled:     passkey2FAEnabled,
		PasskeyLoginEnabled:   passkeyLoginEnabled,
		MagicLinkEnabled:      magicLinkEnabled,
		OidcEnabled:           oidcEnabled,
		BfLockoutEnabled:      bf.LockoutEnabled,
		BfLockoutThreshold:    intPtrToInt32Ptr(bf.LockoutThreshold),
		BfLockoutDurations:    bf.LockoutDurations,
		BfLockoutWindow:       bf.LockoutWindow,
		BfLockoutTierTtl:      bf.LockoutTierTTL,
		BfDelayEnabled:        bf.DelayEnabled,
		BfDelayStartAfter:     intPtrToInt32Ptr(bf.DelayStartAfter),
		BfDelayMaxSeconds:     intPtrToInt32Ptr(bf.DelayMaxSeconds),
		BfDelayTierTtl:        bf.DelayTierTTL,
		BfCaptchaEnabled:      bf.CaptchaEnabled,
		BfCaptchaSiteKey:      bf.CaptchaSiteKey,
		BfCaptchaThreshold:    intPtrToInt32Ptr(bf.CaptchaThreshold),
		LoginLogoUrl:          custom.LoginLogoURL,
		LoginPrimaryColor:     custom.LoginPrimaryColor,
		LoginSecondaryColor:   custom.LoginSecondaryColor,
		LoginDisplayName:      custom.LoginDisplayName,
		PwMinLength:           safeconv.ToInt32(custom.PwMinLength),
		PwMaxLength:           safeconv.ToInt32(custom.PwMaxLength),
		PwRequireUpper:        custom.PwRequireUpper,
		PwRequireLower:        custom.PwRequireLower,
		PwRequireDigit:        custom.PwRequireDigit,
		PwRequireSymbol:       custom.PwRequireSymbol,
		PwHistoryCount:        safeconv.ToInt32(custom.PwHistoryCount),
		PwMaxAgeDays:          safeconv.ToInt32(custom.PwMaxAgeDays),
		AccessTokenTtlMinutes: safeconv.ToInt32(custom.AccessTokenTTLMinutes),
		RefreshTokenTtlHours:  safeconv.ToInt32(custom.RefreshTokenTTLHours),
		ResetPasswordPath:     custom.ResetPasswordPath,
		MagicLinkPath:         custom.MagicLinkPath,
		VerifyEmailPath:       custom.VerifyEmailPath,
	})
	if err != nil {
		return err
	}

	// Only update CAPTCHA secret key if explicitly provided (non-nil and non-empty).
	if bf.CaptchaSecretKey != nil {
		if err := r.queries.AdminUpdateAppCaptchaSecretKey(context.Background(), sqlcgen.AdminUpdateAppCaptchaSecretKeyParams{
			ID:                 uid,
			BfCaptchaSecretKey: bf.CaptchaSecretKey,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) DeleteApp(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminDeleteApp(context.Background(), uid)
}

// UpdateAppSMSTrustedDevice updates the SMS and trusted device settings for an application.
func (r *Repository) UpdateAppSMSTrustedDevice(id string, sms2FAEnabled bool, trustedDeviceEnabled bool, trustedDeviceMaxDays int) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminUpdateAppSMSTrustedDevice(context.Background(), sqlcgen.AdminUpdateAppSMSTrustedDeviceParams{
		ID:                   uid,
		Sms2faEnabled:        sms2FAEnabled,
		TrustedDeviceEnabled: trustedDeviceEnabled,
		TrustedDeviceMaxDays: safeconv.ToInt32(trustedDeviceMaxDays),
	})
}

// AppWithTenant holds an application ID, name, and its tenant name for dropdown selects.
type AppWithTenant struct {
	ID         uuid.UUID
	Name       string
	TenantName string
}

// ListAllAppsWithTenantName returns all applications with their tenant name, ordered by tenant then app name.
// Used for populating dropdown selects in forms and filters.
func (r *Repository) ListAllAppsWithTenantName() ([]AppWithTenant, error) {
	rows, err := r.queries.AdminListAllAppsWithTenantName(context.Background())
	if err != nil {
		return nil, err
	}
	items := make([]AppWithTenant, len(rows))
	for i, row := range rows {
		items[i] = AppWithTenant{
			ID:         row.ID,
			Name:       row.Name,
			TenantName: ptrToStr(row.TenantName),
		}
	}
	return items, nil
}

// ============================================================
// OAuth Config Operations
// ============================================================

func (r *Repository) UpsertOAuthConfig(config *models.OAuthProviderConfig) error {
	ctx := context.Background()
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}
	now := time.Now().UTC()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	if config.UpdatedAt.IsZero() {
		config.UpdatedAt = now
	}

	// Check if exists
	existing, err := r.queries.AdminGetOAuthConfigByAppAndProvider(ctx, sqlcgen.AdminGetOAuthConfigByAppAndProviderParams{
		AppID:    config.AppID,
		Provider: config.Provider,
	})

	if err == nil {
		// Update existing
		config.ID = existing.ID
		return r.queries.AdminUpdateOAuthConfig(ctx, sqlcgen.AdminUpdateOAuthConfigParams{
			ID:           config.ID,
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectUrl:  config.RedirectURL,
			IsEnabled:    config.IsEnabled,
		})
	}

	// Create new
	row, err := r.queries.AdminCreateOAuthConfig(ctx, sqlcgen.AdminCreateOAuthConfigParams{
		ID:           config.ID,
		AppID:        config.AppID,
		Provider:     config.Provider,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectUrl:  config.RedirectURL,
		IsEnabled:    config.IsEnabled,
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    config.UpdatedAt,
	})
	if err != nil {
		return err
	}
	config.ID = row.ID
	return nil
}

// OAuthConfigListItem holds an OAuth config with app and tenant names for list views.
type OAuthConfigListItem struct {
	ID          uuid.UUID
	AppID       uuid.UUID
	AppName     string
	TenantName  string
	Provider    string
	ClientID    string
	RedirectURL string
	IsEnabled   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListOAuthConfigsWithDetails returns paginated OAuth configs with app and tenant names.
// If appID is non-empty, results are filtered to that application.
func (r *Repository) ListOAuthConfigsWithDetails(page, pageSize int, appID string) ([]OAuthConfigListItem, int64, error) {
	ctx := context.Background()

	var appFilter pgtype.UUID
	if appID != "" {
		uid, err := uuid.Parse(appID)
		if err != nil {
			return nil, 0, err
		}
		appFilter = pgtype.UUID{Bytes: uid, Valid: true}
	}

	total, err := r.queries.AdminCountOAuthConfigs(ctx, appFilter)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListOAuthConfigsWithDetails(ctx, sqlcgen.AdminListOAuthConfigsWithDetailsParams{
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
		AppID:  appFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]OAuthConfigListItem, len(rows))
	for i, row := range rows {
		items[i] = OAuthConfigListItem{
			ID:          row.ID,
			AppID:       row.AppID,
			AppName:     ptrToStr(row.AppName),
			TenantName:  ptrToStr(row.TenantName),
			Provider:    row.Provider,
			ClientID:    row.ClientID,
			RedirectURL: row.RedirectUrl,
			IsEnabled:   row.IsEnabled,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		}
	}
	return items, total, nil
}

// GetOAuthConfigByID returns a single OAuth config by ID.
func (r *Repository) GetOAuthConfigByID(id string) (*models.OAuthProviderConfig, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetOAuthConfigByID(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	m := sqlcOAuthToModel(row)
	return &m, nil
}

// UpdateOAuthConfigByID updates an OAuth config by primary key.
// If clientSecret is empty, the existing secret is preserved.
func (r *Repository) UpdateOAuthConfigByID(id string, clientID string, clientSecret string, redirectURL string, isEnabled bool) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	if clientSecret != "" {
		return r.queries.AdminUpdateOAuthConfigByIDWithSecret(context.Background(), sqlcgen.AdminUpdateOAuthConfigByIDWithSecretParams{
			ID:           uid,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectUrl:  redirectURL,
			IsEnabled:    isEnabled,
		})
	}
	return r.queries.AdminUpdateOAuthConfigByID(context.Background(), sqlcgen.AdminUpdateOAuthConfigByIDParams{
		ID:          uid,
		ClientID:    clientID,
		RedirectUrl: redirectURL,
		IsEnabled:   isEnabled,
	})
}

// DeleteOAuthConfig deletes an OAuth config by ID.
func (r *Repository) DeleteOAuthConfig(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminDeleteOAuthConfig(context.Background(), uid)
}

// ToggleOAuthConfigEnabled flips the IsEnabled flag for an OAuth config.
func (r *Repository) ToggleOAuthConfigEnabled(id string) (*models.OAuthProviderConfig, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminToggleOAuthConfigEnabled(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	m := sqlcOAuthToModel(row)
	return &m, nil
}

// GetEnabledOAuthProviders returns the provider names (e.g. "google", "github") that
// are configured and enabled for the given app. Used by the public /app-config endpoint.
func (r *Repository) GetEnabledOAuthProviders(appID string) ([]string, error) {
	uid, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	return r.queries.AdminGetEnabledOAuthProviders(context.Background(), uid)
}

// HasActiveOIDCClients returns true if at least one active OIDC client exists for
// the given app. Used by the public /app-config endpoint.
func (r *Repository) HasActiveOIDCClients(appID string) (bool, error) {
	uid, err := uuid.Parse(appID)
	if err != nil {
		return false, err
	}
	count, err := r.queries.AdminCountActiveOIDCClients(context.Background(), uid)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetFirstActiveOIDCClientLoginTheme returns the login_theme of the first active
// OIDC client for the given app, or an empty string if none exists.
func (r *Repository) GetFirstActiveOIDCClientLoginTheme(appID string) (string, error) {
	uid, err := uuid.Parse(appID)
	if err != nil {
		return "", err
	}
	theme, err := r.queries.AdminGetFirstActiveOIDCClientLoginTheme(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return theme, nil
}

// ============================================================
// User Operations (Admin GUI - read + toggle only)
// ============================================================

// UserListItem represents a user row in the admin GUI list view
type UserListItem struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	Name               string     `json:"name"`
	AppID              uuid.UUID  `json:"app_id"`
	AppName            string     `json:"app_name"`
	TenantName         string     `json:"tenant_name"`
	IsActive           bool       `json:"is_active"`
	EmailVerified      bool       `json:"email_verified"`
	TwoFAEnabled       bool       `json:"two_fa_enabled"`
	HasPassword        bool       `json:"has_password"`
	SocialAccountCount int        `json:"social_account_count"`
	LockedAt           *time.Time `json:"locked_at"`
	LockExpiresAt      *time.Time `json:"lock_expires_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

// UserDetail represents a full user view with social accounts for the admin GUI detail panel
type UserDetail struct {
	ID                  uuid.UUID                   `json:"id"`
	Email               string                      `json:"email"`
	Name                string                      `json:"name"`
	FirstName           string                      `json:"first_name"`
	LastName            string                      `json:"last_name"`
	ProfilePicture      string                      `json:"profile_picture"`
	Locale              string                      `json:"locale"`
	AppID               uuid.UUID                   `json:"app_id"`
	AppName             string                      `json:"app_name"`
	TenantName          string                      `json:"tenant_name"`
	IsActive            bool                        `json:"is_active"`
	EmailVerified       bool                        `json:"email_verified"`
	TwoFAEnabled        bool                        `json:"two_fa_enabled"`
	HasPassword         bool                        `json:"has_password"`
	BackupEmail         string                      `json:"backup_email"`
	BackupEmailVerified bool                        `json:"backup_email_verified"`
	PhoneNumber         string                      `json:"phone_number"`
	PhoneVerified       bool                        `json:"phone_verified"`
	LockedAt            *time.Time                  `json:"locked_at"`
	LockReason          string                      `json:"lock_reason"`
	LockExpiresAt       *time.Time                  `json:"lock_expires_at"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
	SocialAccounts      []models.SocialAccount      `json:"social_accounts"`
	WebAuthnCredentials []models.WebAuthnCredential `json:"webauthn_credentials"`
	TrustedDevices      []models.TrustedDevice      `json:"trusted_devices"`
}

// UserStatusCounts holds active/inactive user counts for dashboard display
type UserStatusCounts struct {
	ActiveUsers   int64 `json:"active_users"`
	InactiveUsers int64 `json:"inactive_users"`
}

// ListUsersWithDetails returns a paginated list of users with app/tenant info and social account counts.
// Supports optional filtering by appID and text search on email/name.
func (r *Repository) ListUsersWithDetails(page, pageSize int, appID, search string) ([]UserListItem, int64, error) {
	ctx := context.Background()

	var appFilter pgtype.UUID
	if appID != "" {
		uid, err := uuid.Parse(appID)
		if err != nil {
			return nil, 0, err
		}
		appFilter = pgtype.UUID{Bytes: uid, Valid: true}
	}

	var searchFilter *string
	if search != "" {
		searchFilter = &search
	}

	total, err := r.queries.AdminCountUsersWithFilters(ctx, sqlcgen.AdminCountUsersWithFiltersParams{
		AppID:  appFilter,
		Search: searchFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListUsersWithDetails(ctx, sqlcgen.AdminListUsersWithDetailsParams{
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
		AppID:  appFilter,
		Search: searchFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]UserListItem, len(rows))
	for i, row := range rows {
		items[i] = UserListItem{
			ID:                 row.ID,
			Email:              row.Email,
			Name:               row.Name,
			AppID:              row.AppID,
			AppName:            ptrToStr(row.AppName),
			TenantName:         row.TenantName,
			IsActive:           row.IsActive,
			EmailVerified:      row.EmailVerified,
			TwoFAEnabled:       row.TwoFaEnabled,
			HasPassword:        row.HasPassword,
			SocialAccountCount: int(row.SocialAccountCount),
			LockedAt:           timestamptzToTimePtr(row.LockedAt),
			LockExpiresAt:      timestamptzToTimePtr(row.LockExpiresAt),
			CreatedAt:          row.CreatedAt,
		}
	}
	return items, total, nil
}

// GetUserDetailByID returns a full user detail view with social accounts, app name, and tenant name.
func (r *Repository) GetUserDetailByID(id string) (*UserDetail, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	row, err := r.queries.AdminGetUserDetailByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}

	detail := &UserDetail{
		ID:                  row.ID,
		Email:               row.Email,
		Name:                row.Name,
		FirstName:           row.FirstName,
		LastName:            row.LastName,
		ProfilePicture:      row.ProfilePicture,
		Locale:              row.Locale,
		AppID:               row.AppID,
		AppName:             ptrToStr(row.AppName),
		TenantName:          row.TenantName,
		IsActive:            row.IsActive,
		EmailVerified:       row.EmailVerified,
		TwoFAEnabled:        row.TwoFaEnabled,
		HasPassword:         row.HasPassword,
		BackupEmail:         row.BackupEmail,
		BackupEmailVerified: row.BackupEmailVerified,
		PhoneNumber:         row.PhoneNumber,
		PhoneVerified:       row.PhoneVerified,
		LockedAt:            timestamptzToTimePtr(row.LockedAt),
		LockReason:          row.LockReason,
		LockExpiresAt:       timestamptzToTimePtr(row.LockExpiresAt),
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}

	// Load social accounts
	saRows, err := r.queries.AdminGetSocialAccountsByUserID(ctx, uid)
	if err != nil {
		return nil, err
	}
	detail.SocialAccounts = make([]models.SocialAccount, len(saRows))
	for i, sa := range saRows {
		detail.SocialAccounts[i] = sqlcSocialAccountToModel(sa)
	}

	// Load WebAuthn passkey credentials
	credRows, err := r.queries.AdminGetWebAuthnCredentialsByUserID(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, err
	}
	detail.WebAuthnCredentials = make([]models.WebAuthnCredential, len(credRows))
	for i, cred := range credRows {
		detail.WebAuthnCredentials[i] = sqlcWebAuthnToModel(cred)
	}

	return detail, nil
}

// ToggleUserActive toggles the is_active flag for a user and returns the new value along with the user's app_id.
func (r *Repository) ToggleUserActive(id string) (isActive bool, appID string, err error) {
	uid, parseErr := uuid.Parse(id)
	if parseErr != nil {
		return false, "", parseErr
	}
	ctx := context.Background()
	row, err := r.queries.AdminGetUserActiveAndAppID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", errNotFound
		}
		return false, "", err
	}
	newActive := !row.IsActive
	if err := r.queries.AdminSetUserActive(ctx, sqlcgen.AdminSetUserActiveParams{
		ID:       uid,
		IsActive: newActive,
	}); err != nil {
		return false, "", err
	}
	return newActive, row.AppID.String(), nil
}

// UnlockUser clears the lockout fields for a user and returns the user's email and app_id.
func (r *Repository) UnlockUser(id string) (email string, appID string, err error) {
	uid, parseErr := uuid.Parse(id)
	if parseErr != nil {
		return "", "", parseErr
	}
	ctx := context.Background()
	row, err := r.queries.AdminGetUserEmailAndAppID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", errNotFound
		}
		return "", "", err
	}
	if err := r.queries.AdminUnlockUser(ctx, uid); err != nil {
		return "", "", err
	}
	return row.Email, row.AppID.String(), nil
}

// CountUsersByStatus returns the count of active and inactive users.
func (r *Repository) CountUsersByStatus() (*UserStatusCounts, error) {
	ctx := context.Background()
	active, err := r.queries.AdminCountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	inactive, err := r.queries.AdminCountInactiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	return &UserStatusCounts{
		ActiveUsers:   active,
		InactiveUsers: inactive,
	}, nil
}

// ============================================================
// Activity Log Operations (Admin GUI - read only)
// ============================================================

// ActivityLogListItem represents an activity log row in the admin GUI list view.
type ActivityLogListItem struct {
	ID        uuid.UUID `json:"id"`
	AppID     uuid.UUID `json:"app_id"`
	AppName   string    `json:"app_name"`
	UserID    uuid.UUID `json:"user_id"`
	UserEmail string    `json:"user_email"`
	EventType string    `json:"event_type"`
	Severity  string    `json:"severity"`
	IPAddress string    `json:"ip_address"`
	IsAnomaly bool      `json:"is_anomaly"`
	Timestamp time.Time `json:"timestamp"`
}

// ActivityLogExportItem extends ActivityLogListItem with extra fields useful for compliance exports.
type ActivityLogExportItem struct {
	ActivityLogListItem
	UserAgent string `json:"user_agent"`
}

// ExportActivityLogsMaxRows is the hard cap for admin GUI log exports.
const ExportActivityLogsMaxRows = 10_000

// ActivityLogDetail represents a full activity log view for the admin GUI detail panel.
type ActivityLogDetail struct {
	ID        uuid.UUID  `json:"id"`
	AppID     uuid.UUID  `json:"app_id"`
	AppName   string     `json:"app_name"`
	UserID    uuid.UUID  `json:"user_id"`
	UserEmail string     `json:"user_email"`
	EventType string     `json:"event_type"`
	Severity  string     `json:"severity"`
	IPAddress string     `json:"ip_address"`
	UserAgent string     `json:"user_agent"`
	Details   string     `json:"details"`
	IsAnomaly bool       `json:"is_anomaly"`
	ExpiresAt *time.Time `json:"expires_at"`
	Timestamp time.Time  `json:"timestamp"`
}

// activityLogFilterArgs builds the common filter params used by activity log queries.
func activityLogFilterArgs(eventType, severity, appID, search, startDate, endDate string) (
	evtType *string,
	sev *string,
	appFilter pgtype.UUID,
	srch *string,
	start pgtype.Timestamptz,
	end pgtype.Timestamptz,
) {
	if eventType != "" {
		evtType = &eventType
	}
	if severity != "" {
		sev = &severity
	}
	if appID != "" {
		if uid, err := uuid.Parse(appID); err == nil {
			appFilter = pgtype.UUID{Bytes: uid, Valid: true}
		}
	}
	if search != "" {
		srch = &search
	}
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			start = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// End of day
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			end = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	return
}

// ListActivityLogs returns a paginated list of activity logs with user email and app name.
func (r *Repository) ListActivityLogs(page, pageSize int, eventType, severity, appID, search, startDate, endDate string) ([]ActivityLogListItem, int64, error) {
	ctx := context.Background()
	evtType, sev, appFilter, srch, start, end := activityLogFilterArgs(eventType, severity, appID, search, startDate, endDate)

	total, err := r.queries.AdminCountActivityLogs(ctx, sqlcgen.AdminCountActivityLogsParams{
		EventType: evtType,
		Severity:  sev,
		AppID:     appFilter,
		Search:    srch,
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListActivityLogs(ctx, sqlcgen.AdminListActivityLogsParams{
		Limit:     safeconv.ToInt32(pageSize),
		Offset:    safeconv.ToInt32(offset),
		EventType: evtType,
		Severity:  sev,
		AppID:     appFilter,
		Search:    srch,
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]ActivityLogListItem, len(rows))
	for i, row := range rows {
		items[i] = ActivityLogListItem{
			ID:        row.ID,
			AppID:     row.AppID,
			AppName:   row.AppName,
			UserID:    row.UserID,
			UserEmail: row.UserEmail,
			EventType: row.EventType,
			Severity:  row.Severity,
			IPAddress: row.IpAddress,
			IsAnomaly: row.IsAnomaly,
			Timestamp: row.Timestamp,
		}
	}
	return items, total, nil
}

// GetActivityLogDetail returns a full activity log detail view with user email and app name.
func (r *Repository) GetActivityLogDetail(id string) (*ActivityLogDetail, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetActivityLogDetail(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}

	detailStr := ""
	if row.Details != nil {
		if s, ok := row.Details.(string); ok {
			detailStr = s
		}
	}

	return &ActivityLogDetail{
		ID:        row.ID,
		AppID:     row.AppID,
		AppName:   row.AppName,
		UserID:    row.UserID,
		UserEmail: row.UserEmail,
		EventType: row.EventType,
		Severity:  row.Severity,
		IPAddress: row.IpAddress,
		UserAgent: row.UserAgent,
		Details:   detailStr,
		IsAnomaly: row.IsAnomaly,
		ExpiresAt: timestamptzToTimePtr(row.ExpiresAt),
		Timestamp: row.Timestamp,
	}, nil
}

// ListDistinctEventTypes returns all distinct event types currently in the activity_logs table.
func (r *Repository) ListDistinctEventTypes() ([]string, error) {
	return r.queries.AdminListDistinctEventTypes(context.Background())
}

// ListDistinctSeverities returns all distinct severity levels currently in the activity_logs table.
func (r *Repository) ListDistinctSeverities() ([]string, error) {
	return r.queries.AdminListDistinctSeverities(context.Background())
}

// ExportActivityLogs returns up to ExportActivityLogsMaxRows activity log rows including UserAgent.
func (r *Repository) ExportActivityLogs(eventType, severity, appID, search, startDate, endDate string) ([]ActivityLogExportItem, bool, error) {
	ctx := context.Background()
	evtType, sev, appFilter, srch, start, end := activityLogFilterArgs(eventType, severity, appID, search, startDate, endDate)

	limit := ExportActivityLogsMaxRows + 1
	rows, err := r.queries.AdminExportActivityLogs(ctx, sqlcgen.AdminExportActivityLogsParams{
		Limit:     int32(limit),
		EventType: evtType,
		Severity:  sev,
		AppID:     appFilter,
		Search:    srch,
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		return nil, false, err
	}

	items := make([]ActivityLogExportItem, len(rows))
	for i, row := range rows {
		items[i] = ActivityLogExportItem{
			ActivityLogListItem: ActivityLogListItem{
				ID:        row.ID,
				AppID:     row.AppID,
				AppName:   row.AppName,
				UserID:    row.UserID,
				UserEmail: row.UserEmail,
				EventType: row.EventType,
				Severity:  row.Severity,
				IPAddress: row.IpAddress,
				IsAnomaly: row.IsAnomaly,
				Timestamp: row.Timestamp,
			},
			UserAgent: row.UserAgent,
		}
	}

	truncated := len(items) > ExportActivityLogsMaxRows
	if truncated {
		items = items[:ExportActivityLogsMaxRows]
	}
	return items, truncated, nil
}

// ============================================================
// API Key Operations (Admin GUI - full CRUD)
// ============================================================

// ApiKeyListItem represents an API key row in the admin GUI list view.
type ApiKeyListItem struct {
	ID               uuid.UUID  `json:"id"`
	KeyType          string     `json:"key_type"`
	Name             string     `json:"name"`
	KeyPrefix        string     `json:"key_prefix"`
	KeySuffix        string     `json:"key_suffix"`
	Scopes           string     `json:"scopes"`
	OperatorRoleName string     `json:"operator_role_name"`
	AppID            *uuid.UUID `json:"app_id"`
	AppName          string     `json:"app_name"`
	TenantName       string     `json:"tenant_name"`
	ExpiresAt        *time.Time `json:"expires_at"`
	LastUsedAt       *time.Time `json:"last_used_at"`
	IsRevoked        bool       `json:"is_revoked"`
	CreatedAt        time.Time  `json:"created_at"`
}

// CreateApiKey inserts a new API key record.
func (r *Repository) CreateApiKey(apiKey *models.ApiKey) error {
	if apiKey.ID == uuid.Nil {
		apiKey.ID = uuid.New()
	}
	now := time.Now().UTC()
	if apiKey.CreatedAt.IsZero() {
		apiKey.CreatedAt = now
	}
	if apiKey.UpdatedAt.IsZero() {
		apiKey.UpdatedAt = now
	}

	row, err := r.queries.AdminCreateApiKey(context.Background(), sqlcgen.AdminCreateApiKeyParams{
		ID:             apiKey.ID,
		KeyType:        apiKey.KeyType,
		Name:           apiKey.Name,
		Description:    apiKey.Description,
		KeyHash:        apiKey.KeyHash,
		KeyPrefix:      apiKey.KeyPrefix,
		KeySuffix:      apiKey.KeySuffix,
		AppID:          uuidPtrToPgtype(apiKey.AppID),
		Scopes:         apiKey.Scopes,
		OperatorRoleID: uuidPtrToPgtype(apiKey.OperatorRoleID),
		ExpiresAt:      timePtrToTimestamptz(apiKey.ExpiresAt),
		IsRevoked:      apiKey.IsRevoked,
		CreatedAt:      apiKey.CreatedAt,
		UpdatedAt:      apiKey.UpdatedAt,
	})
	if err != nil {
		return err
	}
	applyApiKeyRowToModel(apiKey, row)
	return nil
}

// ListApiKeys returns a paginated list of API keys with optional type filter.
func (r *Repository) ListApiKeys(page, pageSize int, keyType string) ([]ApiKeyListItem, int64, error) {
	ctx := context.Background()

	var keyTypeFilter *string
	if keyType != "" {
		keyTypeFilter = &keyType
	}

	total, err := r.queries.AdminCountApiKeys(ctx, keyTypeFilter)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListApiKeys(ctx, sqlcgen.AdminListApiKeysParams{
		Limit:   safeconv.ToInt32(pageSize),
		Offset:  safeconv.ToInt32(offset),
		KeyType: keyTypeFilter,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]ApiKeyListItem, len(rows))
	for i, row := range rows {
		items[i] = ApiKeyListItem{
			ID:               row.ID,
			KeyType:          row.KeyType,
			Name:             row.Name,
			KeyPrefix:        row.KeyPrefix,
			KeySuffix:        row.KeySuffix,
			Scopes:           row.Scopes,
			OperatorRoleName: row.OperatorRoleName,
			AppID:            pgtypeUUIDToPtr(row.AppID),
			AppName:          row.AppName,
			TenantName:       row.TenantName,
			ExpiresAt:        timestamptzToTimePtr(row.ExpiresAt),
			LastUsedAt:       timestamptzToTimePtr(row.LastUsedAt),
			IsRevoked:        row.IsRevoked,
			CreatedAt:        row.CreatedAt,
		}
	}
	return items, total, nil
}

// GetApiKeyByID returns a single API key by ID.
func (r *Repository) GetApiKeyByID(id string) (*models.ApiKey, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetApiKeyByID(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	m := sqlcApiKeyToModel(row)
	return &m, nil
}

// RevokeApiKey sets the is_revoked flag to true for an API key.
func (r *Repository) RevokeApiKey(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminRevokeApiKey(context.Background(), uid)
}

// DeleteApiKey permanently deletes an API key by ID.
func (r *Repository) DeleteApiKey(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminDeleteApiKey(context.Background(), uid)
}

// FindActiveKeyByHash looks up an active (non-revoked, non-expired) API key by its SHA-256 hash.
// Returns nil, nil if no matching key is found.
func (r *Repository) FindActiveKeyByHash(keyHash string) (*models.ApiKey, error) {
	row, err := r.queries.AdminFindActiveKeyByHash(context.Background(), keyHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	m := sqlcApiKeyToModel(row)

	// Check expiration
	if m.ExpiresAt != nil && m.ExpiresAt.Before(time.Now()) {
		return nil, nil // Expired
	}

	return &m, nil
}

// UpdateApiKeyLastUsed sets the last_used_at timestamp to now.
func (r *Repository) UpdateApiKeyLastUsed(id uuid.UUID) {
	// Fire-and-forget update; errors are non-critical
	_ = r.queries.AdminUpdateApiKeyLastUsed(context.Background(), id)
}

// UpdateApiKey updates name, description, leftover scopes, operator role, and expiry.
func (r *Repository) UpdateApiKey(id string, name, description, scopes string, operatorRoleID *uuid.UUID, expiresAt *time.Time) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminUpdateApiKey(context.Background(), sqlcgen.AdminUpdateApiKeyParams{
		ID:             uid,
		Name:           name,
		Description:    description,
		Scopes:         scopes,
		OperatorRoleID: uuidPtrToPgtype(operatorRoleID),
		ExpiresAt:      timePtrToTimestamptz(expiresAt),
	})
}

// UpdateApiKeyScopes updates the name, description, and scopes for an existing key.
func (r *Repository) UpdateApiKeyScopes(id string, name, description, scopes string) error {
	existing, err := r.GetApiKeyByID(id)
	if err != nil {
		return err
	}
	return r.UpdateApiKey(id, name, description, scopes, existing.OperatorRoleID, existing.ExpiresAt)
}

// GetKeysExpiringWithin returns all active (non-revoked) API keys expiring within `days` days.
func (r *Repository) GetKeysExpiringWithin(days int) ([]models.ApiKey, error) {
	cutoff := time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)
	rows, err := r.queries.AdminGetKeysExpiringWithin(context.Background(), pgtype.Timestamptz{Time: cutoff, Valid: true})
	if err != nil {
		return nil, err
	}
	keys := make([]models.ApiKey, len(rows))
	for i, row := range rows {
		keys[i] = sqlcApiKeyToModel(row)
	}
	return keys, err
}

// MarkApiKeyNotified7Days sets the notified_7_days_at timestamp to now.
func (r *Repository) MarkApiKeyNotified7Days(id uuid.UUID) error {
	return r.queries.AdminMarkApiKeyNotified7Days(context.Background(), id)
}

// MarkApiKeyNotified1Day sets the notified_1_day_at timestamp to now.
func (r *Repository) MarkApiKeyNotified1Day(id uuid.UUID) error {
	return r.queries.AdminMarkApiKeyNotified1Day(context.Background(), id)
}

// IncrementDailyUsage upserts a daily usage row for the given API key, incrementing the count by 1.
func (r *Repository) IncrementDailyUsage(keyID uuid.UUID) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	_ = r.queries.AdminIncrementDailyUsage(context.Background(), sqlcgen.AdminIncrementDailyUsageParams{
		ApiKeyID:   keyID,
		PeriodDate: pgtype.Date{Time: today, Valid: true},
	})
}

// ApiKeyUsagePoint is a single data point for usage analytics (one day).
type ApiKeyUsagePoint struct {
	PeriodDate   time.Time `json:"period_date"`
	RequestCount int64     `json:"request_count"`
}

// GetApiKeyUsageSummary returns daily usage data for a key over the last `days` days.
func (r *Repository) GetApiKeyUsageSummary(keyID uuid.UUID, days int) ([]ApiKeyUsagePoint, error) {
	since := time.Now().UTC().Truncate(24 * time.Hour).Add(-time.Duration(days-1) * 24 * time.Hour)
	rows, err := r.queries.AdminGetApiKeyUsageSummary(context.Background(), sqlcgen.AdminGetApiKeyUsageSummaryParams{
		ApiKeyID:   keyID,
		PeriodDate: pgtype.Date{Time: since, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	points := make([]ApiKeyUsagePoint, len(rows))
	for i, row := range rows {
		points[i] = ApiKeyUsagePoint{
			PeriodDate:   row.PeriodDate.Time,
			RequestCount: row.RequestCount,
		}
	}
	return points, nil
}

// GetApiKeyTotalUsage returns the lifetime total request count for a key.
func (r *Repository) GetApiKeyTotalUsage(keyID uuid.UUID) (int64, error) {
	return r.queries.AdminGetApiKeyTotalUsage(context.Background(), keyID)
}

// ============================================================
// Social Account Operations (Admin GUI - unlink support)
// ============================================================

// GetSocialAccountByID returns a single social account by primary key.
func (r *Repository) GetSocialAccountByID(id string) (*models.SocialAccount, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetSocialAccountByID(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	m := sqlcSocialAccountToModel(row)
	return &m, nil
}

// DeleteSocialAccount permanently removes a social account by ID.
func (r *Repository) DeleteSocialAccount(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminDeleteSocialAccount(context.Background(), uid)
}

// CountSocialAccountsByUserID returns the number of social accounts linked to a user.
func (r *Repository) CountSocialAccountsByUserID(userID string) (int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, err
	}
	return r.queries.AdminCountSocialAccountsByUserID(context.Background(), uid)
}

// ============================================================
// WebAuthn Passkey Operations (Admin GUI - delete support)
// ============================================================

// GetWebAuthnCredentialByID returns a single WebAuthn credential by primary key.
func (r *Repository) GetWebAuthnCredentialByID(id string) (*models.WebAuthnCredential, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetWebAuthnCredentialByID(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	m := sqlcWebAuthnToModel(row)
	return &m, nil
}

// DeleteWebAuthnCredential permanently removes a WebAuthn credential by ID.
func (r *Repository) DeleteWebAuthnCredential(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminDeleteWebAuthnCredential(context.Background(), uid)
}

// ============================================================
// Session Operations (Admin GUI - session management)
// ============================================================

// GetUserIDByEmailAndApp returns the user ID (as a string) for the given email
// within the given application. Returns ("", nil) when no matching user exists.
func (r *Repository) GetUserIDByEmailAndApp(appID, email string) (string, error) {
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return "", err
	}
	uid, err := r.queries.AdminGetUserIDByEmailAndApp(context.Background(), sqlcgen.AdminGetUserIDByEmailAndAppParams{
		AppID: appUUID,
		Email: email,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return uid.String(), nil
}

// GetUserEmailsByIDs returns a map of userID -> email for the given user IDs.
func (r *Repository) GetUserEmailsByIDs(userIDs []string) (map[string]string, error) {
	if len(userIDs) == 0 {
		return map[string]string{}, nil
	}
	uuids := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		if uid, err := uuid.Parse(id); err == nil {
			uuids = append(uuids, uid)
		}
	}
	rows, err := r.queries.AdminGetUserEmailsByIDs(context.Background(), uuids)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.ID.String()] = row.Email
	}
	return result, nil
}

// GetAppNamesByIDs returns a map of appID -> appName for the given application IDs.
func (r *Repository) GetAppNamesByIDs(appIDs []string) (map[string]string, error) {
	if len(appIDs) == 0 {
		return map[string]string{}, nil
	}
	uuids := make([]uuid.UUID, 0, len(appIDs))
	for _, id := range appIDs {
		if uid, err := uuid.Parse(id); err == nil {
			uuids = append(uuids, uid)
		}
	}
	rows, err := r.queries.AdminGetAppNamesByIDs(context.Background(), uuids)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.ID.String()] = row.Name
	}
	return result, nil
}

// ============================================================
// User Export / Import Operations
// ============================================================

// ExportUsersMaxRows is the hard cap for user export operations.
const ExportUsersMaxRows = 10_000

// UserExportItem is a flat projection used for user export queries.
type UserExportItem struct {
	ID              uuid.UUID `json:"id"`
	AppID           uuid.UUID `json:"app_id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Locale          string    `json:"locale"`
	EmailVerified   bool      `json:"email_verified"`
	IsActive        bool      `json:"is_active"`
	TwoFAEnabled    bool      `json:"two_fa_enabled"`
	TwoFAMethod     string    `json:"two_fa_method"`
	SocialProviders string    `json:"social_providers"` // STRING_AGG result, comma-separated
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ExportUsers returns up to ExportUsersMaxRows user rows.
func (r *Repository) ExportUsers(appID, search string) ([]UserExportItem, bool, error) {
	ctx := context.Background()

	var appFilter pgtype.UUID
	if appID != "" {
		if uid, err := uuid.Parse(appID); err == nil {
			appFilter = pgtype.UUID{Bytes: uid, Valid: true}
		}
	}
	var searchFilter *string
	if search != "" {
		searchFilter = &search
	}

	limit := ExportUsersMaxRows + 1
	rows, err := r.queries.AdminExportUsers(ctx, sqlcgen.AdminExportUsersParams{
		Limit:  int32(limit),
		AppID:  appFilter,
		Search: searchFilter,
	})
	if err != nil {
		return nil, false, err
	}

	items := make([]UserExportItem, len(rows))
	for i, row := range rows {
		items[i] = UserExportItem{
			ID:              row.ID,
			AppID:           row.AppID,
			Email:           row.Email,
			Name:            row.Name,
			FirstName:       row.FirstName,
			LastName:        row.LastName,
			Locale:          row.Locale,
			EmailVerified:   row.EmailVerified,
			IsActive:        row.IsActive,
			TwoFAEnabled:    row.TwoFaEnabled,
			TwoFAMethod:     row.TwoFaMethod,
			SocialProviders: string(row.SocialProviders),
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		}
	}

	truncated := len(items) > ExportUsersMaxRows
	if truncated {
		items = items[:ExportUsersMaxRows]
	}
	return items, truncated, nil
}

// ImportUsers bulk-creates users from the provided rows under the given appID.
func (r *Repository) ImportUsers(appID string, rows []dto.UserImportRow) (dto.UserImportResult, error) {
	result := dto.UserImportResult{Total: len(rows)}

	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return result, fmt.Errorf("invalid app_id %q: %w", appID, err)
	}

	ctx := context.Background()
	for i, row := range rows {
		rowNum := i + 1
		normalizedEmail := strings.ToLower(strings.TrimSpace(row.Email))

		// Check for existing (email, app_id) pair
		count, err := r.queries.AdminCountUserExistsForImport(ctx, sqlcgen.AdminCountUserExistsForImportParams{
			Email: normalizedEmail,
			AppID: appUUID,
		})
		if err != nil {
			result.Errors = append(result.Errors, dto.UserImportRowError{
				Row:   rowNum,
				Email: row.Email,
				Error: "database error checking duplicate",
			})
			continue
		}

		if count > 0 {
			result.Skipped++
			result.Errors = append(result.Errors, dto.UserImportRowError{
				Row:   rowNum,
				Email: row.Email,
				Error: "skipped: email already exists in this application",
			})
			continue
		}

		_, err = r.queries.AdminCreateImportUser(ctx, sqlcgen.AdminCreateImportUserParams{
			ID:        uuid.New(),
			AppID:     appUUID,
			Email:     normalizedEmail,
			Name:      strings.TrimSpace(row.Name),
			FirstName: strings.TrimSpace(row.FirstName),
			LastName:  strings.TrimSpace(row.LastName),
			Locale:    strings.TrimSpace(row.Locale),
		})
		if err != nil {
			result.Errors = append(result.Errors, dto.UserImportRowError{
				Row:   rowNum,
				Email: row.Email,
				Error: "failed to create user",
			})
			continue
		}
		result.Imported++
	}

	return result, nil
}

// GetUsersByEmailsAndApp returns a slice of {ID, Email} for users matching the given emails in the app.
// Used for post-import activity log correlation.
func (r *Repository) GetUsersByEmailsAndApp(emails []string, appID uuid.UUID) ([]sqlcgen.AdminGetUsersByEmailsAndAppRow, error) {
	return r.queries.AdminGetUsersByEmailsAndApp(context.Background(), sqlcgen.AdminGetUsersByEmailsAndAppParams{
		Column1: emails,
		AppID:   appID,
	})
}

// ============================================================
// Session Group Operations
// ============================================================

// SessionGroupListItem holds a session group with tenant name and app count for list views.
type SessionGroupListItem struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	TenantName   string
	Name         string
	Description  string
	GlobalLogout bool
	AppCount     int64
	CreatedAt    time.Time
}

// CreateSessionGroup inserts a new session group.
func (r *Repository) CreateSessionGroup(sg *models.SessionGroup) error {
	if sg.ID == uuid.Nil {
		sg.ID = uuid.New()
	}
	now := time.Now().UTC()
	if sg.CreatedAt.IsZero() {
		sg.CreatedAt = now
	}
	if sg.UpdatedAt.IsZero() {
		sg.UpdatedAt = now
	}
	row, err := r.queries.AdminCreateSessionGroup(context.Background(), sqlcgen.AdminCreateSessionGroupParams{
		ID:           sg.ID,
		TenantID:     sg.TenantID,
		Name:         sg.Name,
		Description:  sg.Description,
		GlobalLogout: sg.GlobalLogout,
		CreatedAt:    sg.CreatedAt,
		UpdatedAt:    sg.UpdatedAt,
	})
	if err != nil {
		return err
	}
	sg.ID = row.ID
	return nil
}

// GetSessionGroupByID returns a session group by its UUID string.
func (r *Repository) GetSessionGroupByID(id string) (*models.SessionGroup, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetSessionGroupByID(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	return &models.SessionGroup{
		ID:           row.ID,
		TenantID:     row.TenantID,
		Name:         row.Name,
		Description:  row.Description,
		GlobalLogout: row.GlobalLogout,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

// ListSessionGroups returns paginated session groups with tenant name and app count.
func (r *Repository) ListSessionGroups(page, pageSize int) ([]SessionGroupListItem, int64, error) {
	ctx := context.Background()
	total, err := r.queries.AdminCountSessionGroups(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.AdminListSessionGroups(ctx, sqlcgen.AdminListSessionGroupsParams{
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]SessionGroupListItem, len(rows))
	for i, row := range rows {
		items[i] = SessionGroupListItem{
			ID:           row.ID,
			TenantID:     row.TenantID,
			TenantName:   ptrToStr(row.TenantName),
			Name:         row.Name,
			Description:  row.Description,
			GlobalLogout: row.GlobalLogout,
			AppCount:     row.AppCount,
			CreatedAt:    row.CreatedAt,
		}
	}
	return items, total, nil
}

// UpdateSessionGroup updates the mutable fields of a session group.
func (r *Repository) UpdateSessionGroup(id, name, description string, globalLogout bool) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return r.queries.AdminUpdateSessionGroup(context.Background(), sqlcgen.AdminUpdateSessionGroupParams{
		ID:           uid,
		Name:         name,
		Description:  description,
		GlobalLogout: globalLogout,
	})
}

// DeleteSessionGroup deletes a session group and its app memberships.
func (r *Repository) DeleteSessionGroup(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	ctx := context.Background()
	// Delete memberships first
	if err := r.queries.AdminDeleteSessionGroupApps(ctx, uid); err != nil {
		return err
	}
	return r.queries.AdminDeleteSessionGroup(ctx, uid)
}

// AddAppToSessionGroup adds an application to a session group.
func (r *Repository) AddAppToSessionGroup(groupID, appID string) error {
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return errors.New("invalid group ID")
	}
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return errors.New("invalid app ID")
	}
	return r.queries.AdminAddAppToSessionGroup(context.Background(), sqlcgen.AdminAddAppToSessionGroupParams{
		SessionGroupID: groupUUID,
		AppID:          appUUID,
	})
}

// RemoveAppFromSessionGroup removes an application from a session group.
func (r *Repository) RemoveAppFromSessionGroup(groupID, appID string) error {
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return err
	}
	appUUID, err := uuid.Parse(appID)
	if err != nil {
		return err
	}
	return r.queries.AdminRemoveAppFromSessionGroup(context.Background(), sqlcgen.AdminRemoveAppFromSessionGroupParams{
		SessionGroupID: groupUUID,
		AppID:          appUUID,
	})
}

// GetSessionGroupForApp returns the session group that the given appID belongs to.
// Returns (nil, nil) when the app is not in any group.
func (r *Repository) GetSessionGroupForApp(appID string) (*models.SessionGroup, error) {
	uid, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AdminGetSessionGroupForApp(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &models.SessionGroup{
		ID:           row.ID,
		TenantID:     row.TenantID,
		Name:         row.Name,
		Description:  row.Description,
		GlobalLogout: row.GlobalLogout,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

// GetAppsInSessionGroup returns all app IDs (as strings) that belong to the given group.
func (r *Repository) GetAppsInSessionGroup(groupID string) ([]string, error) {
	uid, err := uuid.Parse(groupID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.AdminGetAppsInSessionGroup(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.String())
	}
	return ids, nil
}

// GetPeersForApp returns all peer applications (excluding the requesting app) in the
// same session group, with their app_id and frontend_url as origin.
func (r *Repository) GetPeersForApp(appID string) ([]sso.SSOPeerInfo, error) {
	uid, err := uuid.Parse(appID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.AdminGetPeersForApp(context.Background(), uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []sso.SSOPeerInfo{}, nil
		}
		return nil, err
	}

	peers := make([]sso.SSOPeerInfo, 0, len(rows))
	for _, row := range rows {
		if row.FrontendUrl == "" {
			continue // skip apps without a frontend URL
		}
		peers = append(peers, sso.SSOPeerInfo{
			AppID:  row.SgaAppID,
			Origin: row.FrontendUrl,
		})
	}
	return peers, nil
}

// SessionGroupAppDetail holds an app's membership info within a session group.
type SessionGroupAppDetail struct {
	AppID      uuid.UUID
	AppName    string
	TenantName string
}

// GetAppsInSessionGroupWithDetails returns apps in a session group with their names and tenant names.
func (r *Repository) GetAppsInSessionGroupWithDetails(groupID string) ([]SessionGroupAppDetail, error) {
	uid, err := uuid.Parse(groupID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.AdminGetAppsInSessionGroupWithDetails(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	items := make([]SessionGroupAppDetail, len(rows))
	for i, row := range rows {
		items[i] = SessionGroupAppDetail{
			AppID:      row.AppID,
			AppName:    row.AppName,
			TenantName: ptrToStr(row.TenantName),
		}
	}
	return items, nil
}

// ============================================================
// Conversion helpers
// ============================================================

func sqlcAppToModel(row sqlcgen.Application) models.Application {
	return models.Application{
		ID:                        row.ID,
		TenantID:                  row.TenantID,
		Name:                      row.Name,
		Description:               ptrToStr(row.Description),
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
		BfLockoutThreshold:        int32PtrToIntPtr(row.BfLockoutThreshold),
		BfLockoutDurations:        row.BfLockoutDurations,
		BfLockoutWindow:           row.BfLockoutWindow,
		BfLockoutTierTTL:          row.BfLockoutTierTtl,
		BfDelayEnabled:            row.BfDelayEnabled,
		BfDelayStartAfter:         int32PtrToIntPtr(row.BfDelayStartAfter),
		BfDelayMaxSeconds:         int32PtrToIntPtr(row.BfDelayMaxSeconds),
		BfDelayTierTTL:            row.BfDelayTierTtl,
		BfCaptchaEnabled:          row.BfCaptchaEnabled,
		BfCaptchaSiteKey:          row.BfCaptchaSiteKey,
		BfCaptchaSecretKey:        row.BfCaptchaSecretKey,
		BfCaptchaThreshold:        int32PtrToIntPtr(row.BfCaptchaThreshold),
		FrontendURL:               row.FrontendUrl,
		OIDCEnabled:               row.OidcEnabled,
		OIDCRSAPrivateKey:         row.OidcRsaPrivateKey,
		OIDCIDTokenTTL:            int(row.OidcIDTokenTtl),
		OIDCIssuerURL:             row.OidcIssuerUrl,
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
}

func applySqlcAppToModel(app *models.Application, row sqlcgen.Application) {
	*app = sqlcAppToModel(row)
}

func sqlcOAuthToModel(row sqlcgen.OauthProviderConfig) models.OAuthProviderConfig {
	return models.OAuthProviderConfig{
		ID:           row.ID,
		AppID:        row.AppID,
		Provider:     row.Provider,
		ClientID:     row.ClientID,
		ClientSecret: row.ClientSecret,
		RedirectURL:  row.RedirectUrl,
		IsEnabled:    row.IsEnabled,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func sqlcApiKeyToModel(row sqlcgen.ApiKey) models.ApiKey {
	return models.ApiKey{
		ID:              row.ID,
		KeyType:         row.KeyType,
		Name:            row.Name,
		Description:     row.Description,
		KeyHash:         row.KeyHash,
		KeyPrefix:       row.KeyPrefix,
		KeySuffix:       row.KeySuffix,
		AppID:           pgtypeUUIDToPtr(row.AppID),
		Scopes:          row.Scopes,
		OperatorRoleID:  pgtypeUUIDToPtr(row.OperatorRoleID),
		ExpiresAt:       timestamptzToTimePtr(row.ExpiresAt),
		LastUsedAt:      timestamptzToTimePtr(row.LastUsedAt),
		Notified7DaysAt: timestamptzToTimePtr(row.Notified7DaysAt),
		Notified1DayAt:  timestamptzToTimePtr(row.Notified1DayAt),
		IsRevoked:       row.IsRevoked,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func applyApiKeyRowToModel(m *models.ApiKey, row sqlcgen.ApiKey) {
	*m = sqlcApiKeyToModel(row)
}

func sqlcSocialAccountToModel(row sqlcgen.SocialAccount) models.SocialAccount {
	return models.SocialAccount{
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

func sqlcWebAuthnToModel(row sqlcgen.WebauthnCredential) models.WebAuthnCredential {
	return models.WebAuthnCredential{
		ID:              row.ID,
		UserID:          pgtypeUUIDToPtr(row.UserID),
		AppID:           pgtypeUUIDToPtr(row.AppID),
		AdminID:         pgtypeUUIDToPtr(row.AdminID),
		CredentialID:    row.CredentialID,
		PublicKey:       row.PublicKey,
		AttestationType: ptrToStr(row.AttestationType),
		AAGUID:          row.Aaguid,
		SignCount:       safeconv.Int32ToUint32(row.SignCount),
		Name:            ptrToStr(row.Name),
		Transports:      ptrToStr(row.Transports),
		BackupEligible:  row.BackupEligible,
		BackupState:     row.BackupState,
		LastUsedAt:      timestamptzToTimePtr(row.LastUsedAt),
		CreatedAt:       row.CreatedAt,
	}
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

func uuidPtrToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

func pgtypeUUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

func strToPtr(s string) *string {
	return &s
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	i := safeconv.ToInt32(*v)
	return &i
}

func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	i := int(*v)
	return &i
}
