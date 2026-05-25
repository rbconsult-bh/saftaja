package payment

import (
	"testing"

	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

func TestValidateSessionTransition(t *testing.T) {
	tests := []struct {
		name string
		from domain.PaymentIntentStatus
		to   domain.PaymentIntentStatus
		err  bool
	}{
		{"created to authenticating", domain.PaymentIntentStatusCreated, domain.PaymentIntentStatusAuthenticating, false},
		{"created to failed", domain.PaymentIntentStatusCreated, domain.PaymentIntentStatusFailed, false},
		{"created to completed", domain.PaymentIntentStatusCreated, domain.PaymentIntentStatusCompleted, true},
		{"authenticating to authenticated", domain.PaymentIntentStatusAuthenticating, domain.PaymentIntentStatusAuthenticated, false},
		{"authenticating to failed", domain.PaymentIntentStatusAuthenticating, domain.PaymentIntentStatusFailed, false},
		{"authenticated to paying", domain.PaymentIntentStatusAuthenticated, domain.PaymentIntentStatusPaying, false},
		{"paying to completed", domain.PaymentIntentStatusPaying, domain.PaymentIntentStatusCompleted, false},
		{"paying to failed", domain.PaymentIntentStatusPaying, domain.PaymentIntentStatusFailed, false},
		{"completed to anything", domain.PaymentIntentStatusCompleted, domain.PaymentIntentStatusAuthenticating, true},
		{"failed to anything", domain.PaymentIntentStatusFailed, domain.PaymentIntentStatusCreated, true},
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
	if !IsTerminalSessionStatus(domain.PaymentIntentStatusCompleted) {
		t.Error("completed should be terminal")
	}
	if !IsTerminalSessionStatus(domain.PaymentIntentStatusFailed) {
		t.Error("failed should be terminal")
	}
	if IsTerminalSessionStatus(domain.PaymentIntentStatusAuthenticating) {
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
