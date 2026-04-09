package payment_test

import (
	"testing"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
)

func TestValidateSessionTransition_Valid(t *testing.T) {
	tests := []struct {
		name string
		from domain.PaymentSessionStatus
		to   domain.PaymentSessionStatus
	}{
		{"created to authenticating", domain.PaymentSessionStatusCreated, domain.PaymentSessionStatusAuthenticating},
		{"authenticating to authenticated", domain.PaymentSessionStatusAuthenticating, domain.PaymentSessionStatusAuthenticated},
		{"authenticating to failed", domain.PaymentSessionStatusAuthenticating, domain.PaymentSessionStatusFailed},
		{"authenticated to paying", domain.PaymentSessionStatusAuthenticated, domain.PaymentSessionStatusPaying},
		{"paying to completed", domain.PaymentSessionStatusPaying, domain.PaymentSessionStatusCompleted},
		{"paying to failed", domain.PaymentSessionStatusPaying, domain.PaymentSessionStatusFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := payment.ValidateSessionTransition(tc.from, tc.to)
			if err != nil {
				t.Errorf("expected valid transition %s -> %s, got error: %v", tc.from, tc.to, err)
			}
		})
	}
}

func TestValidateSessionTransition_Invalid(t *testing.T) {
	tests := []struct {
		name string
		from domain.PaymentSessionStatus
		to   domain.PaymentSessionStatus
	}{
		{"created to completed (skip)", domain.PaymentSessionStatusCreated, domain.PaymentSessionStatusCompleted},
		{"created to paying (skip)", domain.PaymentSessionStatusCreated, domain.PaymentSessionStatusPaying},
		{"completed to paying (terminal)", domain.PaymentSessionStatusCompleted, domain.PaymentSessionStatusPaying},
		{"failed to created (terminal)", domain.PaymentSessionStatusFailed, domain.PaymentSessionStatusCreated},
		{"authenticating to paying (skip)", domain.PaymentSessionStatusAuthenticating, domain.PaymentSessionStatusPaying},
		{"authenticated to completed (skip)", domain.PaymentSessionStatusAuthenticated, domain.PaymentSessionStatusCompleted},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := payment.ValidateSessionTransition(tc.from, tc.to)
			if err == nil {
				t.Errorf("expected invalid transition %s -> %s, got nil error", tc.from, tc.to)
			}
			var transitionErr *payment.InvalidStateTransitionError
			if _, ok := err.(*payment.InvalidStateTransitionError); !ok {
				t.Errorf("expected InvalidStateTransitionError, got %T", transitionErr)
			}
		})
	}
}

func TestValidateInvoiceTransition_Valid(t *testing.T) {
	tests := []struct {
		name string
		from domain.InvoiceStatus
		to   domain.InvoiceStatus
	}{
		{"pending to processing", domain.InvoiceStatusPending, domain.InvoiceStatusProcessing},
		{"pending to failed", domain.InvoiceStatusPending, domain.InvoiceStatusFailed},
		{"processing to paid", domain.InvoiceStatusProcessing, domain.InvoiceStatusPaid},
		{"processing to failed", domain.InvoiceStatusProcessing, domain.InvoiceStatusFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := payment.ValidateInvoiceTransition(tc.from, tc.to)
			if err != nil {
				t.Errorf("expected valid transition %s -> %s, got error: %v", tc.from, tc.to, err)
			}
		})
	}
}

func TestValidateInvoiceTransition_Invalid(t *testing.T) {
	tests := []struct {
		name string
		from domain.InvoiceStatus
		to   domain.InvoiceStatus
	}{
		{"pending to paid (skip)", domain.InvoiceStatusPending, domain.InvoiceStatusPaid},
		{"paid to pending (terminal)", domain.InvoiceStatusPaid, domain.InvoiceStatusPending},
		{"failed to processing (terminal)", domain.InvoiceStatusFailed, domain.InvoiceStatusProcessing},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := payment.ValidateInvoiceTransition(tc.from, tc.to)
			if err == nil {
				t.Errorf("expected invalid transition %s -> %s, got nil error", tc.from, tc.to)
			}
		})
	}
}

func TestIsTerminalSessionStatus(t *testing.T) {
	tests := []struct {
		status   domain.PaymentSessionStatus
		terminal bool
	}{
		{domain.PaymentSessionStatusCreated, false},
		{domain.PaymentSessionStatusAuthenticating, false},
		{domain.PaymentSessionStatusAuthenticated, false},
		{domain.PaymentSessionStatusPaying, false},
		{domain.PaymentSessionStatusCompleted, true},
		{domain.PaymentSessionStatusFailed, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			got := payment.IsTerminalSessionStatus(tc.status)
			if got != tc.terminal {
				t.Errorf("IsTerminalSessionStatus(%s) = %v, want %v", tc.status, got, tc.terminal)
			}
		})
	}
}

func TestIsTerminalInvoiceStatus(t *testing.T) {
	tests := []struct {
		status   domain.InvoiceStatus
		terminal bool
	}{
		{domain.InvoiceStatusPending, false},
		{domain.InvoiceStatusProcessing, false},
		{domain.InvoiceStatusPaid, true},
		{domain.InvoiceStatusFailed, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			got := payment.IsTerminalInvoiceStatus(tc.status)
			if got != tc.terminal {
				t.Errorf("IsTerminalInvoiceStatus(%s) = %v, want %v", tc.status, got, tc.terminal)
			}
		})
	}
}
