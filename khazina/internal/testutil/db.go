package testutil

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/server"
	"github.com/stretchr/testify/require"
)

func SetupIsolatedDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	dsn := "postgres://test_user:test_password@localhost:5433/test_khazina?sslmode=disable"

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	schemaName := "test_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	_, err = pool.Exec(ctx, "CREATE SCHEMA "+schemaName)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "SET search_path TO "+schemaName)
	require.NoError(t, err)

	migrationDSN := dsn + "&search_path=" + schemaName

	err = server.RunMigrations(migrationDSN)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
		pool.Close()
	})

	return pool
}
