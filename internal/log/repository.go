package log

import (
	"context"
	"encoding/json"
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

// toFilterParams converts the common filter parameters (eventType string, startDate/endDate *time.Time)
// into the nullable SQLC types. Empty eventType becomes nil; nil times become invalid pgtype.Timestamptz.
// For endDate, one day is added to include the entire end date (matching the original GORM behaviour).
func toFilterParams(eventType string, startDate, endDate *time.Time) (*string, pgtype.Timestamptz, pgtype.Timestamptz) {
	var et *string
	if eventType != "" {
		et = &eventType
	}

	var sd pgtype.Timestamptz
	if startDate != nil {
		sd = pgtype.Timestamptz{Time: *startDate, Valid: true}
	}

	var ed pgtype.Timestamptz
	if endDate != nil {
		endOfDay := endDate.Add(24 * time.Hour)
		ed = pgtype.Timestamptz{Time: endOfDay, Valid: true}
	}

	return et, sd, ed
}

// toModelLog converts a SQLC-generated ActivityLog to the shared models.ActivityLog.
func toModelLog(row sqlcgen.ActivityLog) models.ActivityLog {
	var expiresAt *time.Time
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		expiresAt = &t
	}

	return models.ActivityLog{
		ID:        row.ID,
		AppID:     row.AppID,
		UserID:    row.UserID,
		EventType: row.EventType,
		Timestamp: row.Timestamp,
		IPAddress: row.IpAddress,
		UserAgent: row.UserAgent,
		Details:   json.RawMessage(row.Details),
		Severity:  row.Severity,
		ExpiresAt: expiresAt,
		IsAnomaly: row.IsAnomaly,
	}
}

// toModelLogs converts a slice of SQLC-generated ActivityLog rows.
func toModelLogs(rows []sqlcgen.ActivityLog) []models.ActivityLog {
	out := make([]models.ActivityLog, len(rows))
	for i, r := range rows {
		out[i] = toModelLog(r)
	}
	return out
}

// ListUserActivityLogs retrieves activity logs for a specific user with pagination and filtering.
func (r *Repository) ListUserActivityLogs(userID uuid.UUID, page, limit int, eventType string, startDate, endDate *time.Time) ([]models.ActivityLog, int64, error) {
	ctx := context.Background()
	et, sd, ed := toFilterParams(eventType, startDate, endDate)

	totalCount, err := r.queries.CountUserActivityLogs(ctx, sqlcgen.CountUserActivityLogsParams{
		UserID:    userID,
		EventType: et,
		StartDate: sd,
		EndDate:   ed,
	})
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.queries.ListUserActivityLogs(ctx, sqlcgen.ListUserActivityLogsParams{
		UserID:    userID,
		EventType: et,
		StartDate: sd,
		EndDate:   ed,
		OffsetVal: safeconv.ToInt32(offset),
		LimitVal:  safeconv.ToInt32(limit),
	})
	if err != nil {
		return nil, 0, err
	}

	return toModelLogs(rows), totalCount, nil
}

// ListAllActivityLogs retrieves activity logs for all users (admin functionality) with pagination and filtering.
func toAppID(appID *uuid.UUID) pgtype.UUID {
	if appID == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *appID, Valid: true}
}

func (r *Repository) ListAllActivityLogs(page, limit int, eventType string, startDate, endDate *time.Time, appID *uuid.UUID) ([]models.ActivityLog, int64, error) {
	ctx := context.Background()
	et, sd, ed := toFilterParams(eventType, startDate, endDate)
	filter := toAppID(appID)

	totalCount, err := r.queries.CountAllActivityLogs(ctx, sqlcgen.CountAllActivityLogsParams{
		EventType: et,
		StartDate: sd,
		EndDate:   ed,
		AppID:     filter,
	})
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.queries.ListAllActivityLogs(ctx, sqlcgen.ListAllActivityLogsParams{
		EventType: et,
		StartDate: sd,
		EndDate:   ed,
		AppID:     filter,
		OffsetVal: safeconv.ToInt32(offset),
		LimitVal:  safeconv.ToInt32(limit),
	})
	if err != nil {
		return nil, 0, err
	}

	return toModelLogs(rows), totalCount, nil
}

// GetActivityLogByID retrieves a specific activity log by ID.
func (r *Repository) GetActivityLogByID(id uuid.UUID) (*models.ActivityLog, error) {
	row, err := r.queries.GetActivityLogByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	m := toModelLog(row)
	return &m, nil
}

// ExportUserActivityLogs retrieves activity logs for a specific user without pagination, capped at limit rows.
func (r *Repository) ExportUserActivityLogs(userID uuid.UUID, limit int, eventType string, startDate, endDate *time.Time) ([]models.ActivityLog, error) {
	et, sd, ed := toFilterParams(eventType, startDate, endDate)

	rows, err := r.queries.ExportUserActivityLogs(context.Background(), sqlcgen.ExportUserActivityLogsParams{
		UserID:    userID,
		EventType: et,
		StartDate: sd,
		EndDate:   ed,
		LimitVal:  safeconv.ToInt32(limit),
	})
	if err != nil {
		return nil, err
	}

	return toModelLogs(rows), nil
}

// ExportAllActivityLogs retrieves activity logs for all users without pagination, capped at limit rows.
func (r *Repository) ExportAllActivityLogs(limit int, eventType string, startDate, endDate *time.Time, appID *uuid.UUID) ([]models.ActivityLog, error) {
	et, sd, ed := toFilterParams(eventType, startDate, endDate)

	rows, err := r.queries.ExportAllActivityLogs(context.Background(), sqlcgen.ExportAllActivityLogsParams{
		EventType: et,
		StartDate: sd,
		EndDate:   ed,
		AppID:     toAppID(appID),
		LimitVal:  safeconv.ToInt32(limit),
	})
	if err != nil {
		return nil, err
	}

	return toModelLogs(rows), nil
}
