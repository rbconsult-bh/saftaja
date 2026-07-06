-- name: CreatePaymentIntent :one
INSERT INTO payment_intents (invoice_id, project_id, gateway_account_id, gateway_session_id, payment_method, payer_ip, payer_user_agent, idempotency_key)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
  RETURNING *;

-- name: GetPaymentIntentByIdempotencyKeyAndProject :one
SELECT * FROM payment_intents
WHERE project_id = $1
AND idempotency_key = $2
LIMIT 1;


-- name: GetLatestPaymentIntent :one
SELECT * FROM payment_intents
WHERE invoice_id = $1
AND status IN ('created', 'authenticating', 'authenticated')
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPaymentIntentByID :one
SELECT * FROM payment_intents
WHERE id = $1 LIMIT 1;

-- name: GetPaymentIntentByIDAndProject :one
SELECT * FROM payment_intents
WHERE id = $1 AND project_id = $2 LIMIT 1;

-- name: UpdatePaymentIntentsGatewayID :exec
UPDATE payment_intents
SET gateway_session_id = $2
WHERE id = $1;

-- name: UpdatePaymentIntentStatus :exec
UPDATE payment_intents
SET status = $2
WHERE id = $1;

-- name: GetPaymentIntentByIDAndProjectAndInvoiceForUpdate :one
SELECT * FROM payment_intents
WHERE id = $1 AND project_id = $2 AND invoice_id = $3 LIMIT 1
FOR UPDATE;
