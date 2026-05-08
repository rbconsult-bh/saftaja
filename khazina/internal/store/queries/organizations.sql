-- name: CreateOrganizationForCustomer :one
WITH new_org AS (
  INSERT INTO organizations (name)
  VALUES ($1)
  RETURNING *
)
INSERT INTO organization_customer (organization_id, customer_id, role)
SELECT new_org.id, $2, $3
FROM new_org
RETURNING organization_id;

-- name: ListOrganizationsWithProjectsForCustomer :many
SELECT
  o.id AS org_id,
  o.name AS org_name,
  oc.role AS role,
  p.id AS project_id,
  p.name AS project_name,
  p.environment AS environment
FROM organizations o
JOIN organization_customer oc ON oc.organization_id = o.id AND oc.customer_id = $1
LEFT JOIN projects p ON p.organization_id = o.id AND p.deleted_at IS NULL
WHERE o.deleted_at IS NULL
ORDER BY o.created_at, p.created_at;
