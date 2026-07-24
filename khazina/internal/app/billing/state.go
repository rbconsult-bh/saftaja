package billing

import "fmt"

var cardPaymentIntentTransitions = map[PaymentIntentStatus]map[PaymentIntentStatus]bool{
	PaymentIntentStatusCreated: {
		PaymentIntentStatusReadyToStartChallenge: true,
		PaymentIntentStatusFailed:                true,
	},
	PaymentIntentStatusReadyToStartChallenge: {
		PaymentIntentStatusAwaitingChallengeCompletion: true,
		PaymentIntentStatusReadyToCapture:              true,
		PaymentIntentStatusFailed:                      true,
	},
	PaymentIntentStatusAwaitingChallengeCompletion: {
		PaymentIntentStatusReadyToCapture: true,
		PaymentIntentStatusFailed:         true,
	},
	PaymentIntentStatusReadyToCapture: {
		PaymentIntentStatusCapturingPayment: true,
		PaymentIntentStatusFailed:           true,
	},
	PaymentIntentStatusCapturingPayment: {
		PaymentIntentStatusSucceeded: true,
		PaymentIntentStatusFailed:    true,
	},
	PaymentIntentStatusSucceeded: {},
	PaymentIntentStatusFailed:    {},
}

func (from PaymentIntentStatus) ValidatePaymentIntentTransition(pm PaymentMethod, to PaymentIntentStatus) error {
	switch pm {
	case PaymentMethodCard:
		if cardPaymentIntentTransitions[from][to] {
			return nil
		}
	case PaymentMethodApplePay:
		// TODO: use applePayPaymentIntentTransitions
	}

	return fmt.Errorf(
		"%w: cannot move payment intent from %s to %s for %s",
		ErrPaymentIntentInvalidTransition,
		from,
		to,
		pm,
	)
}

func (s PaymentIntentStatus) IsTerminal() bool {
	return s == PaymentIntentStatusSucceeded || s == PaymentIntentStatusFailed
}
