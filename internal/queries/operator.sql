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
