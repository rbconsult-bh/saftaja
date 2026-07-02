ALTER TABLE payment_intents
  ALTER COLUMN payer_ip SET NOT NULL,
  ALTER COLUMN payer_user_agent SET NOT NULL,
  ALTER COLUMN idempotency_key SET NOT NULL;
