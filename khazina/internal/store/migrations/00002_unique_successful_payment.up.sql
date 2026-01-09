-- Prevent double payments: only ONE successful 'pay' transaction per invoice
CREATE UNIQUE INDEX idx_one_successful_payment_per_invoice
ON transactions(invoice_id)
WHERE status = 'success' AND transaction_type = 'pay';
