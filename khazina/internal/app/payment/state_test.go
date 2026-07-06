package payment

import (
	"testing"
)

func TestValidateSessionTransition(t *testing.T) {
	tests := []struct {
		name string
		from PaymentIntentStatus
		to   PaymentIntentStatus
		err  bool
	}{
		{"created to verifying card", PaymentIntentStatusCreated, PaymentIntentStatusVerifyingCard, false},
		{"created to failed", PaymentIntentStatusCreated, PaymentIntentStatusFailed, false},
		{"created to completed", PaymentIntentStatusCreated, PaymentIntentStatusCompleted, true},
		{"verifying card to ready to capture", PaymentIntentStatusVerifyingCard, PaymentIntentStatusReadyToCapture, false},
		{"verifying card to failed", PaymentIntentStatusVerifyingCard, PaymentIntentStatusFailed, false},
		{"ready to capture to processing payment", PaymentIntentStatusReadyToCapture, PaymentIntentStatusProcessingPayment, false},
		{"processing payment to completed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusCompleted, false},
		{"processing payment to failed", PaymentIntentStatusProcessingPayment, PaymentIntentStatusFailed, false},
		{"completed to anything", PaymentIntentStatusCompleted, PaymentIntentStatusVerifyingCard, true},
		{"failed to anything", PaymentIntentStatusFailed, PaymentIntentStatusCreated, true},
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
		from InvoiceStatus
		to   InvoiceStatus
		err  bool
	}{
		{"pending to paid", InvoiceStatusPending, InvoiceStatusPaid, false},
		{"pending to failed", InvoiceStatusPending, InvoiceStatusFailed, false},
		{"paid to anything", InvoiceStatusPaid, InvoiceStatusPending, true},
		{"failed to anything", InvoiceStatusFailed, InvoiceStatusPaid, true},
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
	if !IsTerminalSessionStatus(PaymentIntentStatusCompleted) {
		t.Error("completed should be terminal")
	}
	if !IsTerminalSessionStatus(PaymentIntentStatusFailed) {
		t.Error("failed should be terminal")
	}
	if IsTerminalSessionStatus(PaymentIntentStatusVerifyingCard) {
		t.Error("verifying card should not be terminal")
	}
}

func TestIsTerminalInvoiceStatus(t *testing.T) {
	if !IsTerminalInvoiceStatus(InvoiceStatusPaid) {
		t.Error("paid should be terminal")
	}
	if !IsTerminalInvoiceStatus(InvoiceStatusFailed) {
		t.Error("failed should be terminal")
	}
	if IsTerminalInvoiceStatus(InvoiceStatusPending) {
		t.Error("pending should not be terminal")
	}
}
