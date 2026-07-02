ALTER TABLE payment_intents
  ALTER COLUMN payer_ip DROP NOT NULL,
  ALTER COLUMN payer_user_agent DROP NOT NULL,
  ALTER COLUMN idempotency_key DROP NOT NULL;
