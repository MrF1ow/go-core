package rbac

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MrF1ow/go-core/internal/safeconv"
	"github.com/MrF1ow/go-core/internal/sqlcgen"
	"github.com/MrF1ow/go-core/pkg/models"
	"github.com/google/uuid"
)

// Repository handles RBAC data access operations.
type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlcgen.Queries
}

// NewRepository creates a new RBAC repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		queries: sqlcgen.New(pool),
	}
}

// ============================================================
// Role Operations
// ============================================================

// GetRolesByAppID returns all roles for an application.
func (r *Repository) GetRolesByAppID(appID string) ([]models.Role, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	// Get roles
	rows, err := r.queries.GetRolesByAppID(context.Background(), parsedAppID)
	if err != nil {
		return nil, err
	}

	roles := make([]models.Role, len(rows))
	roleIDs := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		roles[i] = toModelRole(row)
		roleIDs[i] = row.ID
	}

	// Load permissions for each role
	for i, id := range roleIDs {
		perms, err := r.queries.GetPermissionsByRoleID(context.Background(), id)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = toModelPermissions(perms)
	}

	return roles, nil
}

// GetRoleByID returns a single role with its permissions.
func (r *Repository) GetRoleByID(id string) (*models.Role, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	row, err := r.queries.GetRoleByID(context.Background(), parsedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}

	role := toModelRole(row)

	// Load permissions
	perms, err := r.queries.GetPermissionsByRoleID(context.Background(), parsedID)
	if err != nil {
		return nil, err
	}
	role.Permissions = toModelPermissions(perms)

	return &role, nil
}

// GetRoleByName returns a role by app ID and name.
func (r *Repository) GetRoleByName(appID, name string) (*models.Role, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	row, err := r.queries.GetRoleByName(context.Background(), sqlcgen.GetRoleByNameParams{
		AppID: parsedAppID,
		Name:  name,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("role not found")
		}
		return nil, err
	}

	role := toModelRole(row)
	return &role, nil
}

// CreateRole creates a new role.
func (r *Repository) CreateRole(role *models.Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	now := time.Now().UTC()
	if role.CreatedAt.IsZero() {
		role.CreatedAt = now
	}
	if role.UpdatedAt.IsZero() {
		role.UpdatedAt = now
	}

	return r.queries.CreateRole(context.Background(), sqlcgen.CreateRoleParams{
		ID:          role.ID,
		AppID:       role.AppID,
		Name:        role.Name,
		Description: strPtr(role.Description),
		IsSystem:    role.IsSystem,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	})
}

// UpdateRole updates a role's name and description.
func (r *Repository) UpdateRole(id string, name, description string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	return r.queries.UpdateRole(context.Background(), sqlcgen.UpdateRoleParams{
		ID:          parsedID,
		Name:        name,
		Description: strPtr(description),
	})
}

// DeleteRole deletes a role by ID. Returns error if role is a system role.
func (r *Repository) DeleteRole(id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	row, err := r.queries.GetRoleByID(context.Background(), parsedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("role not found")
		}
		return err
	}
	if row.IsSystem {
		return fmt.Errorf("system roles cannot be deleted")
	}

	// FK cascades handle role_permissions and user_roles cleanup
	return r.queries.DeleteRoleByID(context.Background(), parsedID)
}

// ============================================================
// Permission Operations
// ============================================================

// GetAllPermissions returns all permissions ordered by resource and action.
func (r *Repository) GetAllPermissions() ([]models.Permission, error) {
	rows, err := r.queries.GetAllPermissions(context.Background())
	if err != nil {
		return nil, err
	}
	return toModelPermissions(rows), nil
}

// GetPermissionByID returns a single permission by ID.
func (r *Repository) GetPermissionByID(id string) (*models.Permission, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid permission ID: %w", err)
	}

	row, err := r.queries.GetPermissionByID(context.Background(), parsedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, err
	}

	perm := toModelPermission(row)
	return &perm, nil
}

// CreatePermission creates a new permission.
func (r *Repository) CreatePermission(perm *models.Permission) error {
	if perm.ID == uuid.Nil {
		perm.ID = uuid.New()
	}
	if perm.CreatedAt.IsZero() {
		perm.CreatedAt = time.Now().UTC()
	}

	return r.queries.CreatePermission(context.Background(), sqlcgen.CreatePermissionParams{
		ID:          perm.ID,
		Resource:    perm.Resource,
		Action:      perm.Action,
		Description: strPtr(perm.Description),
		CreatedAt:   perm.CreatedAt,
	})
}

// GetPermissionsByRoleID returns all permissions for a specific role.
func (r *Repository) GetPermissionsByRoleID(roleID string) ([]models.Permission, error) {
	parsedID, err := uuid.Parse(roleID)
	if err != nil {
		return nil, fmt.Errorf("invalid role ID: %w", err)
	}

	rows, err := r.queries.GetPermissionsByRoleID(context.Background(), parsedID)
	if err != nil {
		return nil, err
	}
	return toModelPermissions(rows), nil
}

// SetRolePermissions replaces all permissions for a role.
func (r *Repository) SetRolePermissions(roleID string, permissionIDs []string) error {
	parsedRoleID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	// Verify role exists
	_, err = r.queries.GetRoleByID(context.Background(), parsedRoleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("role not found")
		}
		return err
	}

	// Parse permission IDs and verify they exist
	if len(permissionIDs) > 0 {
		parsedPermIDs := make([]uuid.UUID, len(permissionIDs))
		for i, pid := range permissionIDs {
			parsed, err := uuid.Parse(pid)
			if err != nil {
				return fmt.Errorf("invalid permission ID %q: %w", pid, err)
			}
			parsedPermIDs[i] = parsed
		}

		found, err := r.queries.GetPermissionsByIDs(context.Background(), parsedPermIDs)
		if err != nil {
			return err
		}
		if len(found) != len(parsedPermIDs) {
			return fmt.Errorf("one or more permission IDs not found")
		}
	}

	// Transaction: delete all existing, then insert new
	ctx := context.Background()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	if err := qtx.DeleteRolePermissions(ctx, parsedRoleID); err != nil {
		return err
	}

	for _, pid := range permissionIDs {
		parsedPermID, _ := uuid.Parse(pid) // Already validated above
		if err := qtx.InsertRolePermission(ctx, sqlcgen.InsertRolePermissionParams{
			RoleID:       parsedRoleID,
			PermissionID: parsedPermID,
		}); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// ============================================================
// User-Role Operations
// ============================================================

// GetUserRoles returns all roles assigned to a user in a specific application.
func (r *Repository) GetUserRoles(appID, userID string) ([]models.Role, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	rows, err := r.queries.GetRolesForUserInApp(context.Background(), sqlcgen.GetRolesForUserInAppParams{
		AppID:  parsedAppID,
		UserID: parsedUserID,
	})
	if err != nil {
		return nil, err
	}

	roles := make([]models.Role, len(rows))
	for i, row := range rows {
		roles[i] = toModelRole(row)
		// Load permissions for each role
		perms, err := r.queries.GetPermissionsByRoleID(context.Background(), row.ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = toModelPermissions(perms)
	}

	return roles, nil
}

// GetUserRoleNames returns just the role names for a user in an application.
func (r *Repository) GetUserRoleNames(appID, userID string) ([]string, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return r.queries.GetUserRoleNames(context.Background(), sqlcgen.GetUserRoleNamesParams{
		AppID:  parsedAppID,
		UserID: parsedUserID,
	})
}

// GetUserPermissions returns all unique permission strings for a user in an application.
// Returns permissions as "resource:action" strings.
func (r *Repository) GetUserPermissions(appID, userID string) ([]string, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	rows, err := r.queries.GetUserPermissions(context.Background(), sqlcgen.GetUserPermissionsParams{
		AppID:  parsedAppID,
		UserID: parsedUserID,
	})
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(rows))
	for _, p := range rows {
		result = append(result, p.Resource+":"+p.Action)
	}
	return result, nil
}

// AssignRoleToUser assigns a role to a user. Returns error if already assigned.
func (r *Repository) AssignRoleToUser(userID, roleID, appID string, assignedBy *uuid.UUID) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	parsedRoleID, err := uuid.Parse(roleID)
	if err != nil {
		return err
	}
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return err
	}

	var pgAssignedBy pgtype.UUID
	if assignedBy != nil {
		pgAssignedBy = pgtype.UUID{Bytes: *assignedBy, Valid: true}
	}

	return r.queries.CreateUserRole(context.Background(), sqlcgen.CreateUserRoleParams{
		UserID:     parsedUserID,
		RoleID:     parsedRoleID,
		AppID:      parsedAppID,
		AssignedAt: time.Now().UTC(),
		AssignedBy: pgAssignedBy,
	})
}

// RevokeRoleFromUser removes a role assignment from a user.
func (r *Repository) RevokeRoleFromUser(userID, roleID string) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	parsedRoleID, err := uuid.Parse(roleID)
	if err != nil {
		return err
	}

	return r.queries.DeleteUserRole(context.Background(), sqlcgen.DeleteUserRoleParams{
		UserID: parsedUserID,
		RoleID: parsedRoleID,
	})
}

// GetUsersWithRoleInApp returns all user-role assignments for an app, with user info.
func (r *Repository) GetUsersWithRoleInApp(appID string, page, pageSize int) ([]UserRoleListItem, int64, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid app ID: %w", err)
	}

	total, err := r.queries.CountUsersWithRoleInApp(context.Background(), parsedAppID)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.queries.GetUsersWithRoleInApp(context.Background(), sqlcgen.GetUsersWithRoleInAppParams{
		AppID:  parsedAppID,
		Limit:  safeconv.ToInt32(pageSize),
		Offset: safeconv.ToInt32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]UserRoleListItem, len(rows))
	for i, row := range rows {
		items[i] = UserRoleListItem{
			UserID:     row.UserID,
			RoleID:     row.RoleID,
			AppID:      row.AppID,
			UserEmail:  row.UserEmail,
			UserName:   row.UserName,
			RoleName:   row.RoleName,
			AssignedAt: row.AssignedAt.Format(time.RFC3339),
		}
	}

	return items, total, nil
}

// UserRoleListItem represents a user-role assignment for list views.
type UserRoleListItem struct {
	UserID     uuid.UUID `json:"user_id"`
	RoleID     uuid.UUID `json:"role_id"`
	AppID      uuid.UUID `json:"app_id"`
	UserEmail  string    `json:"user_email"`
	UserName   string    `json:"user_name"`
	RoleName   string    `json:"role_name"`
	AssignedAt string    `json:"assigned_at"`
}

// GetUserRolesForUserInApp returns all role assignments for a specific user in an app.
func (r *Repository) GetUserRolesForUserInApp(appID, userID string) ([]models.UserRole, error) {
	parsedAppID, err := uuid.Parse(appID)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get user_role rows
	urRows, err := r.queries.GetUserRolesForUserInApp(context.Background(), sqlcgen.GetUserRolesForUserInAppParams{
		AppID:  parsedAppID,
		UserID: parsedUserID,
	})
	if err != nil {
		return nil, err
	}

	result := make([]models.UserRole, len(urRows))
	for i, ur := range urRows {
		result[i] = models.UserRole{
			UserID:     ur.UserID,
			RoleID:     ur.RoleID,
			AppID:      ur.AppID,
			AssignedAt: ur.AssignedAt,
			AssignedBy: pgtypeUUIDToPtr(ur.AssignedBy),
		}
		// Load the associated Role (Preload("Role") equivalent)
		roleRow, err := r.queries.GetRoleByID(context.Background(), ur.RoleID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return nil, err
		}
		result[i].Role = toModelRole(roleRow)
	}

	return result, nil
}

// ListAllApps returns all applications (id, name) for dropdown selects.
func (r *Repository) ListAllApps() ([]models.Application, error) {
	rows, err := r.queries.ListAllApps(context.Background())
	if err != nil {
		return nil, err
	}

	apps := make([]models.Application, len(rows))
	for i, row := range rows {
		apps[i] = models.Application{
			ID:   row.ID,
			Name: row.Name,
		}
	}
	return apps, nil
}

// ============================================================
// Type conversion helpers
// ============================================================

func toModelRole(row sqlcgen.Role) models.Role {
	return models.Role{
		ID:          row.ID,
		AppID:       row.AppID,
		Name:        row.Name,
		Description: derefStr(row.Description),
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toModelPermission(row sqlcgen.Permission) models.Permission {
	return models.Permission{
		ID:          row.ID,
		Resource:    row.Resource,
		Action:      row.Action,
		Description: derefStr(row.Description),
		CreatedAt:   row.CreatedAt,
	}
}

func toModelPermissions(rows []sqlcgen.Permission) []models.Permission {
	perms := make([]models.Permission, len(rows))
	for i, row := range rows {
		perms[i] = toModelPermission(row)
	}
	return perms
}

func pgtypeUUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

func strPtr(s string) *string {
	return &s
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
