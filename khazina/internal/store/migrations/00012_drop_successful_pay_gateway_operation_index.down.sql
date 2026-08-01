CREATE UNIQUE INDEX idx_one_successful_gateway_operation_pay_per_invoice
ON gateway_operations(invoice_id)
WHERE status = 'success' AND operation_type = 'pay';
