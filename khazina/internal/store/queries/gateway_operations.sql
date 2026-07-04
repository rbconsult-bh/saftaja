-- name: CreateGatewayOperation :one
INSERT INTO gateway_operations (
    payment_intent_id, invoice_id, project_id, gateway_account_id,
    operation_type, gateway_reference, amount, currency, raw_request
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetPayGatewayOperationByPaymentIntentID :one
SELECT * FROM gateway_operations
WHERE payment_intent_id = $1
AND operation_type = 'pay'
AND status = 'success'
LIMIT 1;

-- name: UpdateGatewayOperationStatus :exec
UPDATE gateway_operations
SET status = $2, raw_response = $3
WHERE id = $1;

-- name: GetSuccessfulAuthGatewayOperation :one
SELECT * FROM gateway_operations
WHERE payment_intent_id = $1
AND operation_type = 'authenticate_payer'
AND status = 'success'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLatestGatewayOperation :one
SELECT * FROM gateway_operations
WHERE payment_intent_id = $1
ORDER BY created_at DESC
LIMIT 1;
