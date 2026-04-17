-- name: CreateCustomerSession :one
INSERT INTO customer_session (customer_id, current_jti_hash, expires_at)
VALUES ($1, $2, NOW() + INTERVAL '30 days')
RETURNING *;

-- name: DeleteCustomerSessionByIDAndCustomerID :exec
DELETE FROM customer_session
WHERE id = $1 AND customer_id = $2;

-- name: UpdateCustomerSessionJtiHashByIDAndCustomerID :one
UPDATE customer_session
SET current_jti_hash = $1
WHERE id = $2 AND customer_id = $3 AND expires_at > NOW()
RETURNING id;
