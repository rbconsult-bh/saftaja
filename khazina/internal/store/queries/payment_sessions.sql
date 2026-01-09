-- name: CreatePaymentSession :one
INSERT INTO payment_sessions (invoice_id, project_id, gateway_account_id, gateway_session_id, payment_method, payer_ip, payer_user_agent, idempotency_key)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
  RETURNING *;

-- name: GetPaymentSessionByIdempotencyKey :one
SELECT * FROM payment_sessions
WHERE invoice_id = $1
AND idempotency_key = $2
LIMIT 1;

