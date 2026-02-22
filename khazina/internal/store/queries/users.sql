-- ============================================================================
-- USERS
-- ============================================================================

-- name: CreateUser :one
INSERT INTO users (email, password_hash, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- ============================================================================
-- ORGANIZATIONS (v1 - user scoped)
-- ============================================================================

-- name: CreateOrganizationV1 :one
INSERT INTO organizations (name)
VALUES ($1)
RETURNING *;

-- name: ListUserOrganizations :many
SELECT o.id, o.name, o.created_at, o.updated_at, o.deleted_at
FROM organizations o
JOIN organization_members om ON om.organization_id = o.id
WHERE om.user_id = $1 AND o.deleted_at IS NULL
ORDER BY o.created_at ASC;

-- ============================================================================
-- ORGANIZATION MEMBERS
-- ============================================================================

-- name: CreateOrganizationMember :one
INSERT INTO organization_members (organization_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOrganizationMember :one
SELECT * FROM organization_members
WHERE organization_id = $1 AND user_id = $2;

-- name: ListOrganizationMembers :many
SELECT om.id, om.organization_id, om.user_id, om.role, om.created_at,
       u.email, u.name AS user_name
FROM organization_members om
JOIN users u ON u.id = om.user_id
WHERE om.organization_id = $1
ORDER BY om.created_at ASC;

-- name: DeleteOrganizationMember :exec
DELETE FROM organization_members
WHERE organization_id = $1 AND user_id = $2;

-- ============================================================================
-- API KEYS
-- ============================================================================

-- name: CreateAPIKey :one
INSERT INTO api_keys (organization_id, project_id, user_id, name, key_hash, key_prefix)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAPIKeysByUser :many
SELECT * FROM api_keys
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = $1 AND revoked_at IS NULL;

-- name: RevokeAPIKey :exec
UPDATE api_keys SET revoked_at = NOW()
WHERE id = $1 AND user_id = $2;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = NOW()
WHERE id = $1;

-- ============================================================================
-- INVITATIONS
-- ============================================================================

-- name: CreateInvitation :one
INSERT INTO invitations (organization_id, email, role, invited_by, token, expires_at)
VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '7 days')
RETURNING *;

-- name: GetInvitationByToken :one
SELECT * FROM invitations
WHERE token = $1 AND accepted_at IS NULL AND expires_at > NOW();

-- name: AcceptInvitation :exec
UPDATE invitations SET accepted_at = NOW()
WHERE id = $1;

-- ============================================================================
-- PROJECTS (v1 - user scoped)
-- ============================================================================

-- name: CreateProjectV1 :one
INSERT INTO projects (organization_id, name, environment, custom_domain, slug)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListProjectsByOrganization :many
SELECT * FROM projects
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: GetProjectByIDAndOrg :one
SELECT * FROM projects
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: UpdateProject :one
UPDATE projects
SET name = $3, custom_domain = $4, slug = $5
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- ============================================================================
-- GATEWAY ACCOUNTS (v1 - user scoped)
-- ============================================================================

-- name: ListGatewayAccountsByProject :many
SELECT * FROM gateway_accounts
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC;

-- ============================================================================
-- INVOICES (v1 - user scoped)
-- ============================================================================

-- name: ListInvoicesByProject :many
SELECT * FROM invoices
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetInvoiceByIDAndOrgProject :one
SELECT i.* FROM invoices i
JOIN projects p ON p.id = i.project_id
JOIN organization_members om ON om.organization_id = p.organization_id
WHERE i.id = $1 AND i.project_id = $2 AND om.user_id = $3 AND i.deleted_at IS NULL;
