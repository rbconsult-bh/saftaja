-- name: CreateCustomerSession :one
INSERT INTO customer_session (customer_id, current_jti_hash, expires_at)
VALUES ($1, $2, NOW() + INTERVAL '30 days')
RETURNING *;

-- name: DeleteCustomerSessionByIDAndCustomerID :exec
DELETE FROM customer_session
WHERE id = $1 AND customer_id = $2;

