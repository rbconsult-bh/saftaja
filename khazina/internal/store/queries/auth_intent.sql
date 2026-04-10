-- name: CreateAuthIntent :exec
INSERT INTO auth_intent (email, token_hash, expires_at, created_at)
VALUES ($1, $2, NOW() + INTERVAL '15 minutes', NOW())
ON CONFLICT (email)
	DO UPDATE
	SET token_hash = $2,
      expires_at = NOW() + INTERVAL '15 minutes';

-- name: ConsumeAuthIntentByTokenHash :one
DELETE FROM auth_intent
WHERE token_hash = $1
  AND expires_at > NOW()
RETURNING email;
