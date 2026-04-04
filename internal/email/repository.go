package email

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JedidiahDigital/go-core/internal/safeconv"
	"github.com/JedidiahDigital/go-core/internal/sqlcgen"
	"github.com/JedidiahDigital/go-core/pkg/models"
)

// Repository handles database operations for the email system.
type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewRepository creates a new email Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// ============================================================================
// Email Server Config operations
// ============================================================================

// GetServerConfig returns the default active SMTP configuration for a specific application.
// Returns nil, nil if no per-app config exists.
func (r *Repository) GetServerConfig(appID uuid.UUID) (*models.EmailServerConfig, error) {
	ctx := context.Background()
	pgAppID := uuidToPgtypeVal(appID)

	// Try active default first
	row, err := r.queries.GetServerConfigActiveDefault(ctx, pgAppID)
	if err == nil {
		return serverConfigRowToModel(row), nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	// Fallback: any active config for this app
	row, err = r.queries.GetServerConfigActiveAny(ctx, pgAppID)
	if err == nil {
		return serverConfigRowToModel(row), nil
	}
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// GetServerConfigByID returns an SMTP configuration by its ID.
func (r *Repository) GetServerConfigByID(id uuid.UUID) (*models.EmailServerConfig, error) {
	row, err := r.queries.GetServerConfigByID(context.Background(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return serverConfigRowToModel(row), nil
}

// GetServerConfigAny returns the first SMTP configuration for an application regardless of is_active.
// Used for admin listing and backward compatibility checks.
// Returns nil, nil if no config exists for the app.
func (r *Repository) GetServerConfigAny(appID uuid.UUID) (*models.EmailServerConfig, error) {
	row, err := r.queries.GetServerConfigAnyByApp(context.Background(), uuidToPgtypeVal(appID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return serverConfigRowToModel(row), nil
}

// GetServerConfigsByApp returns all SMTP configurations for a specific application.
func (r *Repository) GetServerConfigsByApp(appID uuid.UUID) ([]models.EmailServerConfig, error) {
	rows, err := r.queries.GetServerConfigsByApp(context.Background(), uuidToPgtypeVal(appID))
	if err != nil {
		return nil, err
	}
	configs := make([]models.EmailServerConfig, len(rows))
	for i, row := range rows {
		configs[i] = *serverConfigRowToModel(row)
	}
	return configs, nil
}

// GetAllServerConfigs returns all SMTP configurations across all applications and global.
// Global configs (app_id IS NULL) are returned first.
func (r *Repository) GetAllServerConfigs() ([]models.EmailServerConfig, error) {
	rows, err := r.queries.GetAllServerConfigs(context.Background())
	if err != nil {
		return nil, err
	}
	configs := make([]models.EmailServerConfig, len(rows))
	for i, row := range rows {
		configs[i] = *serverConfigRowToModel(row)
	}
	return configs, nil
}

// ListAllActiveServerConfigs returns all active SMTP configurations across all applications.
// Deprecated: Use GetGlobalServerConfig() for admin/system emails instead.
// Kept for backward compatibility with any external callers.
func (r *Repository) ListAllActiveServerConfigs() ([]models.EmailServerConfig, error) {
	rows, err := r.queries.ListAllActiveServerConfigs(context.Background())
	if err != nil {
		return nil, err
	}
	configs := make([]models.EmailServerConfig, len(rows))
	for i, row := range rows {
		configs[i] = *serverConfigRowToModel(row)
	}
	return configs, nil
}

// CreateServerConfig creates a new SMTP configuration for an application.
func (r *Repository) CreateServerConfig(config *models.EmailServerConfig) error {
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
	return r.queries.CreateServerConfig(context.Background(), sqlcgen.CreateServerConfigParams{
		ID:           config.ID,
		AppID:        uuidPtrToPgtype(config.AppID),
		Name:         config.Name,
		SmtpHost:     config.SMTPHost,
		SmtpPort:     safeconv.ToInt32(config.SMTPPort),
		SmtpUsername: &config.SMTPUsername,
		SmtpPassword: &config.SMTPPassword,
		FromAddress:  config.FromAddress,
		FromName:     &config.FromName,
		UseTls:       config.UseTLS,
		IsActive:     config.IsActive,
		IsDefault:    config.IsDefault,
		CreatedAt:    config.CreatedAt,
		UpdatedAt:    config.UpdatedAt,
	})
}

// UpdateServerConfig updates an existing SMTP configuration.
func (r *Repository) UpdateServerConfig(config *models.EmailServerConfig) error {
	config.UpdatedAt = time.Now().UTC()
	return r.queries.UpdateServerConfig(context.Background(), sqlcgen.UpdateServerConfigParams{
		ID:           config.ID,
		AppID:        uuidPtrToPgtype(config.AppID),
		Name:         config.Name,
		SmtpHost:     config.SMTPHost,
		SmtpPort:     safeconv.ToInt32(config.SMTPPort),
		SmtpUsername: &config.SMTPUsername,
		SmtpPassword: &config.SMTPPassword,
		FromAddress:  config.FromAddress,
		FromName:     &config.FromName,
		UseTls:       config.UseTLS,
		IsActive:     config.IsActive,
		IsDefault:    config.IsDefault,
		UpdatedAt:    config.UpdatedAt,
	})
}

// DeleteServerConfig removes the SMTP configuration for an application.
func (r *Repository) DeleteServerConfig(appID uuid.UUID) error {
	return r.queries.DeleteServerConfigByApp(context.Background(), uuidToPgtypeVal(appID))
}

// DeleteServerConfigByID removes a specific SMTP configuration by its ID.
func (r *Repository) DeleteServerConfigByID(id uuid.UUID) error {
	return r.queries.DeleteServerConfigByID(context.Background(), id)
}

// ClearDefaultFlag unsets is_default on all configs for the same scope (app or global).
// If appID is nil, clears the default flag on global configs (app_id IS NULL).
// If appID is set, clears the default flag on configs for that specific app.
func (r *Repository) ClearDefaultFlag(appID *uuid.UUID) error {
	ctx := context.Background()
	if appID == nil {
		return r.queries.ClearDefaultFlagGlobal(ctx)
	}
	return r.queries.ClearDefaultFlagByApp(ctx, uuidToPgtypeVal(*appID))
}

// GetGlobalServerConfig returns the active default global SMTP configuration (app_id IS NULL).
// Returns nil, nil if no global config exists.
func (r *Repository) GetGlobalServerConfig() (*models.EmailServerConfig, error) {
	ctx := context.Background()

	// Try active default first
	row, err := r.queries.GetGlobalServerConfigActiveDefault(ctx)
	if err == nil {
		return serverConfigRowToModel(row), nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	// Fallback: any active global config
	row, err = r.queries.GetGlobalServerConfigActiveAny(ctx)
	if err == nil {
		return serverConfigRowToModel(row), nil
	}
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// ============================================================================
// Email Type operations
// ============================================================================

// GetAllEmailTypes returns all email types.
func (r *Repository) GetAllEmailTypes() ([]models.EmailType, error) {
	rows, err := r.queries.GetAllEmailTypes(context.Background())
	if err != nil {
		return nil, err
	}
	types := make([]models.EmailType, len(rows))
	for i, row := range rows {
		types[i] = emailTypeRowToModel(row)
	}
	return types, nil
}

// GetActiveEmailTypes returns all active email types.
func (r *Repository) GetActiveEmailTypes() ([]models.EmailType, error) {
	rows, err := r.queries.GetActiveEmailTypes(context.Background())
	if err != nil {
		return nil, err
	}
	types := make([]models.EmailType, len(rows))
	for i, row := range rows {
		types[i] = emailTypeRowToModel(row)
	}
	return types, nil
}

// GetEmailTypeByCode returns an email type by its unique code.
func (r *Repository) GetEmailTypeByCode(code string) (*models.EmailType, error) {
	row, err := r.queries.GetEmailTypeByCode(context.Background(), code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	m := emailTypeRowToModel(row)
	return &m, nil
}

// GetEmailTypeByID returns an email type by its ID.
func (r *Repository) GetEmailTypeByID(id uuid.UUID) (*models.EmailType, error) {
	row, err := r.queries.GetEmailTypeByID(context.Background(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	m := emailTypeRowToModel(row)
	return &m, nil
}

// CreateEmailType creates a new email type.
func (r *Repository) CreateEmailType(emailType *models.EmailType) error {
	if emailType.ID == uuid.Nil {
		emailType.ID = uuid.New()
	}
	now := time.Now().UTC()
	if emailType.CreatedAt.IsZero() {
		emailType.CreatedAt = now
	}
	if emailType.UpdatedAt.IsZero() {
		emailType.UpdatedAt = now
	}
	return r.queries.CreateEmailType(context.Background(), sqlcgen.CreateEmailTypeParams{
		ID:             emailType.ID,
		Code:           emailType.Code,
		Name:           emailType.Name,
		Description:    strOrNil(emailType.Description),
		DefaultSubject: strOrNil(emailType.DefaultSubject),
		Variables:      jsonOrDefault(emailType.Variables),
		IsSystem:       emailType.IsSystem,
		IsActive:       emailType.IsActive,
		CreatedAt:      emailType.CreatedAt,
		UpdatedAt:      emailType.UpdatedAt,
	})
}

// UpdateEmailType updates an existing email type.
func (r *Repository) UpdateEmailType(emailType *models.EmailType) error {
	emailType.UpdatedAt = time.Now().UTC()
	return r.queries.UpdateEmailType(context.Background(), sqlcgen.UpdateEmailTypeParams{
		ID:             emailType.ID,
		Code:           emailType.Code,
		Name:           emailType.Name,
		Description:    strOrNil(emailType.Description),
		DefaultSubject: strOrNil(emailType.DefaultSubject),
		Variables:      jsonOrDefault(emailType.Variables),
		IsSystem:       emailType.IsSystem,
		IsActive:       emailType.IsActive,
		UpdatedAt:      emailType.UpdatedAt,
	})
}

// DeleteEmailType deletes a custom email type (only non-system types).
// Also deletes any templates associated with this type.
func (r *Repository) DeleteEmailType(id uuid.UUID) error {
	ctx := context.Background()

	// Use a transaction for atomicity
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	// Delete associated templates first
	if err := qtx.DeleteTemplatesByEmailTypeID(ctx, id); err != nil {
		return err
	}

	// Delete the type (only if non-system)
	_, err = qtx.DeleteNonSystemEmailType(ctx, id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ============================================================================
// Email Template operations
// ============================================================================

// GetTemplate resolves the template for a given app and email type code.
// Resolution order: app-specific -> global default (app_id IS NULL) -> nil (use hardcoded).
func (r *Repository) GetTemplate(appID uuid.UUID, typeCode string) (*models.EmailTemplate, error) {
	ctx := context.Background()

	// First get the email type
	emailType, err := r.GetEmailTypeByCode(typeCode)
	if err != nil {
		return nil, err
	}
	if emailType == nil {
		return nil, nil
	}

	// Try app-specific template first
	row, err := r.queries.GetTemplateByAppAndType(ctx, sqlcgen.GetTemplateByAppAndTypeParams{
		AppID:       uuidToPgtypeVal(appID),
		EmailTypeID: emailType.ID,
	})
	if err == nil {
		tmpl := templateRowToModel(row)
		tmpl.EmailType = *emailType
		return &tmpl, nil
	}
	if err != pgx.ErrNoRows {
		return nil, err
	}

	// Fall back to global default template (app_id IS NULL)
	row2, err := r.queries.GetTemplateGlobalByType(ctx, emailType.ID)
	if err == nil {
		tmpl := templateRowToModel(row2)
		tmpl.EmailType = *emailType
		return &tmpl, nil
	}
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return nil, err
}

// GetTemplatesByApp returns all templates for a specific application.
func (r *Repository) GetTemplatesByApp(appID uuid.UUID) ([]models.EmailTemplate, error) {
	rows, err := r.queries.GetTemplatesByApp(context.Background(), uuidToPgtypeVal(appID))
	if err != nil {
		return nil, err
	}
	templates := make([]models.EmailTemplate, len(rows))
	for i, row := range rows {
		templates[i] = templateWithTypeRowToModel(row.ID, row.AppID, row.EmailTypeID,
			row.ServerConfigID, row.Name, row.Subject, row.BodyHtml, row.BodyText,
			row.FromEmail, row.FromName, row.TemplateEngine, row.IsActive,
			row.CreatedAt, row.UpdatedAt,
			row.EtID, row.EtCode, row.EtName, row.EtDescription, row.EtDefaultSubject,
			row.EtVariables, row.EtIsSystem, row.EtIsActive, row.EtCreatedAt, row.EtUpdatedAt)
	}
	return templates, nil
}

// GetGlobalDefaultTemplates returns all global default templates (app_id IS NULL).
func (r *Repository) GetGlobalDefaultTemplates() ([]models.EmailTemplate, error) {
	rows, err := r.queries.GetGlobalDefaultTemplates(context.Background())
	if err != nil {
		return nil, err
	}
	templates := make([]models.EmailTemplate, len(rows))
	for i, row := range rows {
		templates[i] = templateWithTypeRowToModel(row.ID, row.AppID, row.EmailTypeID,
			row.ServerConfigID, row.Name, row.Subject, row.BodyHtml, row.BodyText,
			row.FromEmail, row.FromName, row.TemplateEngine, row.IsActive,
			row.CreatedAt, row.UpdatedAt,
			row.EtID, row.EtCode, row.EtName, row.EtDescription, row.EtDefaultSubject,
			row.EtVariables, row.EtIsSystem, row.EtIsActive, row.EtCreatedAt, row.EtUpdatedAt)
	}
	return templates, nil
}

// GetTemplateByID returns a template by its ID.
func (r *Repository) GetTemplateByID(id uuid.UUID) (*models.EmailTemplate, error) {
	row, err := r.queries.GetTemplateByID(context.Background(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	tmpl := templateWithTypeRowToModel(row.ID, row.AppID, row.EmailTypeID,
		row.ServerConfigID, row.Name, row.Subject, row.BodyHtml, row.BodyText,
		row.FromEmail, row.FromName, row.TemplateEngine, row.IsActive,
		row.CreatedAt, row.UpdatedAt,
		row.EtID, row.EtCode, row.EtName, row.EtDescription, row.EtDefaultSubject,
		row.EtVariables, row.EtIsSystem, row.EtIsActive, row.EtCreatedAt, row.EtUpdatedAt)
	return &tmpl, nil
}

// CreateTemplate creates a new email template.
func (r *Repository) CreateTemplate(template *models.EmailTemplate) error {
	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}
	now := time.Now().UTC()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	}
	if template.UpdatedAt.IsZero() {
		template.UpdatedAt = now
	}
	return r.queries.CreateTemplate(context.Background(), sqlcgen.CreateTemplateParams{
		ID:             template.ID,
		AppID:          uuidPtrToPgtype(template.AppID),
		EmailTypeID:    template.EmailTypeID,
		ServerConfigID: uuidPtrToPgtype(template.ServerConfigID),
		Name:           template.Name,
		Subject:        template.Subject,
		BodyHtml:       &template.BodyHTML,
		BodyText:       &template.BodyText,
		FromEmail:      &template.FromEmail,
		FromName:       &template.FromName,
		TemplateEngine: template.TemplateEngine,
		IsActive:       template.IsActive,
		CreatedAt:      template.CreatedAt,
		UpdatedAt:      template.UpdatedAt,
	})
}

// UpdateTemplate updates an existing email template.
func (r *Repository) UpdateTemplate(template *models.EmailTemplate) error {
	template.UpdatedAt = time.Now().UTC()
	return r.queries.UpdateTemplate(context.Background(), sqlcgen.UpdateTemplateParams{
		ID:             template.ID,
		AppID:          uuidPtrToPgtype(template.AppID),
		EmailTypeID:    template.EmailTypeID,
		ServerConfigID: uuidPtrToPgtype(template.ServerConfigID),
		Name:           template.Name,
		Subject:        template.Subject,
		BodyHtml:       &template.BodyHTML,
		BodyText:       &template.BodyText,
		FromEmail:      &template.FromEmail,
		FromName:       &template.FromName,
		TemplateEngine: template.TemplateEngine,
		IsActive:       template.IsActive,
		UpdatedAt:      template.UpdatedAt,
	})
}

// DeleteTemplate removes an email template.
func (r *Repository) DeleteTemplate(id uuid.UUID) error {
	return r.queries.DeleteTemplateByID(context.Background(), id)
}

// UpsertAppTemplate creates or updates a template for a specific app and email type.
func (r *Repository) UpsertAppTemplate(appID uuid.UUID, emailTypeID uuid.UUID, template *models.EmailTemplate) error {
	ctx := context.Background()

	// Check if one already exists
	existing, err := r.queries.GetTemplateByAppAndTypeID(ctx, sqlcgen.GetTemplateByAppAndTypeIDParams{
		AppID:       uuidToPgtypeVal(appID),
		EmailTypeID: emailTypeID,
	})
	if err == nil {
		// Update existing
		return r.queries.UpdateTemplate(ctx, sqlcgen.UpdateTemplateParams{
			ID:             existing.ID,
			AppID:          existing.AppID,
			EmailTypeID:    existing.EmailTypeID,
			ServerConfigID: uuidPtrToPgtype(template.ServerConfigID),
			Name:           template.Name,
			Subject:        template.Subject,
			BodyHtml:       &template.BodyHTML,
			BodyText:       &template.BodyText,
			FromEmail:      &template.FromEmail,
			FromName:       &template.FromName,
			TemplateEngine: template.TemplateEngine,
			IsActive:       template.IsActive,
			UpdatedAt:      time.Now().UTC(),
		})
	}
	if err != pgx.ErrNoRows {
		return err
	}

	// Create new
	template.AppID = &appID
	template.EmailTypeID = emailTypeID
	return r.CreateTemplate(template)
}

// UpsertGlobalTemplate creates or updates a global default template for an email type.
func (r *Repository) UpsertGlobalTemplate(emailTypeID uuid.UUID, template *models.EmailTemplate) error {
	ctx := context.Background()

	existing, err := r.queries.GetTemplateGlobalByTypeID(ctx, emailTypeID)
	if err == nil {
		// Update existing
		return r.queries.UpdateTemplate(ctx, sqlcgen.UpdateTemplateParams{
			ID:             existing.ID,
			AppID:          existing.AppID,
			EmailTypeID:    existing.EmailTypeID,
			ServerConfigID: uuidPtrToPgtype(template.ServerConfigID),
			Name:           template.Name,
			Subject:        template.Subject,
			BodyHtml:       &template.BodyHTML,
			BodyText:       &template.BodyText,
			FromEmail:      &template.FromEmail,
			FromName:       &template.FromName,
			TemplateEngine: template.TemplateEngine,
			IsActive:       template.IsActive,
			UpdatedAt:      time.Now().UTC(),
		})
	}
	if err != pgx.ErrNoRows {
		return err
	}

	// Create new
	template.AppID = nil
	template.EmailTypeID = emailTypeID
	return r.CreateTemplate(template)
}

// ============================================================================
// Type conversion helpers
// ============================================================================

// uuidToPgtypeVal converts a uuid.UUID value to a pgtype.UUID (always valid).
func uuidToPgtypeVal(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// uuidPtrToPgtype converts a *uuid.UUID to a pgtype.UUID (nullable).
func uuidPtrToPgtype(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *u, Valid: true}
}

// pgtypeUUIDToPtr converts a pgtype.UUID to *uuid.UUID.
func pgtypeUUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

// strOrNil returns a *string from a string value, returning nil for empty strings
// to match the SQLC generated nullable column types.
func strOrNil(s string) *string {
	return &s
}

// derefStr returns the string value from a *string, or empty string if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// jsonOrDefault returns the JSON bytes or a default empty array if nil/empty.
func jsonOrDefault(j json.RawMessage) json.RawMessage {
	if len(j) == 0 {
		return json.RawMessage("[]")
	}
	return json.RawMessage(j)
}

// serverConfigRowToModel converts a sqlcgen.EmailServerConfig to a models.EmailServerConfig.
func serverConfigRowToModel(row sqlcgen.EmailServerConfig) *models.EmailServerConfig {
	return &models.EmailServerConfig{
		ID:           row.ID,
		AppID:        pgtypeUUIDToPtr(row.AppID),
		Name:         row.Name,
		SMTPHost:     row.SmtpHost,
		SMTPPort:     int(row.SmtpPort),
		SMTPUsername: derefStr(row.SmtpUsername),
		SMTPPassword: derefStr(row.SmtpPassword),
		FromAddress:  row.FromAddress,
		FromName:     derefStr(row.FromName),
		UseTLS:       row.UseTls,
		IsActive:     row.IsActive,
		IsDefault:    row.IsDefault,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

// emailTypeRowToModel converts a sqlcgen.EmailType to a models.EmailType.
func emailTypeRowToModel(row sqlcgen.EmailType) models.EmailType {
	return models.EmailType{
		ID:             row.ID,
		Code:           row.Code,
		Name:           row.Name,
		Description:    derefStr(row.Description),
		DefaultSubject: derefStr(row.DefaultSubject),
		Variables:      json.RawMessage(row.Variables),
		IsSystem:       row.IsSystem,
		IsActive:       row.IsActive,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

// templateRowToModel converts a sqlcgen.EmailTemplate to a models.EmailTemplate (without EmailType join).
func templateRowToModel(row sqlcgen.EmailTemplate) models.EmailTemplate {
	return models.EmailTemplate{
		ID:             row.ID,
		AppID:          pgtypeUUIDToPtr(row.AppID),
		EmailTypeID:    row.EmailTypeID,
		ServerConfigID: pgtypeUUIDToPtr(row.ServerConfigID),
		Name:           row.Name,
		Subject:        row.Subject,
		BodyHTML:       derefStr(row.BodyHtml),
		BodyText:       derefStr(row.BodyText),
		FromEmail:      derefStr(row.FromEmail),
		FromName:       derefStr(row.FromName),
		TemplateEngine: row.TemplateEngine,
		IsActive:       row.IsActive,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

// templateWithTypeRowToModel converts a joined template+email_type row to a models.EmailTemplate.
func templateWithTypeRowToModel(
	id uuid.UUID, appID pgtype.UUID, emailTypeID uuid.UUID,
	serverConfigID pgtype.UUID, name, subject string,
	bodyHTML, bodyText, fromEmail, fromName *string,
	templateEngine string, isActive bool,
	createdAt, updatedAt time.Time,
	etID uuid.UUID, etCode, etName string,
	etDescription, etDefaultSubject *string,
	etVariables []byte, etIsSystem, etIsActive bool,
	etCreatedAt, etUpdatedAt time.Time,
) models.EmailTemplate {
	return models.EmailTemplate{
		ID:             id,
		AppID:          pgtypeUUIDToPtr(appID),
		EmailTypeID:    emailTypeID,
		ServerConfigID: pgtypeUUIDToPtr(serverConfigID),
		Name:           name,
		Subject:        subject,
		BodyHTML:       derefStr(bodyHTML),
		BodyText:       derefStr(bodyText),
		FromEmail:      derefStr(fromEmail),
		FromName:       derefStr(fromName),
		TemplateEngine: templateEngine,
		IsActive:       isActive,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		EmailType: models.EmailType{
			ID:             etID,
			Code:           etCode,
			Name:           etName,
			Description:    derefStr(etDescription),
			DefaultSubject: derefStr(etDefaultSubject),
			Variables:      json.RawMessage(etVariables),
			IsSystem:       etIsSystem,
			IsActive:       etIsActive,
			CreatedAt:      etCreatedAt,
			UpdatedAt:      etUpdatedAt,
		},
	}
}
