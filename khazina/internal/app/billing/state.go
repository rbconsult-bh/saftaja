package billing

import "fmt"

var cardPaymentIntentTransitions = map[PaymentIntentStatus]map[PaymentIntentStatus]bool{
	PaymentIntentStatusCreated: {
		PaymentIntentStatusVerifyingCard: true,
		PaymentIntentStatusFailed:        true,
	},
	PaymentIntentStatusVerifyingCard: {
		PaymentIntentStatusCardVerified: true,
		PaymentIntentStatusFailed:       true,
	},
	PaymentIntentStatusCardVerified: {
		PaymentIntentStatusProcessingPayment: true,
		PaymentIntentStatusFailed:            true,
	},
	PaymentIntentStatusProcessingPayment: {
		PaymentIntentStatusCompleted: true,
		PaymentIntentStatusFailed:    true,
	},
	PaymentIntentStatusCompleted: {},
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
	return s == PaymentIntentStatusCompleted || s == PaymentIntentStatusFailed
}
