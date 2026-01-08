package payment

import (
	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/internal/domain"
)

type CheckoutData struct {
	Invoice        InvoiceInfo
	Items          []ItemInfo
	PaymentOptions []PaymentOption
	MPGSConfig     *MPGSConfig
	IsPaid         bool
}

type InvoiceInfo struct {
	ID            uuid.UUID
	Amount        string
	Currency      string
	Description   string
	CustomerEmail string
	CustomerName  string
}

type ItemInfo struct {
	Name      string
	Quantity  int32
	UnitPrice string
	Amount    string
}

type PaymentOption struct {
	GatewayAccountID uuid.UUID
	Method           domain.PaymentMethod
	Label            string
}

type MPGSConfig struct {
	BaseURL    string
	MerchantID string
	APIVersion string
}

type InitiateSessionRequest struct {
	InvoiceID        uuid.UUID
	GatewayAccountID uuid.UUID
	PaymentMethod    domain.PaymentMethod
	PayerIP          string
	PayerUserAgent   string
}

type InitiateSessionResult struct {
	PaymentSessionID uuid.UUID
	GatewaySessionID string
}

type InitiateAuthRequest struct {
	InvoiceID        uuid.UUID
	PaymentSessionID uuid.UUID
}

type InitiateAuthResult struct {
	TransactionID string
	NextStep      string
}

type BrowserDetails struct {
	AcceptHeaders string `json:"acceptHeaders,omitempty"`
	ColorDepth    int    `json:"colorDepth"`
	JavaEnabled   bool   `json:"javaEnabled"`
	Language      string `json:"language"`
	ScreenHeight  int    `json:"screenHeight"`
	ScreenWidth   int    `json:"screenWidth"`
	TimeZone      int    `json:"timeZone"`
}

type ProcessAuthRequest struct {
	InvoiceID        uuid.UUID
	PaymentSessionID uuid.UUID
	BrowserDetails   BrowserDetails
	PayerIP          string
	UserAgent        string
	AcceptHeaders    string
}

type ProcessAuthResult struct {
	NextStep     string
	RedirectHTML string
}

type FinalizePaymentRequest struct {
	InvoiceID        uuid.UUID
	PaymentSessionID uuid.UUID
}

type PaymentResultCode string

const (
	ResultSuccess      PaymentResultCode = "PAYMENT_SUCCESS"
	ResultDeclined     PaymentResultCode = "PAYMENT_DECLINED"
	ResultAuthFailed   PaymentResultCode = "AUTH_FAILED"
	ResultGatewayError PaymentResultCode = "GATEWAY_ERROR"
)

type FinalizePaymentResult struct {
	Success     bool
	ResultCode  PaymentResultCode
	CheckoutURL string
}
