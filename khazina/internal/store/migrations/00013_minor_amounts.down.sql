ALTER TABLE gateway_operations
  ADD COLUMN amount DECIMAL(12, 3) NOT NULL,
  ADD COLUMN currency VARCHAR(3) NOT NULL;

ALTER TABLE payment_intents
  DROP COLUMN currency,
  DROP COLUMN amount_minor;

ALTER TABLE invoice_items DROP COLUMN amount_minor;
ALTER TABLE invoice_items DROP COLUMN unit_price_minor;
ALTER TABLE invoice_items ADD COLUMN amount DECIMAL(12, 3) NOT NULL;
ALTER TABLE invoice_items ADD COLUMN unit_price DECIMAL(12, 3) NOT NULL;

ALTER TABLE invoices DROP COLUMN amount_minor;
ALTER TABLE invoices ADD COLUMN amount DECIMAL(12, 3) NOT NULL;
