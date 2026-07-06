package payment

var validSessionTransitions = map[PaymentIntentStatus][]PaymentIntentStatus{
	PaymentIntentStatusCreated:           {PaymentIntentStatusVerifyingCard, PaymentIntentStatusFailed},
	PaymentIntentStatusVerifyingCard:     {PaymentIntentStatusCardVerified, PaymentIntentStatusFailed},
	PaymentIntentStatusCardVerified:      {PaymentIntentStatusProcessingPayment},
	PaymentIntentStatusProcessingPayment: {PaymentIntentStatusCompleted, PaymentIntentStatusFailed},
}

func ValidateSessionTransition(from, to PaymentIntentStatus) error {
	allowed, ok := validSessionTransitions[from]
	if !ok {
		return &InvalidStateTransitionError{Entity: EntitySession, From: string(from), To: string(to)}
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return &InvalidStateTransitionError{Entity: EntitySession, From: string(from), To: string(to)}
}

var validInvoiceTransitions = map[InvoiceStatus][]InvoiceStatus{
	InvoiceStatusPending: {InvoiceStatusPaid, InvoiceStatusFailed},
}

func ValidateInvoiceTransition(from, to InvoiceStatus) error {
	allowed, ok := validInvoiceTransitions[from]
	if !ok {
		return &InvalidStateTransitionError{Entity: EntityInvoice, From: string(from), To: string(to)}
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return &InvalidStateTransitionError{Entity: EntityInvoice, From: string(from), To: string(to)}
}

func IsTerminalSessionStatus(s PaymentIntentStatus) bool {
	return s == PaymentIntentStatusCompleted || s == PaymentIntentStatusFailed
}

func IsTerminalInvoiceStatus(s InvoiceStatus) bool {
	return s == InvoiceStatusPaid || s == InvoiceStatusFailed
}
