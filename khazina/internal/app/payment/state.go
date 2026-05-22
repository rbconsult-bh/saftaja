package payment

import "github.com/rbconsult-bh/saftaja/khazina/internal/domain"

var validSessionTransitions = map[domain.PaymentSessionStatus][]domain.PaymentSessionStatus{
	domain.PaymentSessionStatusCreated:        {domain.PaymentSessionStatusAuthenticating, domain.PaymentSessionStatusFailed},
	domain.PaymentSessionStatusAuthenticating: {domain.PaymentSessionStatusAuthenticated, domain.PaymentSessionStatusFailed},
	domain.PaymentSessionStatusAuthenticated:  {domain.PaymentSessionStatusPaying},
	domain.PaymentSessionStatusPaying:         {domain.PaymentSessionStatusCompleted, domain.PaymentSessionStatusFailed},
}

func ValidateSessionTransition(from, to domain.PaymentSessionStatus) error {
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

func IsTerminalSessionStatus(s domain.PaymentSessionStatus) bool {
	return s == domain.PaymentSessionStatusCompleted || s == domain.PaymentSessionStatusFailed
}

func IsTerminalInvoiceStatus(s domain.InvoiceStatus) bool {
	return s == domain.InvoiceStatusPaid || s == domain.InvoiceStatusFailed
}
