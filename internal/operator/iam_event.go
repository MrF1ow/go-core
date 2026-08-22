package operator

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/MrF1ow/go-core/internal/sqlcgen"
)

const (
	ActorKindSetupCLI      = "setup_cli"
	ActionAssign           = "assign"
	ActionCreatePrincipal  = "create_principal"
	ActionRevokeKey        = "revoke_key"
	ActionDisablePrincipal = "disable_principal"
)

// IAMEvent is one append-only operator role assignment or principal lifecycle row.
type IAMEvent struct {
	ID              uuid.UUID  `json:"id,omitempty"`
	At              time.Time  `json:"at,omitempty"`
	ActorKind       string     `json:"actor_kind"`
	ActorKeyID      *uuid.UUID `json:"actor_key_id,omitempty"`
	ActorAccountID  *uuid.UUID `json:"actor_account_id,omitempty"`
	TargetKind      Kind       `json:"target_kind"`
	TargetKeyID     *uuid.UUID `json:"target_key_id,omitempty"`
	TargetAccountID *uuid.UUID `json:"target_account_id,omitempty"`
	OldRoleID       *uuid.UUID `json:"old_role_id,omitempty"`
	NewRoleID       *uuid.UUID `json:"new_role_id,omitempty"`
	Action          string     `json:"action"`
}

func (r *Repository) InsertIAMEvent(ctx context.Context, ev IAMEvent) error {
	_, err := r.queries.InsertOperatorIAMEvent(ctx, sqlcgen.InsertOperatorIAMEventParams{
		ActorKind:       ev.ActorKind,
		ActorKeyID:      uuidPtrToPgtype(ev.ActorKeyID),
		ActorAccountID:  uuidPtrToPgtype(ev.ActorAccountID),
		TargetKind:      string(ev.TargetKind),
		TargetKeyID:     uuidPtrToPgtype(ev.TargetKeyID),
		TargetAccountID: uuidPtrToPgtype(ev.TargetAccountID),
		OldRoleID:       uuidPtrToPgtype(ev.OldRoleID),
		NewRoleID:       uuidPtrToPgtype(ev.NewRoleID),
		Action:          ev.Action,
	})
	return err
}

func (r *Repository) ListIAMEvents(ctx context.Context, limit, offset int32, targetKeyID, targetAccountID *uuid.UUID) ([]IAMEvent, error) {
	rows, err := r.queries.ListOperatorIAMEvents(ctx, sqlcgen.ListOperatorIAMEventsParams{
		Limit:           limit,
		Offset:          offset,
		TargetKeyID:     uuidPtrToPgtype(targetKeyID),
		TargetAccountID: uuidPtrToPgtype(targetAccountID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]IAMEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, IAMEvent{
			ID:              row.ID,
			At:              row.At,
			ActorKind:       row.ActorKind,
			ActorKeyID:      pgtypeUUIDToPtr(row.ActorKeyID),
			ActorAccountID:  pgtypeUUIDToPtr(row.ActorAccountID),
			TargetKind:      Kind(row.TargetKind),
			TargetKeyID:     pgtypeUUIDToPtr(row.TargetKeyID),
			TargetAccountID: pgtypeUUIDToPtr(row.TargetAccountID),
			OldRoleID:       pgtypeUUIDToPtr(row.OldRoleID),
			NewRoleID:       pgtypeUUIDToPtr(row.NewRoleID),
			Action:          row.Action,
		})
	}
	return out, nil
}
