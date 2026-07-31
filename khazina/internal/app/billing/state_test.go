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
		{"created_to_ready_to_start_challenge", PaymentIntentStatusCreated, PaymentIntentStatusReadyToStartChallenge, false},
		{"created_to_failed", PaymentIntentStatusCreated, PaymentIntentStatusFailed, false},
		{"created_cannot_skip_to_capturing_payment", PaymentIntentStatusCreated, PaymentIntentStatusCapturingPayment, true},
		{"ready_to_start_challenge_to_awaiting_challenge_completion", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusAwaitingChallengeCompletion, false},
		{"ready_to_start_challenge_to_ready_to_capture", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusReadyToCapture, false},
		{"ready_to_start_challenge_to_failed", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusFailed, false},
		{"awaiting_challenge_completion_to_ready_to_capture", PaymentIntentStatusAwaitingChallengeCompletion, PaymentIntentStatusReadyToCapture, false},
		{"awaiting_challenge_completion_to_failed", PaymentIntentStatusAwaitingChallengeCompletion, PaymentIntentStatusFailed, false},
		{"awaiting_challenge_completion_cannot_capture_payment", PaymentIntentStatusAwaitingChallengeCompletion, PaymentIntentStatusCapturingPayment, true},
		{"ready_to_capture_to_capturing_payment", PaymentIntentStatusReadyToCapture, PaymentIntentStatusCapturingPayment, false},
		{"ready_to_capture_to_failed", PaymentIntentStatusReadyToCapture, PaymentIntentStatusFailed, false},
		{"capturing_payment_to_succeeded", PaymentIntentStatusCapturingPayment, PaymentIntentStatusSucceeded, false},
		{"capturing_payment_to_failed", PaymentIntentStatusCapturingPayment, PaymentIntentStatusFailed, false},
		{"succeeded_is_terminal", PaymentIntentStatusSucceeded, PaymentIntentStatusFailed, true},
		{"failed_is_terminal", PaymentIntentStatusFailed, PaymentIntentStatusCreated, true},
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
