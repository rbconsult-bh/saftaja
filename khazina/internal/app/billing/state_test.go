package billing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPaymentIntentStatusCanMoveTo_Card(t *testing.T) {
	tests := []struct {
		name string
		from PaymentIntentStatus
		to   PaymentIntentStatus
		want bool
	}{
		{"created to verifying card", PaymentIntentStatusCreated, PaymentIntentStatusVerifyingCard, true},
		{"created to failed", PaymentIntentStatusCreated, PaymentIntentStatusFailed, true},
		{"created cannot skip to processing payment", PaymentIntentStatusCreated, PaymentIntentStatusProcessingPayment, false},
		{"verifying card to card verified", PaymentIntentStatusVerifyingCard, PaymentIntentStatusCardVerified, true},
		{"verifying card to failed", PaymentIntentStatusVerifyingCard, PaymentIntentStatusFailed, true},
		{"card verified to processing payment", PaymentIntentStatusCardVerified, PaymentIntentStatusProcessingPayment, true},
		{"card verified to failed", PaymentIntentStatusCardVerified, PaymentIntentStatusFailed, true},
		{"processing payment to completed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusCompleted, true},
		{"processing payment to failed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusFailed, true},
		{"completed is terminal", PaymentIntentStatusCompleted, PaymentIntentStatusFailed, false},
		{"failed is terminal", PaymentIntentStatusFailed, PaymentIntentStatusCreated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.from.CanMoveTo(PaymentMethodCard, tt.to))
		})
	}
}

func TestPaymentIntentStatusHelpers(t *testing.T) {
	assert.True(t, PaymentIntentStatusCreated.IsKnown())
	assert.True(t, PaymentIntentStatusCompleted.IsTerminal())
	assert.True(t, PaymentIntentStatusFailed.IsTerminal())
	assert.False(t, PaymentIntentStatusVerifyingCard.IsTerminal())
	assert.False(t, PaymentIntentStatus("wat").IsKnown())
}
