-- ============================================================================
-- Role operations
-- ============================================================================

-- name: GetRolesByAppID :many
SELECT id, app_id, name, description, is_system, created_at, updated_at
FROM roles
WHERE app_id = $1
ORDER BY is_system DESC, name ASC;

-- name: GetRoleByID :one
SELECT id, app_id, name, description, is_system, created_at, updated_at
FROM roles
WHERE id = $1;

-- name: GetRoleByName :one
SELECT id, app_id, name, description, is_system, created_at, updated_at
FROM roles
WHERE app_id = $1 AND name = $2;

-- name: CreateRole :exec
INSERT INTO roles (id, app_id, name, description, is_system, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateRole :exec
UPDATE roles SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1;

-- name: DeleteRoleByID :exec
DELETE FROM roles WHERE id = $1;

-- ============================================================================
-- Permission operations
-- ============================================================================

-- name: GetAllPermissions :many
SELECT id, resource, action, description, created_at
FROM permissions
ORDER BY resource ASC, action ASC;

-- name: GetPermissionByID :one
SELECT id, resource, action, description, created_at
FROM permissions
WHERE id = $1;

-- name: CreatePermission :exec
INSERT INTO permissions (id, resource, action, description, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetPermissionsByRoleID :many
SELECT p.id, p.resource, p.action, p.description, p.created_at
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = $1
ORDER BY p.resource ASC, p.action ASC;

-- ============================================================================
-- Role-Permission operations
-- ============================================================================

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: InsertRolePermission :exec
INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2);

-- name: GetPermissionsByIDs :many
SELECT id, resource, action, description, created_at
FROM permissions
WHERE id = ANY($1::uuid[]);

-- ============================================================================
-- User-Role operations
-- ============================================================================

-- name: GetUserRoleNames :many
SELECT r.name
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.app_id = $1 AND ur.user_id = $2;

-- name: GetUserPermissions :many
SELECT DISTINCT p.resource, p.action
FROM user_roles ur
JOIN role_permissions rp ON rp.role_id = ur.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE ur.app_id = $1 AND ur.user_id = $2;

-- name: CreateUserRole :exec
INSERT INTO user_roles (user_id, role_id, app_id, assigned_at, assigned_by)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteUserRole :exec
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;

-- name: CountUsersWithRoleInApp :one
SELECT COUNT(*) FROM user_roles WHERE app_id = $1;

-- name: GetUsersWithRoleInApp :many
SELECT ur.user_id, ur.role_id, ur.app_id, ur.assigned_at,
       u.email AS user_email, u.name AS user_name,
       r.name AS role_name
FROM user_roles ur
JOIN users u ON u.id = ur.user_id
JOIN roles r ON r.id = ur.role_id
WHERE ur.app_id = $1
ORDER BY u.email ASC, r.name ASC
LIMIT $2 OFFSET $3;

-- name: GetUserRolesForUserInApp :many
SELECT ur.user_id, ur.role_id, ur.app_id, ur.assigned_at, ur.assigned_by
FROM user_roles ur
WHERE ur.app_id = $1 AND ur.user_id = $2;

-- name: GetRolesForUserInApp :many
SELECT r.id, r.app_id, r.name, r.description, r.is_system, r.created_at, r.updated_at
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.app_id = $1 AND ur.user_id = $2;

-- ============================================================================
-- Application operations
-- ============================================================================

-- name: ListAllApps :many
SELECT id, name FROM applications ORDER BY name ASC;
