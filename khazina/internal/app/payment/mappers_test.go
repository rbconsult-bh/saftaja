package payment

import (
	"errors"
	"testing"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestMapStoreInvoiceStatusToInvoiceStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  store.InvoiceStatus
		want    InvoiceStatus
		wantErr bool
	}{
		{"pending", store.InvoiceStatusPending, InvoiceStatusPending, false},
		{"paid", store.InvoiceStatusPaid, InvoiceStatusPaid, false},
		{"cancelled", store.InvoiceStatusCancelled, InvoiceStatusCancelled, false},
		{"unknown", store.InvoiceStatus("wat"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapStoreInvoiceStatusToInvoiceStatus(tt.status)

			if tt.wantErr {
				assert.True(t, errors.Is(err, ErrInvoiceInvalidState))
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMapStorePaymentMethodReferenceToPaymentMethodReference(t *testing.T) {
	tests := []struct {
		name      string
		reference *string
		want      *PaymentMethodReference
	}{
		{"missing_reference", nil, nil},
		{"present_reference", ptr.To("SESSION123"), ptr.To(PaymentMethodReference("SESSION123"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapStorePaymentMethodReferenceToPaymentMethodReference(tt.reference)

			assert.Equal(t, tt.want, got)
		})
	}
}
