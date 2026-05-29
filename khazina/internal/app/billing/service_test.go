package billing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/testutil"
	"github.com/stretchr/testify/assert"
)

type billingTestEnv struct {
	ctx     context.Context
	queries store.TransactionQuerier
	svc     Service
}

func setupTestEnv(t *testing.T) billingTestEnv {
	db := testutil.SetupIsolatedDBWithFixtures(
		t,
		"./testdata/fixtures",
	)

	queries := store.NewTransactionQuerier(db.Pool)
	svc := New(queries)

	return billingTestEnv{
		ctx:     t.Context(),
		queries: queries,
		svc:     svc,
	}
}

// --- Test: GetInvoice

func TestGetInvoice_NilInvoiceID_InvalidArgument(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrInvalidArgument)
}

func TestGetInvoice_NilProjectID_InvalidArgument(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.New(),
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrInvalidArgument)
}

func TestGetInvoice_Success(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("e7543086-71e8-4ce0-abeb-db4327235a8c"),
		ProjectID: uuid.MustParse("0c48fe1f-d469-4182-8b63-415bfa59a743"),
	})
	assert.NoError(t, err)
	assert.Equal(t, uuid.MustParse("e7543086-71e8-4ce0-abeb-db4327235a8c"), resp.Invoice.ID)
	assert.Len(t, resp.Invoice.Items, 1)
}
