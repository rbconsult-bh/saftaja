-- 1. Revert the trigger name
ALTER TRIGGER update_payment_intents_updated_at ON payment_intents 
RENAME TO update_payment_sessions_updated_at;

-- 2. Revert the reference column in transactions
ALTER TABLE transactions RENAME COLUMN payment_intent_id TO payment_session_id;

-- 3. Revert the main table name
ALTER TABLE payment_intents RENAME TO payment_sessions;
