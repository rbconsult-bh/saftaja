ALTER TABLE invoices DROP COLUMN amount;
ALTER TABLE invoices ADD COLUMN amount_minor BIGINT NOT NULL;

ALTER TABLE invoice_items DROP COLUMN unit_price;
ALTER TABLE invoice_items DROP COLUMN amount;
ALTER TABLE invoice_items ADD COLUMN unit_price_minor BIGINT NOT NULL;
ALTER TABLE invoice_items ADD COLUMN amount_minor BIGINT NOT NULL;

ALTER TABLE payment_intents
  ADD COLUMN amount_minor BIGINT NOT NULL,
  ADD COLUMN currency VARCHAR(3) NOT NULL;

ALTER TABLE gateway_operations
  DROP COLUMN amount,
  DROP COLUMN currency;
