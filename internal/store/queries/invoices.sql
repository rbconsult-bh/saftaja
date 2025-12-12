-- name: GetInvoiceByID :one
SELECT * FROM invoices WHERE id = $1;
