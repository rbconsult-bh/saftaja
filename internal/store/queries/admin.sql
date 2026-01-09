-- name: CreateOrganization :one
INSERT INTO organizations (name) VALUES ($1) RETURNING *;

-- name: CreateProject :one
INSERT INTO projects (organization_id, name, environment, custom_domain)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: CreateGatewayAccount :one
INSERT INTO gateway_accounts (project_id, connector_type, account_name, credentials, settings, payment_methods, is_active)
VALUES ($1, $2, $3, $4, $5, $6, true) RETURNING *;

-- name: CreateInvoice :one
INSERT INTO invoices (project_id, amount, currency, customer_email, customer_name, description, status)
VALUES ($1, $2, $3, $4, $5, $6, 'pending') RETURNING *;
