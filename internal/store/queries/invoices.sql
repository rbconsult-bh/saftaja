-- name: GetInvoiceByID :one
SELECT * FROM invoices WHERE id = $1;

-- name: UpdateInvoiceStatus :exec
UPDATE invoices
SET status = $2
WHERE id = $1;
