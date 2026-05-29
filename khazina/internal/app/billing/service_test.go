package billing

import (
	"testing"

	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetInvoice_NilInvoiceID_InvalidArgument(t *testing.T) {
	ctx := t.Context()

	dbPool := testutil.SetupIsolatedDB(t)
	svc := New(store.NewTransactionQuerier(dbPool))

	resp, err := svc.GetInvoice(ctx, GetInvoiceRequest{})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrInvalidArgument)
}
