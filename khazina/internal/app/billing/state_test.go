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
		{"verifying card to challenging card", PaymentIntentStatusVerifyingCard, PaymentIntentStatusChallengingCard, false},
		{"verifying card to ready to capture", PaymentIntentStatusVerifyingCard, PaymentIntentStatusReadyToCapture, false},
		{"verifying card to failed", PaymentIntentStatusVerifyingCard, PaymentIntentStatusFailed, false},
		{"challenging card to ready to capture", PaymentIntentStatusChallengingCard, PaymentIntentStatusReadyToCapture, false},
		{"challenging card to failed", PaymentIntentStatusChallengingCard, PaymentIntentStatusFailed, false},
		{"challenging card cannot capture payment", PaymentIntentStatusChallengingCard, PaymentIntentStatusProcessingPayment, true},
		{"ready to capture to processing payment", PaymentIntentStatusReadyToCapture, PaymentIntentStatusProcessingPayment, false},
		{"ready to capture to failed", PaymentIntentStatusReadyToCapture, PaymentIntentStatusFailed, false},
		{"processing payment to completed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusCompleted, false},
		{"processing payment to failed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusFailed, false},
		{"completed is terminal", PaymentIntentStatusCompleted, PaymentIntentStatusFailed, true},
		{"failed is terminal", PaymentIntentStatusFailed, PaymentIntentStatusCreated, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.from.ValidatePaymentIntentTransition(PaymentMethodCard, tt.to)
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
	assert.False(t, PaymentIntentStatusChallengingCard.IsTerminal())
}
