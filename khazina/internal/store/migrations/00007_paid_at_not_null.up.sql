ALTER TABLE invoices ADD CONSTRAINT paid_invoices_have_paid_at
  CHECK (status != 'paid' OR paid_at IS NOT NULL);
