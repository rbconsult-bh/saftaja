package payment

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
		{"created_to_ready_to_authenticate", PaymentIntentStatusCreated, PaymentIntentStatusReadyToAuthenticate, false},
		{"created_to_failed", PaymentIntentStatusCreated, PaymentIntentStatusFailed, false},
		{"created_cannot_skip_to_capturing", PaymentIntentStatusCreated, PaymentIntentStatusCapturing, true},
		{"ready_to_authenticate_to_awaiting_authentication_result", PaymentIntentStatusReadyToAuthenticate, PaymentIntentStatusAwaitingAuthenticationResult, false},
		{"ready_to_authenticate_to_ready_to_capture", PaymentIntentStatusReadyToAuthenticate, PaymentIntentStatusReadyToCapture, false},
		{"ready_to_authenticate_to_failed", PaymentIntentStatusReadyToAuthenticate, PaymentIntentStatusFailed, false},
		{"awaiting_authentication_result_to_ready_to_capture", PaymentIntentStatusAwaitingAuthenticationResult, PaymentIntentStatusReadyToCapture, false},
		{"awaiting_authentication_result_to_failed", PaymentIntentStatusAwaitingAuthenticationResult, PaymentIntentStatusFailed, false},
		{"awaiting_authentication_result_cannot_transition_to_capturing", PaymentIntentStatusAwaitingAuthenticationResult, PaymentIntentStatusCapturing, true},
		{"ready_to_capture_to_capturing", PaymentIntentStatusReadyToCapture, PaymentIntentStatusCapturing, false},
		{"ready_to_capture_to_failed", PaymentIntentStatusReadyToCapture, PaymentIntentStatusFailed, false},
		{"capturing_to_succeeded", PaymentIntentStatusCapturing, PaymentIntentStatusSucceeded, false},
		{"capturing_to_failed", PaymentIntentStatusCapturing, PaymentIntentStatusFailed, false},
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
	assert.False(t, PaymentIntentStatusReadyToAuthenticate.IsTerminal())
	assert.False(t, PaymentIntentStatusAwaitingAuthenticationResult.IsTerminal())
}
