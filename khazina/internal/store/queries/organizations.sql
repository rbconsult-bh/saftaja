-- name: CreateDefaultOrganizationForCustomer :one
WITH new_org AS (
  INSERT INTO organizations (name)
  VALUES ($1)
  RETURNING *
)
INSERT INTO organization_customer (organization_id, customer_id, role)
SELECT new_org.id, $2, 'owner'
FROM new_org
RETURNING organization_id;
