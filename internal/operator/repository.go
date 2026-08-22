package operator

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/sqlcgen"
)

// Repository loads operator roles and grants from Postgres.
type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, queries: sqlcgen.New(pool)}
}

// RoleGrants implements GrantLookup.
func (r *Repository) RoleGrants(ctx context.Context, roleID uuid.UUID) (string, []string, error) {
	role, err := r.queries.GetOperatorRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, err
		}
		return "", nil, err
	}
	rows, err := r.queries.ListOperatorPermissionsByRoleID(ctx, roleID)
	if err != nil {
		return "", nil, err
	}
	keys := make([]string, 0, len(rows))
	for _, p := range rows {
		keys = append(keys, p.Resource+":"+p.Action)
	}
	return role.Name, keys, nil
}

// RoleByName returns a seeded or custom role row.
func (r *Repository) RoleByName(ctx context.Context, name string) (sqlcgen.OperatorRole, error) {
	return r.queries.GetOperatorRoleByName(ctx, name)
}

// GetOperatorRoleByID returns a role row by id.
func (r *Repository) GetOperatorRoleByID(ctx context.Context, id uuid.UUID) (sqlcgen.OperatorRole, error) {
	return r.queries.GetOperatorRoleByID(ctx, id)
}

// ListRoles returns every operator role.
func (r *Repository) ListRoles(ctx context.Context) ([]sqlcgen.OperatorRole, error) {
	return r.queries.ListOperatorRoles(ctx)
}

// RoleName returns the role name for id. pgx.ErrNoRows if missing.
func (r *Repository) RoleName(ctx context.Context, id uuid.UUID) (string, error) {
	role, err := r.queries.GetOperatorRoleByID(ctx, id)
	if err != nil {
		return "", err
	}
	return role.Name, nil
}

func (r *Repository) CreateCustomRole(ctx context.Context, name, description string, grants []Permission) (sqlcgen.OperatorRole, error) {
	if IsReservedSystemRoleName(name) {
		return sqlcgen.OperatorRole{}, ErrReservedSystemRoleName
	}
	if err := rejectAdminIAMGrants(grants); err != nil {
		return sqlcgen.OperatorRole{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return sqlcgen.OperatorRole{}, err
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			log.Printf("operator role create rollback: %v", rbErr)
		}
	}()
	qtx := r.queries.WithTx(tx)
	role, err := qtx.InsertOperatorRole(ctx, sqlcgen.InsertOperatorRoleParams{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return sqlcgen.OperatorRole{}, mapRoleWriteError(err)
	}
	if err := insertRoleGrants(ctx, qtx, role.ID, grants); err != nil {
		return sqlcgen.OperatorRole{}, mapRoleWriteError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlcgen.OperatorRole{}, err
	}
	return role, nil
}

func (r *Repository) UpdateCustomRole(ctx context.Context, id uuid.UUID, name, description string) (sqlcgen.OperatorRole, error) {
	if IsReservedSystemRoleName(name) {
		return sqlcgen.OperatorRole{}, ErrReservedSystemRoleName
	}
	existing, err := r.GetOperatorRoleByID(ctx, id)
	if err := rejectImmutableOperatorRole(existing, err); err != nil {
		return sqlcgen.OperatorRole{}, err
	}
	role, err := r.queries.UpdateOperatorRoleIfNotSystem(ctx, sqlcgen.UpdateOperatorRoleIfNotSystemParams{
		ID:          id,
		Name:        name,
		Description: description,
	})
	if err != nil {
		return sqlcgen.OperatorRole{}, mapRoleWriteError(err)
	}
	return role, nil
}

func (r *Repository) CountRoleAssignments(ctx context.Context, id uuid.UUID) (int64, error) {
	accounts, err := r.queries.CountAdminAccountsByOperatorRole(ctx, id)
	if err != nil {
		return 0, err
	}
	keys, err := r.queries.CountAPIKeysByOperatorRole(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return 0, err
	}
	return accounts + keys, nil
}

func (r *Repository) DeleteCustomRole(ctx context.Context, id uuid.UUID) (int64, error) {
	existing, err := r.GetOperatorRoleByID(ctx, id)
	if err := rejectImmutableOperatorRole(existing, err); err != nil {
		return 0, err
	}
	assigned, err := r.CountRoleAssignments(ctx, id)
	if err != nil {
		return 0, err
	}
	if assigned > 0 {
		return 0, ErrRoleAssigned
	}
	n, err := r.queries.DeleteOperatorRoleIfNotSystem(ctx, id)
	if err != nil {
		return 0, mapRoleWriteError(err)
	}
	return n, nil
}

func (r *Repository) ReplaceRolePermissions(ctx context.Context, roleID uuid.UUID, grants []Permission) error {
	if err := rejectAdminIAMGrants(grants); err != nil {
		return err
	}
	existing, err := r.GetOperatorRoleByID(ctx, roleID)
	if err := rejectImmutableOperatorRole(existing, err); err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			log.Printf("operator role permissions rollback: %v", rbErr)
		}
	}()
	qtx := r.queries.WithTx(tx)
	if err := qtx.DeleteOperatorRolePermissions(ctx, roleID); err != nil {
		return err
	}
	if err := insertRoleGrants(ctx, qtx, roleID, grants); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertRoleGrants(ctx context.Context, q *sqlcgen.Queries, roleID uuid.UUID, grants []Permission) error {
	for _, g := range grants {
		perm, err := q.GetOperatorPermissionByResourceAction(ctx, sqlcgen.GetOperatorPermissionByResourceActionParams{
			Resource: g.Resource,
			Action:   g.Action,
		})
		if err != nil {
			return err
		}
		if err := q.InsertOperatorRolePermission(ctx, sqlcgen.InsertOperatorRolePermissionParams{
			RoleID:       roleID,
			PermissionID: perm.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func rejectImmutableOperatorRole(role sqlcgen.OperatorRole, err error) error {
	if err != nil {
		return mapRoleWriteError(err)
	}
	if role.IsSystem {
		return ErrSystemRoleImmutable
	}
	return nil
}

func rejectAdminIAMGrants(grants []Permission) error {
	for _, g := range grants {
		if g.Resource == ResAdminIAM {
			return ErrAdminIAMOnCustomRole
		}
	}
	return nil
}

func mapRoleWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return ErrRoleNameTaken
	case "23503":
		switch pgErr.ConstraintName {
		case "operator_iam_events_old_role_id_fkey", "operator_iam_events_new_role_id_fkey":
			return ErrRoleReferenced
		default:
			return ErrRoleAssigned
		}
	default:
		return err
	}
}
