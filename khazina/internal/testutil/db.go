package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/stretchr/testify/require"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const baseDSN = "postgres://test_user:test_password@localhost:5433/test_khazina?sslmode=disable"
const migrationAdvisoryLockID int64 = 917431742001

type TestDB struct {
	Pool   *pgxpool.Pool
	Schema string
	DSN    string
}

func SetupIsolatedDB(t *testing.T) TestDB {
	t.Helper()

	ctx := context.Background()

	schemaName := "test_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	config, err := pgxpool.ParseConfig(baseDSN)
	require.NoError(t, err)

	adminPool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)

	_, err = adminPool.Exec(
		ctx,
		fmt.Sprintf(`CREATE SCHEMA "%s"`, schemaName),
	)
	require.NoError(t, err)

	schemaDSN := baseDSN + "&search_path=" + schemaName

	migrationConn, err := adminPool.Acquire(ctx)
	require.NoError(t, err)
	defer migrationConn.Release()

	_, err = migrationConn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationAdvisoryLockID)
	require.NoError(t, err)
	defer func() {
		_, _ = migrationConn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", migrationAdvisoryLockID)
	}()

	err = store.RunMigrationsInSchema(schemaDSN, schemaName)
	require.NoError(t, err)

	testConfig, err := pgxpool.ParseConfig(baseDSN)
	require.NoError(t, err)

	testConfig.ConnConfig.RuntimeParams["search_path"] = schemaName

	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = adminPool.Exec(
			context.Background(),
			fmt.Sprintf(`DROP SCHEMA "%s" CASCADE`, schemaName),
		)

		testPool.Close()
		adminPool.Close()
	})

	return TestDB{
		Pool:   testPool,
		Schema: schemaName,
		DSN:    schemaDSN,
	}
}

func SetupIsolatedDBWithFixtures(
	t *testing.T,
	fixturesDir string,
) TestDB {
	t.Helper()

	db := SetupIsolatedDB(t)

	LoadFixtures(t, db.DSN, fixturesDir)

	return db
}

func LoadFixtures(
	t *testing.T,
	dsn string,
	fixturesDir string,
) {
	t.Helper()

	sqlDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	fixtures, err := testfixtures.New(
		testfixtures.Database(sqlDB),
		testfixtures.Dialect("postgresql", testfixtures.RespectSearchPath()),
		testfixtures.Directory(fixturesDir),
	)
	require.NoError(t, err)

	err = fixtures.Load()
	require.NoError(t, err)
}
