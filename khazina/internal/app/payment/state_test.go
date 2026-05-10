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
		{"created to completed", domain.PaymentSessionStatusCreated, domain.PaymentSessionStatusCompleted, true},
		{"authenticating to authenticated", domain.PaymentSessionStatusAuthenticating, domain.PaymentSessionStatusAuthenticated, false},
		{"authenticated to paying", domain.PaymentSessionStatusAuthenticated, domain.PaymentSessionStatusPaying, false},
		{"completed to anything", domain.PaymentSessionStatusCompleted, domain.PaymentSessionStatusAuthenticating, true},
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
		{"pending to processing", domain.InvoiceStatusPending, domain.InvoiceStatusProcessing, false},
		{"pending to failed", domain.InvoiceStatusPending, domain.InvoiceStatusFailed, false},
		{"processing to paid", domain.InvoiceStatusProcessing, domain.InvoiceStatusPaid, false},
		{"paid to processing", domain.InvoiceStatusPaid, domain.InvoiceStatusProcessing, true},
		{"failed to paid", domain.InvoiceStatusFailed, domain.InvoiceStatusPaid, true},
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
