-- name: CreateDefaultOrganizationForCustomer :exec
WITH new_org AS (
  INSERT INTO organizations (name)
  VALUES ($1)
  RETURNING *
)
INSERT INTO organization_customer (organization_id, customer_id)
VALUES (new_org.id, $2);
