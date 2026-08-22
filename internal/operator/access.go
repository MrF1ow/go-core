package operator

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/MrF1ow/go-core/internal/safeconv"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
)

const (
	DecisionAllow = "allow"
	DecisionDeny  = "deny"
)

// AccessRecord is one operator permission decision.
// Insert, list JSON, and middleware share this type.
type AccessRecord struct {
	ID        uuid.UUID  `json:"id,omitempty"`
	At        time.Time  `json:"at,omitempty"`
	Kind      Kind       `json:"kind"`
	KeyID     *uuid.UUID `json:"key_id,omitempty"`
	AccountID *uuid.UUID `json:"account_id,omitempty"`
	RoleName  string     `json:"role_name"`
	Method    string     `json:"method"`
	Path      string     `json:"path"`
	Decision  string     `json:"decision"`
	Resource  string     `json:"resource"`
	Action    string     `json:"action"`
	Status    int        `json:"status"`
}

// ShouldLogAccess is the v1 insert policy.
// Deny always. Write allow always. Env-key allow always. Ordinary read allows skip.
func ShouldLogAccess(rec AccessRecord) bool {
	switch rec.Decision {
	case DecisionDeny:
		return true
	case DecisionAllow:
		return rec.Action == ActionWrite || rec.Kind == KindEnvKey
	default:
		return false
	}
}

func (r *Repository) InsertAccessLog(ctx context.Context, rec AccessRecord) error {
	_, err := r.queries.InsertOperatorAccessLog(ctx, sqlcgen.InsertOperatorAccessLogParams{
		Kind:      string(rec.Kind),
		KeyID:     uuidPtrToPgtype(rec.KeyID),
		AccountID: uuidPtrToPgtype(rec.AccountID),
		RoleName:  rec.RoleName,
		Method:    rec.Method,
		Path:      rec.Path,
		Decision:  rec.Decision,
		Resource:  rec.Resource,
		Action:    rec.Action,
		Status:    safeconv.ToInt32(rec.Status),
	})
	return err
}

func (r *Repository) ListAccessLogs(ctx context.Context, limit int32, offset int32, decision *string) ([]AccessRecord, error) {
	rows, err := r.queries.ListOperatorAccessLogs(ctx, sqlcgen.ListOperatorAccessLogsParams{
		Limit:    limit,
		Offset:   offset,
		Decision: decision,
	})
	if err != nil {
		return nil, err
	}
	out := make([]AccessRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, AccessRecord{
			ID:        row.ID,
			At:        row.At,
			Kind:      Kind(row.Kind),
			KeyID:     pgtypeUUIDToPtr(row.KeyID),
			AccountID: pgtypeUUIDToPtr(row.AccountID),
			RoleName:  row.RoleName,
			Method:    row.Method,
			Path:      row.Path,
			Decision:  row.Decision,
			Resource:  row.Resource,
			Action:    row.Action,
			Status:    int(row.Status),
		})
	}
	return out, nil
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
