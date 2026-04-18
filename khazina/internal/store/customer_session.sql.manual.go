package store

import (
	"context"

	"github.com/google/uuid"
)

const rotateOrRevokeCustomerSession = `-- name: RotateOrRevokeCustomerSession :one
WITH locked_session AS (
  SELECT id, current_jti_hash
  FROM customer_session cs
  WHERE cs.id = $1
    AND cs.customer_id = $2
    AND cs.expires_at > NOW()
  FOR UPDATE
),
rotated_session AS (
  UPDATE customer_session cs
  SET current_jti_hash = $3, expires_at = NOW() + INTERVAL '30 days'
  FROM locked_session ls
  WHERE cs.id = ls.id
    AND ls.current_jti_hash = $4
  RETURNING cs.id
),
revoked_session AS (
  DELETE FROM customer_session cs
  USING locked_session ls
  WHERE cs.id = ls.id
    AND ls.current_jti_hash != $4
  RETURNING cs.id
)
SELECT
  (SELECT COUNT(*) FROM rotated_session) > 0 AS was_rotated,
  (SELECT COUNT(*) FROM revoked_session) > 0 AS was_revoked;`

type RotateOrRevokeCustomerSessionArgs struct {
	SessionID      uuid.UUID
	CustomerID     uuid.UUID
	NewJtiHash     []byte
	CurrentJtiHash []byte
}

type RotateOrRevokeCustomerSessionRow struct {
	WasRotated bool
	WasRevoked bool
}

func (q *Queries) RotateOrRevokeCustomerSession(ctx context.Context, arg RotateOrRevokeCustomerSessionArgs) (RotateOrRevokeCustomerSessionRow, error) {
	row := q.db.QueryRow(ctx, rotateOrRevokeCustomerSession, arg.SessionID, arg.CustomerID, arg.NewJtiHash, arg.CurrentJtiHash)
	var i RotateOrRevokeCustomerSessionRow
	err := row.Scan(&i.WasRotated, &i.WasRevoked)
	return i, err
}
