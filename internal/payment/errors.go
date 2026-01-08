package payment

import "fmt"

type EntityType string

const (
	EntitySession EntityType = "session"
	EntityInvoice EntityType = "invoice"
)

type InvalidStateTransitionError struct {
	Entity EntityType
	From   string
	To     string
}

func (e *InvalidStateTransitionError) Error() string {
	return fmt.Sprintf("invalid %s state transition: %s -> %s", e.Entity, e.From, e.To)
}

type SessionExpiredError struct {
	SessionID string
}

func (e *SessionExpiredError) Error() string {
	return fmt.Sprintf("payment session %s has expired", e.SessionID)
}

type InvoiceAlreadyPaidError struct {
	InvoiceID string
}

func (e *InvoiceAlreadyPaidError) Error() string {
	return fmt.Sprintf("invoice %s is already paid", e.InvoiceID)
}

type SessionInvoiceMismatchError struct {
	SessionID string
	InvoiceID string
}

func (e *SessionInvoiceMismatchError) Error() string {
	return fmt.Sprintf("session %s does not belong to invoice %s", e.SessionID, e.InvoiceID)
}

type GatewayError struct {
	Gateway string
	Err     error
}

func (e *GatewayError) Error() string {
	return fmt.Sprintf("%s gateway error: %v", e.Gateway, e.Err)
}

func (e *GatewayError) Unwrap() error {
	return e.Err
}
