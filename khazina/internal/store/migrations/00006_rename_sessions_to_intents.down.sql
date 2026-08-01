-- 5. Revert foreign key constraint names
ALTER TABLE payment_intents RENAME CONSTRAINT payment_intents_gateway_account_id_fkey
  TO payment_sessions_gateway_account_id_fkey;

ALTER TABLE payment_intents RENAME CONSTRAINT payment_intents_project_id_fkey
  TO payment_sessions_project_id_fkey;

ALTER TABLE payment_intents RENAME CONSTRAINT payment_intents_invoice_id_fkey
  TO payment_sessions_invoice_id_fkey;

ALTER TABLE transactions RENAME CONSTRAINT transactions_payment_intent_id_fkey
  TO transactions_payment_session_id_fkey;

-- 4. Revert gateway_session_id back to NOT NULL
--    (requires setting a default for any existing NULLs first)
UPDATE payment_intents SET gateway_session_id = '' WHERE gateway_session_id IS NULL;
ALTER TABLE payment_intents ALTER COLUMN gateway_session_id SET NOT NULL;

-- 3. Revert the trigger name
ALTER TRIGGER update_payment_intents_updated_at ON payment_intents 
RENAME TO update_payment_sessions_updated_at;

-- 2. Revert the reference column in transactions
ALTER TABLE transactions RENAME COLUMN payment_intent_id TO payment_session_id;

-- 1. Revert the main table name
ALTER TABLE payment_intents RENAME TO payment_sessions;
