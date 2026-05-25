-- 1. Rename the main table
ALTER TABLE payment_sessions RENAME TO payment_intents;

-- 2. Rename the reference column in the transactions table
ALTER TABLE transactions RENAME COLUMN payment_session_id TO payment_intent_id;

-- 3. Rename the trigger so it stays clean
ALTER TRIGGER update_payment_sessions_updated_at ON payment_intents 
RENAME TO update_payment_intents_updated_at;

-- 4. Make gateway_session_id nullable (Apple Pay and other direct flows
--    don't create an MPGSA session)
ALTER TABLE payment_intents ALTER COLUMN gateway_session_id DROP NOT NULL;

-- 5. Fix foreign key constraint names to match new table/column names
ALTER TABLE transactions RENAME CONSTRAINT transactions_payment_session_id_fkey
  TO transactions_payment_intent_id_fkey;

ALTER TABLE payment_intents RENAME CONSTRAINT payment_sessions_invoice_id_fkey
  TO payment_intents_invoice_id_fkey;

ALTER TABLE payment_intents RENAME CONSTRAINT payment_sessions_project_id_fkey
  TO payment_intents_project_id_fkey;

ALTER TABLE payment_intents RENAME CONSTRAINT payment_sessions_gateway_account_id_fkey
  TO payment_intents_gateway_account_id_fkey;
