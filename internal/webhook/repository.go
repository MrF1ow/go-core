package webhook

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/safeconv"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/google/uuid"
)

// Repository handles all database operations for webhook endpoints and deliveries.
type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewRepository creates a new webhook repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// ============================================================================
// Endpoint operations
// ============================================================================

// CreateEndpoint persists a new webhook endpoint.
func (r *Repository) CreateEndpoint(ep *models.WebhookEndpoint) error {
	if ep.ID == uuid.Nil {
		ep.ID = uuid.New()
	}
	now := time.Now().UTC()
	if ep.CreatedAt.IsZero() {
		ep.CreatedAt = now
	}
	if ep.UpdatedAt.IsZero() {
		ep.UpdatedAt = now
	}
	return r.queries.CreateWebhookEndpoint(context.Background(), sqlcgen.CreateWebhookEndpointParams{
		ID:        ep.ID,
		AppID:     ep.AppID,
		EventType: ep.EventType,
		Url:       ep.URL,
		Secret:    ep.Secret,
		IsActive:  ep.IsActive,
		CreatedAt: ep.CreatedAt,
		UpdatedAt: ep.UpdatedAt,
	})
}

// GetEndpointByID returns a webhook endpoint by its primary key.
// Returns nil, nil when not found.
func (r *Repository) GetEndpointByID(id uuid.UUID) (*models.WebhookEndpoint, error) {
	row, err := r.queries.GetWebhookEndpointByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ep := toModelEndpoint(row)
	return &ep, nil
}

// GetEndpointByAppAndEvent returns the endpoint for a specific (appID, eventType) pair.
// Returns nil, nil when not found.
func (r *Repository) GetEndpointByAppAndEvent(appID uuid.UUID, eventType string) (*models.WebhookEndpoint, error) {
	row, err := r.queries.GetWebhookEndpointByAppAndEvent(context.Background(), sqlcgen.GetWebhookEndpointByAppAndEventParams{
		AppID:     appID,
		EventType: eventType,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ep := toModelEndpoint(row)
	return &ep, nil
}

// ListEndpointsByApp returns all (non-deleted) webhook endpoints for an application.
func (r *Repository) ListEndpointsByApp(appID uuid.UUID, page, pageSize int) ([]models.WebhookEndpoint, int64, error) {
	total, err := r.queries.CountWebhookEndpointsByApp(context.Background(), appID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.ListWebhookEndpointsByApp(context.Background(), sqlcgen.ListWebhookEndpointsByAppParams{
		AppID:  appID,
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	return toModelEndpoints(rows), total, nil
}

// ListAllEndpoints returns all non-deleted webhook endpoints (admin use).
func (r *Repository) ListAllEndpoints(page, pageSize int) ([]models.WebhookEndpoint, int64, error) {
	total, err := r.queries.CountAllWebhookEndpoints(context.Background())
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.ListAllWebhookEndpoints(context.Background(), sqlcgen.ListAllWebhookEndpointsParams{
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	return toModelEndpoints(rows), total, nil
}

// GetActiveEndpointsForEvent returns all active (non-deleted, is_active=true) endpoints
// for a given appID and eventType. Used by the dispatch path.
func (r *Repository) GetActiveEndpointsForEvent(appID uuid.UUID, eventType string) ([]models.WebhookEndpoint, error) {
	rows, err := r.queries.GetActiveWebhookEndpointsForEvent(context.Background(), sqlcgen.GetActiveWebhookEndpointsForEventParams{
		AppID:     appID,
		EventType: eventType,
	})
	if err != nil {
		return nil, err
	}
	return toModelEndpoints(rows), nil
}

// UpdateEndpointActive sets the is_active flag on an endpoint.
func (r *Repository) UpdateEndpointActive(id uuid.UUID, isActive bool) error {
	return r.queries.UpdateWebhookEndpointActive(context.Background(), sqlcgen.UpdateWebhookEndpointActiveParams{
		ID:       id,
		IsActive: isActive,
	})
}

// SoftDeleteEndpoint sets deleted_at on an endpoint.
func (r *Repository) SoftDeleteEndpoint(id uuid.UUID) error {
	return r.queries.SoftDeleteWebhookEndpoint(context.Background(), id)
}

// ============================================================================
// Delivery operations
// ============================================================================

// CreateDelivery persists a delivery record.
func (r *Repository) CreateDelivery(d *models.WebhookDelivery) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	return r.queries.CreateWebhookDelivery(context.Background(), sqlcgen.CreateWebhookDeliveryParams{
		ID:           d.ID,
		EndpointID:   d.EndpointID,
		AppID:        d.AppID,
		EventType:    d.EventType,
		Payload:      d.Payload,
		Attempt:      safeconv.ToInt32(d.Attempt),
		StatusCode:   intToInt32Ptr(d.StatusCode),
		ResponseBody: strPtr(d.ResponseBody),
		LatencyMs:    int64Ptr(d.LatencyMs),
		Success:      d.Success,
		ErrorMessage: strPtr(d.ErrorMessage),
		NextRetryAt:  timeToPgTimestamptz(d.NextRetryAt),
		CreatedAt:    d.CreatedAt,
	})
}

// GetDeliveriesByEndpoint returns delivery history for a specific endpoint, paginated.
func (r *Repository) GetDeliveriesByEndpoint(endpointID uuid.UUID, page, pageSize int) ([]models.WebhookDelivery, int64, error) {
	total, err := r.queries.CountWebhookDeliveriesByEndpoint(context.Background(), endpointID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.ListWebhookDeliveriesByEndpoint(context.Background(), sqlcgen.ListWebhookDeliveriesByEndpointParams{
		EndpointID: endpointID,
		Limit:      safeconv.ToInt32(pageSize),
		Offset:     safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	return toModelDeliveries(rows), total, nil
}

// GetDeliveriesByApp returns delivery history across all endpoints for an app, paginated.
func (r *Repository) GetDeliveriesByApp(appID uuid.UUID, page, pageSize int) ([]models.WebhookDelivery, int64, error) {
	total, err := r.queries.CountWebhookDeliveriesByApp(context.Background(), appID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.ListWebhookDeliveriesByApp(context.Background(), sqlcgen.ListWebhookDeliveriesByAppParams{
		AppID:  appID,
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}
	return toModelDeliveries(rows), total, nil
}

// GetPendingRetries returns all delivery records where next_retry_at <= now and success = false.
// Used by the background retry worker.
func (r *Repository) GetPendingRetries(now time.Time, limit int) ([]models.WebhookDelivery, error) {
	rows, err := r.queries.GetPendingWebhookRetries(context.Background(), sqlcgen.GetPendingWebhookRetriesParams{
		NextRetryAt: pgtype.Timestamptz{Time: now, Valid: true},
		Limit:       safeconv.ToInt32(limit),
	})
	if err != nil {
		return nil, err
	}
	return toModelDeliveries(rows), nil
}

// ClearRetrySchedule nullifies next_retry_at (used after all retries exhausted or on success).
func (r *Repository) ClearRetrySchedule(id uuid.UUID) error {
	return r.queries.ClearWebhookRetrySchedule(context.Background(), id)
}

// ============================================================================
// Type conversion helpers
// ============================================================================

// toModelEndpoints converts a slice of SQLC-generated rows to the shared model type.
func toModelEndpoints(rows []sqlcgen.WebhookEndpoint) []models.WebhookEndpoint {
	out := make([]models.WebhookEndpoint, len(rows))
	for i, row := range rows {
		out[i] = toModelEndpoint(row)
	}
	return out
}

// toModelEndpoint converts a single SQLC-generated row to the shared model type.
func toModelEndpoint(row sqlcgen.WebhookEndpoint) models.WebhookEndpoint {
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		deletedAt = &row.DeletedAt.Time
	}
	return models.WebhookEndpoint{
		ID:        row.ID,
		AppID:     row.AppID,
		EventType: row.EventType,
		URL:       row.Url,
		Secret:    row.Secret,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: deletedAt,
	}
}

// toModelDeliveries converts a slice of SQLC-generated rows to the shared model type.
func toModelDeliveries(rows []sqlcgen.WebhookDelivery) []models.WebhookDelivery {
	out := make([]models.WebhookDelivery, len(rows))
	for i, row := range rows {
		out[i] = toModelDelivery(row)
	}
	return out
}

// toModelDelivery converts a single SQLC-generated row to the shared model type.
func toModelDelivery(row sqlcgen.WebhookDelivery) models.WebhookDelivery {
	var nextRetryAt *time.Time
	if row.NextRetryAt.Valid {
		nextRetryAt = &row.NextRetryAt.Time
	}
	return models.WebhookDelivery{
		ID:           row.ID,
		EndpointID:   row.EndpointID,
		AppID:        row.AppID,
		EventType:    row.EventType,
		Payload:      row.Payload,
		Attempt:      int(row.Attempt),
		StatusCode:   derefInt32(row.StatusCode),
		ResponseBody: derefStr(row.ResponseBody),
		LatencyMs:    derefInt64(row.LatencyMs),
		Success:      row.Success,
		ErrorMessage: derefStr(row.ErrorMessage),
		NextRetryAt:  nextRetryAt,
		CreatedAt:    row.CreatedAt,
	}
}

// timeToPgTimestamptz converts a *time.Time to pgtype.Timestamptz.
func timeToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
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

// intToInt32Ptr converts an int to *int32, returning nil for zero values.
func intToInt32Ptr(v int) *int32 {
	if v == 0 {
		return nil
	}
	i := safeconv.ToInt32(v)
	return &i
}

// derefInt32 safely dereferences a *int32, returning 0 if nil.
func derefInt32(v *int32) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

// int64Ptr converts an int64 to *int64, returning nil for zero values.
func int64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

// derefInt64 safely dereferences a *int64, returning 0 if nil.
func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
