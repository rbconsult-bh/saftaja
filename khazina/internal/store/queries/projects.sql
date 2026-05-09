-- name: CreateProjectForOrganization :one
INSERT INTO projects (organization_id, name, environment, custom_domain)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: GetProjectByPaymentSessionID :one
SELECT * FROM projects p
JOIN payment_sessions ps ON p.id = ps.project_id
WHERE ps.id = $1;

-- name: GetProjectByCustomDomain :one
SELECT * FROM projects
WHERE custom_domain = $1
AND deleted_at IS NULL
LIMIT 1;
