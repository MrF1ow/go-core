package admin

import (
	"context"
	"encoding/json"
	"time"

	appRedis "github.com/JedidiahDigital/go-core/internal/redis"
	"github.com/JedidiahDigital/go-core/internal/safeconv"
	"github.com/JedidiahDigital/go-core/internal/sqlcgen"
	"github.com/JedidiahDigital/go-core/pkg/models"
	"github.com/jackc/pgx/v5/pgxpool"
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
func (s *DashboardService) GetStats() (*DashboardStats, error) {
	stats := &DashboardStats{}
	ctx := context.Background()

	// Count total users
	total, err := s.queries.AdminCountTotalUsers(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = total

	// Count active users
	active, err := s.queries.AdminCountActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	stats.ActiveUsers = active

	// Count inactive users
	stats.InactiveUsers = stats.TotalUsers - stats.ActiveUsers

	// Count total tenants
	tenants, err := s.queries.AdminCountTenants(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalTenants = tenants

	// Count total applications
	apps, err := s.queries.AdminCountApps(ctx)
	if err != nil {
		return nil, err
	}
	stats.TotalApps = apps

	// Count activity logs in the last 24 hours
	since := time.Now().Add(-24 * time.Hour)
	recentCount, err := s.queries.AdminCountRecentActivityLogs(ctx, since)
	if err != nil {
		return nil, err
	}
	stats.RecentEventsCount = recentCount

	// Count active sessions across all apps (from Redis)
	appIDRows, err := s.queries.AdminListAllAppIDs(ctx)
	if err != nil {
		return nil, err
	}
	for _, appID := range appIDRows {
		count, err := appRedis.CountAppSessions(appID)
		if err != nil {
			continue // Don't fail dashboard if Redis is unavailable
		}
		stats.ActiveSessions += count
	}

	// Count active (non-expired) trusted devices
	trustedCount, err := s.queries.AdminCountActiveTrustedDevices(ctx)
	if err != nil {
		// Non-fatal: table may not exist yet on first startup
		stats.TrustedDeviceCount = 0
	} else {
		stats.TrustedDeviceCount = trustedCount
	}

	// Count users with verified phone numbers
	phoneCount, err := s.queries.AdminCountVerifiedPhoneUsers(ctx)
	if err != nil {
		stats.VerifiedPhoneCount = 0
	} else {
		stats.VerifiedPhoneCount = phoneCount
	}

	return stats, nil
}

// GetRecentActivity returns the most recent activity log entries.
func (s *DashboardService) GetRecentActivity(limit int) ([]models.ActivityLog, error) {
	rows, err := s.queries.AdminGetRecentActivity(context.Background(), safeconv.ToInt32(limit))
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
