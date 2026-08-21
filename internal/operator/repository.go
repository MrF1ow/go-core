package operator

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/sqlcgen"
)

// Repository loads operator roles and grants from Postgres.
type Repository struct {
	queries *sqlcgen.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{queries: sqlcgen.New(pool)}
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

// ListRoles returns every operator role.
func (r *Repository) ListRoles(ctx context.Context) ([]sqlcgen.OperatorRole, error) {
	return r.queries.ListOperatorRoles(ctx)
}
