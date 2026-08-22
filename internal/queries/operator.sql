-- ============================================================================
-- Operator IAM
-- ============================================================================

-- name: GetOperatorRoleByName :one
SELECT id, name, description, is_system, created_at, updated_at
FROM operator_roles
WHERE name = $1;

-- name: GetOperatorRoleByID :one
SELECT id, name, description, is_system, created_at, updated_at
FROM operator_roles
WHERE id = $1;

-- name: ListOperatorRoles :many
SELECT id, name, description, is_system, created_at, updated_at
FROM operator_roles
ORDER BY is_system DESC, name ASC;

-- name: ListOperatorPermissionsByRoleID :many
SELECT p.id, p.resource, p.action, p.description, p.created_at
FROM operator_permissions p
JOIN operator_role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = $1
ORDER BY p.resource ASC, p.action ASC;

-- name: ListOperatorPermissions :many
SELECT id, resource, action, description, created_at
FROM operator_permissions
ORDER BY resource ASC, action ASC;

-- name: InsertOperatorRole :one
INSERT INTO operator_roles (name, description, is_system)
VALUES ($1, $2, false)
RETURNING id, name, description, is_system, created_at, updated_at;

-- name: UpdateOperatorRoleIfNotSystem :one
UPDATE operator_roles
SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1 AND is_system = false
RETURNING id, name, description, is_system, created_at, updated_at;

-- name: DeleteOperatorRoleIfNotSystem :execrows
DELETE FROM operator_roles
WHERE id = $1 AND is_system = false;

-- name: DeleteOperatorRolePermissions :exec
DELETE FROM operator_role_permissions WHERE role_id = $1;

-- name: InsertOperatorRolePermission :exec
INSERT INTO operator_role_permissions (role_id, permission_id) VALUES ($1, $2);

-- name: GetOperatorPermissionByResourceAction :one
SELECT id, resource, action, description, created_at
FROM operator_permissions
WHERE resource = $1 AND action = $2;

-- name: CountAdminAccountsByOperatorRole :one
SELECT count(*) FROM admin_accounts WHERE operator_role_id = $1;

-- name: CountAPIKeysByOperatorRole :one
SELECT count(*) FROM api_keys WHERE operator_role_id = $1;
