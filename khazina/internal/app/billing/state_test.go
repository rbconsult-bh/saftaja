package billing

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePaymentIntentTransition_Card(t *testing.T) {
	tests := []struct {
		name    string
		from    PaymentIntentStatus
		to      PaymentIntentStatus
		wantErr bool
	}{
		{"created to verifying card", PaymentIntentStatusCreated, PaymentIntentStatusVerifyingCard, false},
		{"created to failed", PaymentIntentStatusCreated, PaymentIntentStatusFailed, false},
		{"created cannot skip to processing payment", PaymentIntentStatusCreated, PaymentIntentStatusProcessingPayment, true},
		{"verifying card to card verified", PaymentIntentStatusVerifyingCard, PaymentIntentStatusCardVerified, false},
		{"verifying card to failed", PaymentIntentStatusVerifyingCard, PaymentIntentStatusFailed, false},
		{"card verified to processing payment", PaymentIntentStatusCardVerified, PaymentIntentStatusProcessingPayment, false},
		{"card verified to failed", PaymentIntentStatusCardVerified, PaymentIntentStatusFailed, false},
		{"processing payment to completed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusCompleted, false},
		{"processing payment to failed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusFailed, false},
		{"completed is terminal", PaymentIntentStatusCompleted, PaymentIntentStatusFailed, true},
		{"failed is terminal", PaymentIntentStatusFailed, PaymentIntentStatusCreated, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePaymentIntentTransition(PaymentMethodCard, tt.from, tt.to)
			if tt.wantErr {
				assert.True(t, errors.Is(err, ErrPaymentIntentInvalidTransition))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPaymentIntentStatusHelpers(t *testing.T) {
	assert.True(t, PaymentIntentStatusCompleted.IsTerminal())
	assert.True(t, PaymentIntentStatusFailed.IsTerminal())
	assert.False(t, PaymentIntentStatusVerifyingCard.IsTerminal())
}
