package operator

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MrF1ow/go-core/internal/sqlcgen"
)

func TestRejectImmutableOperatorRole_System(t *testing.T) {
	err := rejectImmutableOperatorRole(sqlcgen.OperatorRole{IsSystem: true}, nil)
	if !errors.Is(err, ErrSystemRoleImmutable) {
		t.Fatalf("err = %v, want ErrSystemRoleImmutable", err)
	}
}

func TestRejectImmutableOperatorRole_Custom(t *testing.T) {
	if err := rejectImmutableOperatorRole(sqlcgen.OperatorRole{IsSystem: false}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRejectImmutableOperatorRole_Missing(t *testing.T) {
	err := rejectImmutableOperatorRole(sqlcgen.OperatorRole{}, pgx.ErrNoRows)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("err = %v, want pgx.ErrNoRows", err)
	}
}

func TestRejectAdminIAMGrants(t *testing.T) {
	if err := rejectAdminIAMGrants(nil); err != nil {
		t.Fatal(err)
	}
	err := rejectAdminIAMGrants([]Permission{{Resource: ResUsers, Action: ActionRead}, {Resource: ResAdminIAM, Action: ActionWrite}})
	if !errors.Is(err, ErrAdminIAMOnCustomRole) {
		t.Fatalf("err = %v, want ErrAdminIAMOnCustomRole", err)
	}
}

func TestMapRoleWriteError_IAMEventFK(t *testing.T) {
	err := mapRoleWriteError(&pgconn.PgError{Code: "23503", ConstraintName: "operator_iam_events_new_role_id_fkey"})
	if !errors.Is(err, ErrRoleReferenced) {
		t.Fatalf("err = %v, want ErrRoleReferenced", err)
	}
	err = mapRoleWriteError(&pgconn.PgError{Code: "23503", ConstraintName: "operator_iam_events_old_role_id_fkey"})
	if !errors.Is(err, ErrRoleReferenced) {
		t.Fatalf("err = %v, want ErrRoleReferenced", err)
	}
}

func TestMapRoleWriteError_AssignedPrincipalFK(t *testing.T) {
	err := mapRoleWriteError(&pgconn.PgError{Code: "23503", ConstraintName: "admin_accounts_operator_role_id_fkey"})
	if !errors.Is(err, ErrRoleAssigned) {
		t.Fatalf("err = %v, want ErrRoleAssigned", err)
	}
}
