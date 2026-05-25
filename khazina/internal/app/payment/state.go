package payment

import "github.com/rbconsult-bh/saftaja/khazina/internal/domain"

var validSessionTransitions = map[domain.PaymentIntentStatus][]domain.PaymentIntentStatus{
	domain.PaymentIntentStatusCreated:        {domain.PaymentIntentStatusAuthenticating, domain.PaymentIntentStatusFailed},
	domain.PaymentIntentStatusAuthenticating: {domain.PaymentIntentStatusAuthenticated, domain.PaymentIntentStatusFailed},
	domain.PaymentIntentStatusAuthenticated:  {domain.PaymentIntentStatusPaying},
	domain.PaymentIntentStatusPaying:         {domain.PaymentIntentStatusCompleted, domain.PaymentIntentStatusFailed},
}

func ValidateSessionTransition(from, to domain.PaymentIntentStatus) error {
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

var validInvoiceTransitions = map[domain.InvoiceStatus][]domain.InvoiceStatus{
	domain.InvoiceStatusPending: {domain.InvoiceStatusPaid, domain.InvoiceStatusFailed},
}

func ValidateInvoiceTransition(from, to domain.InvoiceStatus) error {
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

func IsTerminalSessionStatus(s domain.PaymentIntentStatus) bool {
	return s == domain.PaymentIntentStatusCompleted || s == domain.PaymentIntentStatusFailed
}

func IsTerminalInvoiceStatus(s domain.InvoiceStatus) bool {
	return s == domain.InvoiceStatusPaid || s == domain.InvoiceStatusFailed
}
