-- name: CreateTransaction :one
INSERT INTO transactions (
    payment_intent_id, invoice_id, project_id,
    transaction_type, gateway_transaction_id, amount, currency, raw_request
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetPayTransactionByPaymentIntentID :one
SELECT * FROM transactions
WHERE payment_intent_id = $1
AND transaction_type = 'pay'
AND status = 'success'
LIMIT 1;

-- name: UpdateTransactionStatus :exec
UPDATE transactions
SET status = $2, raw_response = $3
WHERE id = $1;

-- name: GetSuccessfulAuthTransaction :one
SELECT * FROM transactions
WHERE payment_intent_id = $1
AND transaction_type = 'authenticate_payer'
AND status = 'success'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLatestTransaction :one
SELECT * FROM transactions
WHERE payment_intent_id = $1
ORDER BY created_at DESC
LIMIT 1;
