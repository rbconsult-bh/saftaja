package billing

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

func (s PaymentIntentStatus) CanMoveTo(pm PaymentMethod, next PaymentIntentStatus) bool {
	switch pm {
	case PaymentMethodCard:
		return cardPaymentIntentTransitions[s][next]
	case PaymentMethodApplePay:
		// TODO: use applePayPaymentIntentTransitions
		fallthrough
	default:
		return false
	}
}

func (s PaymentIntentStatus) IsTerminal() bool {
	return s == PaymentIntentStatusCompleted || s == PaymentIntentStatusFailed
}
