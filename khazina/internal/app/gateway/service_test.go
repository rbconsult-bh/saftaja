package gateway

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/testutil"
	"github.com/stretchr/testify/require"
)

type gatewayTestEnv struct {
	ctx           context.Context
	queries       store.TransactionQuerier
	encryptionKey []byte
	svc           Service
}

func setupTestEnv(t *testing.T) gatewayTestEnv {
	db := testutil.SetupIsolatedDBWithFixtures(
		t,
		"./testdata/fixtures",
	)

	queries := store.NewTransactionQuerier(db.Pool)

	encryptionKey, err := base64.StdEncoding.DecodeString("rsLjhTr9f1nAhTIRNQG09kO26KpVKKWGx51LsCTKS30=")
	require.NoError(t, err, "encryption key is not decodable from base64")

	svc := New(queries, encryptionKey)

	return gatewayTestEnv{
		ctx:           t.Context(),
		queries:       queries,
		encryptionKey: encryptionKey,
		svc:           svc,
	}
}

// --- Test: ListPaymentMethods

func TestListPaymentMethods_NilProjectID_InvalidArgument(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.ListPaymentMethods(testEnv.ctx, ListPaymentMethodsRequest{
		ProjectID: uuid.Nil,
	})
	require.Nil(t, resp)
	require.ErrorIs(t, err, ErrInvalidArgument)
}
