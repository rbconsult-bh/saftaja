-- name: ListActiveGatewayAccounts :many
SELECT * FROM gateway_accounts
WHERE project_id = $1
AND is_active = true
AND deleted_at IS NULL;

-- name: GetGatewayAccount :one
SELECT * FROM gateway_accounts
WHERE id = $1 LIMIT 1;

-- name: GetLatestPaymentSession :one
SELECT * FROM payment_sessions
WHERE invoice_id = $1
AND status IN ('created', 'authenticating', 'authenticated')
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPaymentSessionByID :one
SELECT * FROM payment_sessions
WHERE id = $1 LIMIT 1;

-- name: UpdatePaymentSessionGatewayID :exec
UPDATE payment_sessions
SET gateway_session_id = $2
WHERE id = $1;

-- name: UpdatePaymentSessionStatus :exec
UPDATE payment_sessions
SET status = $2
WHERE id = $1;

-- name: CreateTransaction :one
INSERT INTO transactions (
    payment_session_id, invoice_id, project_id,
    transaction_type, gateway_transaction_id, amount, currency
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdateTransactionStatus :exec
UPDATE transactions
SET status = $2, raw_response = $3
WHERE id = $1;

-- name: GetSuccessfulAuthTransaction :one
SELECT * FROM transactions
WHERE payment_session_id = $1
AND transaction_type = 'authenticate_payer'
AND status = 'success'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLatestTransaction :one
SELECT * FROM transactions
WHERE payment_session_id = $1
ORDER BY created_at DESC
LIMIT 1;
