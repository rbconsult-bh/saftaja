-- name: CreateCustomerIfNotExists :one
WITH upserted AS (
  INSERT INTO customer (name, email)
  VALUES ($1, $2)
  ON CONFLICT (email) DO UPDATE SET email = customer.email
  RETURNING *
)
SELECT 
  upserted.*,
  NOT EXISTS (
    SELECT 1 FROM organization_customer 
    WHERE customer_id = upserted.id
  ) AS needs_default_org
FROM upserted;

