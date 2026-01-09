-- Add idempotency key to payment_sessions
ALTER TABLE payment_sessions ADD COLUMN idempotency_key VARCHAR(64);

-- Create unique index for idempotency key per invoice
-- This prevents duplicate sessions for the same idempotency key + invoice combination
CREATE UNIQUE INDEX idx_payment_sessions_idempotency_key
ON payment_sessions(invoice_id, idempotency_key)
WHERE idempotency_key IS NOT NULL;
