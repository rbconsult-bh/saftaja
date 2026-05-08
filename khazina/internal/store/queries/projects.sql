-- name: CreateProjectForOrganization :one
INSERT INTO projects (organization_id, name, environment, custom_domain)
VALUES ($1, $2, $3, $4)
RETURNING id;
