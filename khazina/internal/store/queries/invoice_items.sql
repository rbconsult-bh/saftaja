-- name: GetInvoiceItems :many
SELECT * FROM invoice_items
WHERE invoice_id = $1
ORDER BY created_at ASC;
