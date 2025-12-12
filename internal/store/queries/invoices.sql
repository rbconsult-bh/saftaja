-- name: CreateInvoice :one
INSERT INTO invoices (project_id, amount, currency, external_id, customer_email, customer_name, description)
  VALUES ($1, $2, $3, $4, $5, $6, $7)
  RETURNING *;
