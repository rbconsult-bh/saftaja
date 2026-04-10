-- name: CreateCustomerIfNotExists :one
INSERT INTO customer (name, email)
VALUES ($1, $2)
ON CONFLICT (email) DO UPDATE SET email = customer.email
RETURNING *, (xmax != 0) as did_exit_before;

