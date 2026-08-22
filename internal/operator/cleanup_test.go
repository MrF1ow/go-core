package operator

import (
	"context"
	"testing"
	"time"
)

type memEvidenceRow struct {
	id int
	at time.Time
}

type memEvidence struct {
	access []memEvidenceRow
	events []memEvidenceRow
}

func (m *memEvidence) deleteBefore(rows *[]memEvidenceRow, cutoff time.Time, limit int32) (int64, error) {
	kept := make([]memEvidenceRow, 0, len(*rows))
	deleted := int64(0)
	for _, row := range *rows {
		if deleted < int64(limit) && row.at.Before(cutoff) {
			deleted++
			continue
		}
		kept = append(kept, row)
	}
	*rows = kept
	return deleted, nil
}

func TestEvidenceCutoff_Is365Days(t *testing.T) {
	now := time.Date(2026, 8, 22, 15, 4, 5, 0, time.UTC)
	got := EvidenceCutoff(now)
	want := time.Date(2025, 8, 22, 15, 4, 5, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("cutoff = %s, want %s", got, want)
	}
}

func TestEvidenceCleanup_DeletesRowsOlderThan365Days(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -366)
	fresh := now
	store := &memEvidence{
		access: []memEvidenceRow{{id: 1, at: old}, {id: 2, at: fresh}},
		events: []memEvidenceRow{{id: 3, at: old}, {id: 4, at: fresh}},
	}
	cleanup := &EvidenceCleanup{
		Now:   func() time.Time { return now },
		Sleep: func(time.Duration) {},
		DeleteAccessLogs: func(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
			return store.deleteBefore(&store.access, cutoff, limit)
		},
		DeleteIAMEvents: func(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
			return store.deleteBefore(&store.events, cutoff, limit)
		},
	}
	if err := cleanup.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.access) != 1 || store.access[0].id != 2 {
		t.Fatalf("access = %#v", store.access)
	}
	if len(store.events) != 1 || store.events[0].id != 4 {
		t.Fatalf("events = %#v", store.events)
	}
	if err := cleanup.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.access) != 1 || store.access[0].id != 2 {
		t.Fatalf("second run access = %#v", store.access)
	}
	if len(store.events) != 1 || store.events[0].id != 4 {
		t.Fatalf("second run events = %#v", store.events)
	}
}

func TestEvidenceCleanup_BatchesUntilUnderLimit(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -366)
	store := &memEvidence{}
	for i := 0; i < EvidenceCleanupBatchSize+1; i++ {
		store.access = append(store.access, memEvidenceRow{id: i + 1, at: old})
	}
	calls := 0
	cleanup := &EvidenceCleanup{
		Now:   func() time.Time { return now },
		Sleep: func(time.Duration) {},
		DeleteAccessLogs: func(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
			calls++
			return store.deleteBefore(&store.access, cutoff, limit)
		},
		DeleteIAMEvents: func(ctx context.Context, cutoff time.Time, limit int32) (int64, error) {
			return 0, nil
		},
	}
	if err := cleanup.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls < 2 {
		t.Fatalf("batch calls = %d, want at least 2", calls)
	}
	if len(store.access) != 0 {
		t.Fatalf("remaining = %d", len(store.access))
	}
}
