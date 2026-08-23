package admin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	appRedis "github.com/MrF1ow/go-core/internal/redis"
	"github.com/MrF1ow/go-core/internal/safeconv"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/pkg/models"
)

// DashboardStats holds aggregate counts for the admin dashboard.
type DashboardStats struct {
	TotalUsers         int64
	ActiveUsers        int64
	InactiveUsers      int64
	TotalTenants       int64
	TotalApps          int64
	RecentEventsCount  int64 // activity logs in last 24 hours
	ActiveSessions     int64 // active user sessions across all apps
	TrustedDeviceCount int64 // active (non-expired) trusted devices
	VerifiedPhoneCount int64 // users with phone_verified = true
}

// DashboardService provides aggregated data for the admin dashboard.
type DashboardService struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewDashboardService creates a new DashboardService.
func NewDashboardService(pool *pgxpool.Pool) *DashboardService {
	return &DashboardService{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// GetStats returns aggregate counts for the dashboard stat cards.
// A non-nil appID scopes every card to that application.
func (s *DashboardService) GetStats(appID *uuid.UUID) (*DashboardStats, error) {
	if appID != nil {
		return s.getStatsForApp(*appID)
	}
	stats := &DashboardStats{}
	ctx := context.Background()

	total, err := s.queries.AdminCountTotalUsers(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = total

	active, err := s.queries.AdminCountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	stats.ActiveUsers = active

	stats.InactiveUsers = stats.TotalUsers - stats.ActiveUsers

	tenants, err := s.queries.AdminCountTenants(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalTenants = tenants

	apps, err := s.queries.AdminCountApps(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalApps = apps

	since := time.Now().Add(-24 * time.Hour)
	recentCount, err := s.queries.AdminCountRecentActivityLogs(ctx, since)
	if err != nil {
		return nil, err
	}
	stats.RecentEventsCount = recentCount

	appIDRows, err := s.queries.AdminListAllAppIDs(ctx)
	if err != nil {
		return nil, err
	}
	for _, id := range appIDRows {
		count, err := appRedis.CountAppSessions(id)
		if err != nil {
			continue
		}
		stats.ActiveSessions += count
	}

	trustedCount, err := s.queries.AdminCountActiveTrustedDevices(ctx)
	if err != nil {
		stats.TrustedDeviceCount = 0
	} else {
		stats.TrustedDeviceCount = trustedCount
	}

	phoneCount, err := s.queries.AdminCountVerifiedPhoneUsers(ctx)
	if err != nil {
		stats.VerifiedPhoneCount = 0
	} else {
		stats.VerifiedPhoneCount = phoneCount
	}

	return stats, nil
}

func (s *DashboardService) getStatsForApp(appID uuid.UUID) (*DashboardStats, error) {
	stats := &DashboardStats{}
	ctx := context.Background()

	stats.TotalApps = 1
	app, err := s.queries.AdminGetAppByID(ctx, appID)
	if err == nil && app.TenantID != uuid.Nil {
		stats.TotalTenants = 1
	}

	total, err := s.queries.AdminCountTotalUsersByApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = total

	active, err := s.queries.AdminCountActiveUsersByApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	stats.ActiveUsers = active
	stats.InactiveUsers = stats.TotalUsers - stats.ActiveUsers

	since := time.Now().Add(-24 * time.Hour)
	recentCount, err := s.queries.AdminCountRecentActivityLogsByApp(ctx, sqlcgen.AdminCountRecentActivityLogsByAppParams{
		Timestamp: since,
		AppID:     appID,
	})
	if err != nil {
		return nil, err
	}
	stats.RecentEventsCount = recentCount

	count, sessErr := appRedis.CountAppSessions(appID.String())
	if sessErr == nil {
		stats.ActiveSessions = count
	}

	trustedCount, err := s.queries.AdminCountActiveTrustedDevicesByApp(ctx, appID)
	if err != nil {
		stats.TrustedDeviceCount = 0
	} else {
		stats.TrustedDeviceCount = trustedCount
	}

	phoneCount, err := s.queries.AdminCountVerifiedPhoneUsersByApp(ctx, appID)
	if err != nil {
		stats.VerifiedPhoneCount = 0
	} else {
		stats.VerifiedPhoneCount = phoneCount
	}

	return stats, nil
}

// GetRecentActivity returns the most recent activity log entries.
func (s *DashboardService) GetRecentActivity(limit int, appID *uuid.UUID) ([]models.ActivityLog, error) {
	ctx := context.Background()
	var rows []sqlcgen.ActivityLog
	var err error
	if appID != nil {
		rows, err = s.queries.AdminGetRecentActivityByApp(ctx, sqlcgen.AdminGetRecentActivityByAppParams{
			AppID: *appID,
			Limit: safeconv.ToInt32(limit),
		})
	} else {
		rows, err = s.queries.AdminGetRecentActivity(ctx, safeconv.ToInt32(limit))
	}
	if err != nil {
		return nil, err
	}
	logs := make([]models.ActivityLog, len(rows))
	for i, row := range rows {
		var expiresAt *time.Time
		if row.ExpiresAt.Valid {
			t := row.ExpiresAt.Time
			expiresAt = &t
		}
		logs[i] = models.ActivityLog{
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
	return logs, nil
}
