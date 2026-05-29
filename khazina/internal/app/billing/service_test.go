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
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000001000"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
	})
	assert.NoError(t, err)
	assert.Equal(t, uuid.MustParse("00000000-0000-0000-0000-000000001000"), resp.Invoice.ID)
	assert.Len(t, resp.Invoice.Items, 1)
}

func TestGetInvoice_NotFound_WhenProjectIDWrong(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000001000"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000200"),
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetInvoice_NotFound_WhenInvoiceIDNotExists(t *testing.T) {
	testEnv := setupTestEnv(t)

	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000002222"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetInvoice_NotFound_WhenNoItems(t *testing.T) {
	testEnv := setupTestEnv(t)

	// Invoice 001001 exists but has zero invoice_items — JOIN drops it.
	resp, err := testEnv.svc.GetInvoice(testEnv.ctx, GetInvoiceRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000001001"),
		ProjectID: uuid.MustParse("00000000-0000-0000-0000-000000000100"),
	})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrNotFound)
}
