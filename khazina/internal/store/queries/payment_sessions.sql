-- name: CreatePaymentSession :one
INSERT INTO payment_sessions (invoice_id, project_id, gateway_account_id, gateway_session_id, payment_method, payer_ip, payer_user_agent, idempotency_key)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
  RETURNING *;

-- name: GetPaymentSessionByIdempotencyKey :one
SELECT * FROM payment_sessions
WHERE invoice_id = $1
AND idempotency_key = $2
LIMIT 1;


-- name: GetLatestPaymentSession :one
SELECT * FROM payment_sessions
WHERE invoice_id = $1
AND status IN ('created', 'authenticating', 'authenticated')
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPaymentSessionByID :one
SELECT * FROM payment_sessions
WHERE id = $1 LIMIT 1;

-- name: GetPaymentSessionByIDAndProject :one
SELECT * FROM payment_sessions
WHERE id = $1 AND project_id = $2 LIMIT 1;

-- name: UpdatePaymentSessionGatewayID :exec
UPDATE payment_sessions
SET gateway_session_id = $2
WHERE id = $1;

-- name: UpdatePaymentSessionStatus :exec
UPDATE payment_sessions
SET status = $2
WHERE id = $1;
