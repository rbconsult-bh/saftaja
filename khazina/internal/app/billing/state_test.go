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
		{"created to ready to start challenge", PaymentIntentStatusCreated, PaymentIntentStatusReadyToStartChallenge, false},
		{"created to failed", PaymentIntentStatusCreated, PaymentIntentStatusFailed, false},
		{"created cannot skip to capturing payment", PaymentIntentStatusCreated, PaymentIntentStatusCapturingPayment, true},
		{"ready to start challenge to awaiting challenge completion", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusAwaitingChallengeCompletion, false},
		{"ready to start challenge to ready to capture", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusReadyToCapture, false},
		{"ready to start challenge to failed", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusFailed, false},
		{"awaiting challenge completion to ready to capture", PaymentIntentStatusAwaitingChallengeCompletion, PaymentIntentStatusReadyToCapture, false},
		{"awaiting challenge completion to failed", PaymentIntentStatusAwaitingChallengeCompletion, PaymentIntentStatusFailed, false},
		{"awaiting challenge completion cannot capture payment", PaymentIntentStatusAwaitingChallengeCompletion, PaymentIntentStatusCapturingPayment, true},
		{"ready to capture to capturing payment", PaymentIntentStatusReadyToCapture, PaymentIntentStatusCapturingPayment, false},
		{"ready to capture to failed", PaymentIntentStatusReadyToCapture, PaymentIntentStatusFailed, false},
		{"capturing payment to succeeded", PaymentIntentStatusCapturingPayment, PaymentIntentStatusSucceeded, false},
		{"capturing payment to failed", PaymentIntentStatusCapturingPayment, PaymentIntentStatusFailed, false},
		{"succeeded is terminal", PaymentIntentStatusSucceeded, PaymentIntentStatusFailed, true},
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
	assert.True(t, PaymentIntentStatusSucceeded.IsTerminal())
	assert.True(t, PaymentIntentStatusFailed.IsTerminal())
	assert.False(t, PaymentIntentStatusReadyToStartChallenge.IsTerminal())
	assert.False(t, PaymentIntentStatusAwaitingChallengeCompletion.IsTerminal())
}
