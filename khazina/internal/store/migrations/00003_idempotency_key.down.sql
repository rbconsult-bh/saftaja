-- Remove idempotency key from payment_sessions
DROP INDEX IF EXISTS idx_payment_sessions_idempotency_key;
ALTER TABLE payment_sessions DROP COLUMN IF EXISTS idempotency_key;
