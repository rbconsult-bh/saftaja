-- name: CreatePaymentIntent :one
INSERT INTO payment_intents (invoice_id, project_id, gateway_account_id, gateway_setup_reference, payment_method, payer_ip, payer_user_agent, idempotency_key, amount_minor, currency)
  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
  RETURNING *;

-- name: GetPaymentIntentByIdempotencyKeyAndProject :one
SELECT * FROM payment_intents
WHERE project_id = $1
AND idempotency_key = $2
LIMIT 1;

-- name: GetPaymentIntentByID :one
SELECT * FROM payment_intents
WHERE id = $1 LIMIT 1;

-- name: GetPaymentIntentByIDAndProject :one
SELECT * FROM payment_intents
WHERE id = $1 AND project_id = $2 LIMIT 1;

-- name: UpdatePaymentIntentGatewaySetupReference :exec
UPDATE payment_intents
SET gateway_setup_reference = $2
WHERE id = $1;

-- name: UpdatePaymentIntentStatus :exec
UPDATE payment_intents
SET status = $2
WHERE id = $1;

-- name: GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdate :one
SELECT * FROM payment_intents
WHERE id = $1 AND project_id = $2 AND invoice_id = $3 LIMIT 1
FOR NO KEY UPDATE;

-- name: InvoiceHasOtherCapturingPaymentIntent :one
SELECT EXISTS (
  SELECT 1 FROM payment_intents
  WHERE invoice_id = $1
    AND id != $2
    AND status = 'capturing'
    AND deleted_at IS NULL
);
