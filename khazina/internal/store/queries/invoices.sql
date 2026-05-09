-- name: GetInvoiceByID :one
SELECT * FROM invoices WHERE id = $1;

-- name: GetInvoiceByIDAndProject :one
SELECT * FROM invoices WHERE id = $1 AND project_id = $2;

-- name: UpdateInvoiceStatus :exec
UPDATE invoices
SET status = $2
WHERE id = $1;

-- name: CreateInvoice :one
INSERT INTO invoices (project_id, amount, currency, customer_email, customer_name, description, status)
VALUES ($1, $2, $3, $4, $5, $6, 'pending') RETURNING *;
