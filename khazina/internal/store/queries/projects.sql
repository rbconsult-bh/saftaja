-- name: CreateProjectForOrganization :one
INSERT INTO projects (organization_id, name, environment, custom_domain)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: GetProjectByPaymentIntentID :one
SELECT * FROM projects p
JOIN payment_intents pi ON p.id = pi.project_id
WHERE pi.id = $1;

-- name: GetProjectByCustomDomain :one
SELECT * FROM projects
WHERE custom_domain = $1
AND deleted_at IS NULL
LIMIT 1;
