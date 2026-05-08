package admin

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/pkg/models"
)

// SettingsRepository handles CRUD operations for the system_settings table.
type SettingsRepository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewSettingsRepository creates a new SettingsRepository backed by pgx/SQLC.
func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// GetAllSettings returns all system settings from the database.
func (r *SettingsRepository) GetAllSettings() ([]models.SystemSetting, error) {
	rows, err := r.queries.GetAllSettings(context.Background())
	if err != nil {
		return nil, err
	}
	return toModelSettings(rows), nil
}

// GetSettingsByCategory returns all settings for a given category.
func (r *SettingsRepository) GetSettingsByCategory(category string) ([]models.SystemSetting, error) {
	rows, err := r.queries.GetSettingsByCategory(context.Background(), category)
	if err != nil {
		return nil, err
	}
	return toModelSettings(rows), nil
}

// GetSettingByKey returns a single setting by its key.
// Returns nil, nil if the key is not found in the database.
func (r *SettingsRepository) GetSettingByKey(key string) (*models.SystemSetting, error) {
	row, err := r.queries.GetSettingByKey(context.Background(), key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s := toModelSetting(row)
	return &s, nil
}

// UpsertSetting inserts or updates a setting by key.
// Uses PostgreSQL ON CONFLICT DO UPDATE (upsert).
func (r *SettingsRepository) UpsertSetting(key, value, category string) error {
	return r.queries.UpsertSetting(context.Background(), sqlcgen.UpsertSettingParams{
		Key:      key,
		Value:    value,
		Category: category,
	})
}

// DeleteSetting removes a setting from the database (reverts to default behavior).
func (r *SettingsRepository) DeleteSetting(key string) error {
	return r.queries.DeleteSetting(context.Background(), key)
}

// toModelSettings converts a slice of SQLC-generated rows to the shared model type.
func toModelSettings(rows []sqlcgen.SystemSetting) []models.SystemSetting {
	out := make([]models.SystemSetting, len(rows))
	for i, row := range rows {
		out[i] = toModelSetting(row)
	}
	return out
}

// toModelSetting converts a single SQLC-generated row to the shared model type.
func toModelSetting(row sqlcgen.SystemSetting) models.SystemSetting {
	return models.SystemSetting{
		Key:       row.Key,
		Value:     row.Value,
		Category:  row.Category,
		UpdatedAt: row.UpdatedAt,
	}
}
