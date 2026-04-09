-- name: CreateAuthIntent :exec
INSERT INTO auth_intent (email, token_hash, expires_at, created_at)
VALUES ($1, $2, NOW() + INTERVAL '15 minutes', NOW());
