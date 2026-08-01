ALTER INDEX idx_one_successful_gateway_operation_pay_per_invoice
RENAME TO idx_one_successful_payment_per_invoice;

ALTER TRIGGER update_gateway_operations_updated_at ON gateway_operations
RENAME TO update_transactions_updated_at;

ALTER TABLE gateway_operations RENAME CONSTRAINT gateway_operations_project_id_fkey
  TO transactions_project_id_fkey;

ALTER TABLE gateway_operations RENAME CONSTRAINT gateway_operations_invoice_id_fkey
  TO transactions_invoice_id_fkey;

ALTER TABLE gateway_operations RENAME CONSTRAINT gateway_operations_payment_intent_id_fkey
  TO transactions_payment_intent_id_fkey;

ALTER TABLE gateway_operations RENAME CONSTRAINT gateway_operations_pkey
  TO transactions_pkey;

ALTER TABLE gateway_operations DROP CONSTRAINT gateway_operations_gateway_account_id_fkey;

ALTER TABLE gateway_operations DROP COLUMN gateway_account_id;

ALTER TABLE gateway_operations RENAME COLUMN gateway_reference TO gateway_transaction_id;

ALTER TABLE gateway_operations RENAME COLUMN operation_type TO transaction_type;

ALTER TABLE gateway_operations RENAME TO transactions;
