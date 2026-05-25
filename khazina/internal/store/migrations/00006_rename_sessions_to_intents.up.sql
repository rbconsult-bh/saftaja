-- 1. Rename the main table
ALTER TABLE payment_sessions RENAME TO payment_intents;

-- 2. Rename the reference column in the transactions table
ALTER TABLE transactions RENAME COLUMN payment_session_id TO payment_intent_id;

-- 3. Rename the trigger so it stays clean
ALTER TRIGGER update_payment_sessions_updated_at ON payment_intents 
RENAME TO update_payment_intents_updated_at;
