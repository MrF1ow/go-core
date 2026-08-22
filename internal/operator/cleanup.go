package operator

import (
	"context"
	"time"
)

const (
	EvidenceRetentionDays    = 365
	EvidenceCleanupBatchSize = 1000
)

type EvidenceCleanup struct {
	Now              func() time.Time
	Sleep            func(time.Duration)
	DeleteAccessLogs func(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
	DeleteIAMEvents  func(ctx context.Context, cutoff time.Time, limit int32) (int64, error)
}

func EvidenceCutoff(now time.Time) time.Time {
	return now
}

func (c *EvidenceCleanup) Run(ctx context.Context) error {
	return nil
}
