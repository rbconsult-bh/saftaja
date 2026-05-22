package payment

import (
	"testing"

	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

func TestValidateSessionTransition(t *testing.T) {
	tests := []struct {
		name string
		from domain.PaymentSessionStatus
		to   domain.PaymentSessionStatus
		err  bool
	}{
		{"created to authenticating", domain.PaymentSessionStatusCreated, domain.PaymentSessionStatusAuthenticating, false},
		{"created to failed", domain.PaymentSessionStatusCreated, domain.PaymentSessionStatusFailed, false},
		{"created to completed", domain.PaymentSessionStatusCreated, domain.PaymentSessionStatusCompleted, true},
		{"authenticating to authenticated", domain.PaymentSessionStatusAuthenticating, domain.PaymentSessionStatusAuthenticated, false},
		{"authenticating to failed", domain.PaymentSessionStatusAuthenticating, domain.PaymentSessionStatusFailed, false},
		{"authenticated to paying", domain.PaymentSessionStatusAuthenticated, domain.PaymentSessionStatusPaying, false},
		{"paying to completed", domain.PaymentSessionStatusPaying, domain.PaymentSessionStatusCompleted, false},
		{"paying to failed", domain.PaymentSessionStatusPaying, domain.PaymentSessionStatusFailed, false},
		{"completed to anything", domain.PaymentSessionStatusCompleted, domain.PaymentSessionStatusAuthenticating, true},
		{"failed to anything", domain.PaymentSessionStatusFailed, domain.PaymentSessionStatusCreated, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionTransition(tt.from, tt.to)
			if (err != nil) != tt.err {
				t.Errorf("ValidateSessionTransition(%s, %s) error = %v, wantError %v", tt.from, tt.to, err, tt.err)
			}
		})
	}
}

func TestValidateInvoiceTransition(t *testing.T) {
	tests := []struct {
		name string
		from domain.InvoiceStatus
		to   domain.InvoiceStatus
		err  bool
	}{
		{"pending to paid", domain.InvoiceStatusPending, domain.InvoiceStatusPaid, false},
		{"pending to failed", domain.InvoiceStatusPending, domain.InvoiceStatusFailed, false},
		{"paid to anything", domain.InvoiceStatusPaid, domain.InvoiceStatusPending, true},
		{"failed to anything", domain.InvoiceStatusFailed, domain.InvoiceStatusPaid, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInvoiceTransition(tt.from, tt.to)
			if (err != nil) != tt.err {
				t.Errorf("ValidateInvoiceTransition(%s, %s) error = %v, wantError %v", tt.from, tt.to, err, tt.err)
			}
		})
	}
}

func TestIsTerminalSessionStatus(t *testing.T) {
	if !IsTerminalSessionStatus(domain.PaymentSessionStatusCompleted) {
		t.Error("completed should be terminal")
	}
	if !IsTerminalSessionStatus(domain.PaymentSessionStatusFailed) {
		t.Error("failed should be terminal")
	}
	if IsTerminalSessionStatus(domain.PaymentSessionStatusAuthenticating) {
		t.Error("authenticating should not be terminal")
	}
}

func TestIsTerminalInvoiceStatus(t *testing.T) {
	if !IsTerminalInvoiceStatus(domain.InvoiceStatusPaid) {
		t.Error("paid should be terminal")
	}
	if !IsTerminalInvoiceStatus(domain.InvoiceStatusFailed) {
		t.Error("failed should be terminal")
	}
	if IsTerminalInvoiceStatus(domain.InvoiceStatusPending) {
		t.Error("pending should not be terminal")
	}
}
