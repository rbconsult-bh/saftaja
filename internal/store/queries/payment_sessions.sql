-- name: CreatePaymentSession :one
INSERT INTO payment_sessions (invoice_id, project_id, gateway_account_id, gateway_session_id, payment_method, payer_ip, payer_user_agent)
  VALUES ($1, $2, $3, $4, $5, $6, $7)
  RETURNING *;
