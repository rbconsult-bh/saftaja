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
		{"created to ready to start challenge", PaymentIntentStatusCreated, PaymentIntentStatusReadyToStartChallenge, false},
		{"created to failed", PaymentIntentStatusCreated, PaymentIntentStatusFailed, false},
		{"created to succeeded", PaymentIntentStatusCreated, PaymentIntentStatusSucceeded, true},
		{"ready to start challenge to ready to capture", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusReadyToCapture, false},
		{"ready to start challenge to failed", PaymentIntentStatusReadyToStartChallenge, PaymentIntentStatusFailed, false},
		{"ready to capture to capturing payment", PaymentIntentStatusReadyToCapture, PaymentIntentStatusCapturingPayment, false},
		{"capturing payment to succeeded", PaymentIntentStatusCapturingPayment, PaymentIntentStatusSucceeded, false},
		{"capturing payment to failed", PaymentIntentStatusCapturingPayment, PaymentIntentStatusFailed, false},
		{"succeeded to anything", PaymentIntentStatusSucceeded, PaymentIntentStatusReadyToStartChallenge, true},
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
	if !IsTerminalSessionStatus(PaymentIntentStatusSucceeded) {
		t.Error("succeeded should be terminal")
	}
	if !IsTerminalSessionStatus(PaymentIntentStatusFailed) {
		t.Error("failed should be terminal")
	}
	if IsTerminalSessionStatus(PaymentIntentStatusReadyToStartChallenge) {
		t.Error("ready to start challenge should not be terminal")
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
