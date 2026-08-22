package operator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/safeconv"
)

const (
	EvidenceRetentionDays      = 365
	EvidenceCleanupBatchSize   = 1000
	evidenceCleanupInterval    = 24 * time.Hour
	evidenceCleanupBatchPause  = 100 * time.Millisecond
	evidenceCleanupInitialWait = 30 * time.Second

	deleteAccessLogsSQL = `DELETE FROM operator_access_logs WHERE id IN (
		SELECT id FROM operator_access_logs WHERE at < $1 LIMIT $2
	)`
	deleteIAMEventsSQL = `DELETE FROM operator_iam_events WHERE id IN (
		SELECT id FROM operator_iam_events WHERE at < $1 LIMIT $2
	)`
)

type EvidenceCleanup struct {
	Now              func() time.Time
	Sleep            func(time.Duration)
	DeleteAccessLogs func(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
	DeleteIAMEvents  func(ctx context.Context, cutoff time.Time, limit int32) (int64, error)

	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

func NewEvidenceCleanup(pool *pgxpool.Pool) *EvidenceCleanup {
	return &EvidenceCleanup{
		Now:   time.Now,
		Sleep: time.Sleep,
		DeleteAccessLogs: func(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
			return execEvidenceDelete(ctx, pool, deleteAccessLogsSQL, cutoff, limit)
		},
		DeleteIAMEvents: func(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
			return execEvidenceDelete(ctx, pool, deleteIAMEventsSQL, cutoff, limit)
		},
	}
}

func execEvidenceDelete(ctx context.Context, pool *pgxpool.Pool, sql string, cutoff time.Time, limit int32) (int64, error) {
	tag, err := pool.Exec(ctx, sql, cutoff, limit)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func EvidenceCutoff(now time.Time) time.Time {
	return now.UTC().AddDate(0, 0, -EvidenceRetentionDays)
}

func (c *EvidenceCleanup) Run(ctx context.Context) error {
	nowFn := c.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	sleepFn := c.Sleep
	if sleepFn == nil {
		sleepFn = time.Sleep
	}
	cutoff := EvidenceCutoff(nowFn())
	var first error
	if err := c.deleteAll(ctx, cutoff, sleepFn, c.DeleteAccessLogs); err != nil {
		log.Printf("operator access log cleanup: %v", err)
		first = err
	}
	if err := c.deleteAll(ctx, cutoff, sleepFn, c.DeleteIAMEvents); err != nil {
		log.Printf("operator IAM event cleanup: %v", err)
		if first == nil {
			first = err
		}
	}
	return first
}

func (c *EvidenceCleanup) deleteAll(ctx context.Context, cutoff time.Time, sleepFn func(time.Duration), del func(context.Context, time.Time, int32) (int64, error)) error {
	if del == nil {
		return nil
	}
	limit := safeconv.ToInt32(EvidenceCleanupBatchSize)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := del(ctx, cutoff, limit)
		if err != nil {
			return fmt.Errorf("operator evidence delete: %w", err)
		}
		if n < int64(limit) {
			return nil
		}
		sleepFn(evidenceCleanupBatchPause)
	}
}

func (c *EvidenceCleanup) Start() {
	if c == nil {
		return
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.ticker = time.NewTicker(evidenceCleanupInterval)
	go c.worker()
}

func (c *EvidenceCleanup) worker() {
	timer := time.NewTimer(evidenceCleanupInitialWait)
	defer timer.Stop()
	select {
	case <-c.ctx.Done():
		return
	case <-timer.C:
		if err := c.Run(c.ctx); err != nil {
			log.Printf("operator evidence cleanup: %v", err)
		}
	}
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.ticker.C:
			if err := c.Run(c.ctx); err != nil {
				log.Printf("operator evidence cleanup: %v", err)
			}
		}
	}
}

func (c *EvidenceCleanup) Shutdown() {
	if c == nil || c.cancel == nil {
		return
	}
	c.cancel()
	if c.ticker != nil {
		c.ticker.Stop()
	}
}
