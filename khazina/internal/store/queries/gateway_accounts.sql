-- name: ListActiveGatewayAccounts :many
SELECT * FROM gateway_accounts
WHERE project_id = $1
AND is_active = true
AND deleted_at IS NULL;

-- name: GetGatewayAccount :one
SELECT * FROM gateway_accounts
WHERE id = $1 LIMIT 1;

-- name: GetGatewayAccountByIDAndProject :one
SELECT * FROM gateway_accounts
WHERE id = $1 AND project_id = $2 LIMIT 1;

-- name: GetGatewayAccountByPaymentSessionID :one
SELECT * FROM gateway_accounts ga
JOIN payment_sessions ps ON ga.id = ps.gateway_account_id
WHERE ps.id = $1;

-- name: CreateGatewayAccount :one
INSERT INTO gateway_accounts (project_id, connector_type, account_name, credentials, settings, payment_methods, is_active)
VALUES ($1, $2, $3, $4, $5, $6, true) RETURNING *;
