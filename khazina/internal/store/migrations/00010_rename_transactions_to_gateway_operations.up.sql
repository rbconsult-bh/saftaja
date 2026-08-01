ALTER TABLE transactions RENAME TO gateway_operations;

ALTER TABLE gateway_operations RENAME COLUMN transaction_type TO operation_type;

ALTER TABLE gateway_operations RENAME COLUMN gateway_transaction_id TO gateway_reference;

ALTER TABLE gateway_operations ADD COLUMN gateway_account_id UUID;

UPDATE gateway_operations go
SET gateway_account_id = pi.gateway_account_id
FROM payment_intents pi
WHERE go.payment_intent_id = pi.id;

ALTER TABLE gateway_operations ALTER COLUMN gateway_account_id SET NOT NULL;

ALTER TABLE gateway_operations ADD CONSTRAINT gateway_operations_gateway_account_id_fkey
  FOREIGN KEY (gateway_account_id) REFERENCES gateway_accounts(id);

ALTER TABLE gateway_operations RENAME CONSTRAINT transactions_pkey
  TO gateway_operations_pkey;

ALTER TABLE gateway_operations RENAME CONSTRAINT transactions_payment_intent_id_fkey
  TO gateway_operations_payment_intent_id_fkey;

ALTER TABLE gateway_operations RENAME CONSTRAINT transactions_invoice_id_fkey
  TO gateway_operations_invoice_id_fkey;

ALTER TABLE gateway_operations RENAME CONSTRAINT transactions_project_id_fkey
  TO gateway_operations_project_id_fkey;

ALTER TRIGGER update_transactions_updated_at ON gateway_operations
RENAME TO update_gateway_operations_updated_at;

ALTER INDEX idx_one_successful_payment_per_invoice
RENAME TO idx_one_successful_gateway_operation_pay_per_invoice;
