package payment

import (
	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type PaymentMethod = store.PaymentMethod
type InvoiceStatus = store.InvoiceStatus
type PaymentIntentStatus = store.PaymentIntentStatus
type TransactionType = store.TransactionType
type TransactionStatus = store.TransactionStatus

const (
	PaymentMethodCard     = store.PaymentMethodCard
	PaymentMethodApplePay = store.PaymentMethodApplePay
	InvoiceStatusPending  = store.InvoiceStatusPending
	InvoiceStatusPaid     = store.InvoiceStatusPaid
	InvoiceStatusFailed   = store.InvoiceStatusFailed
	PaymentIntentStatusCreated        = store.PaymentIntentStatusCreated
	PaymentIntentStatusAuthenticating = store.PaymentIntentStatusAuthenticating
	PaymentIntentStatusAuthenticated  = store.PaymentIntentStatusAuthenticated
	PaymentIntentStatusPaying         = store.PaymentIntentStatusPaying
	PaymentIntentStatusCompleted      = store.PaymentIntentStatusCompleted
	PaymentIntentStatusFailed         = store.PaymentIntentStatusFailed
	TransactionTypeInitiateAuth       = store.TransactionTypeInitiateAuth
	TransactionTypeAuthenticatePayer  = store.TransactionTypeAuthenticatePayer
	TransactionTypePay                = store.TransactionTypePay
	TransactionStatusPending          = store.TransactionStatusPending
	TransactionStatusSuccess          = store.TransactionStatusSuccess
	TransactionStatusFailed           = store.TransactionStatusFailed
)

type InitiateSessionRequest struct {
	ProjectID        uuid.UUID
	InvoiceID        uuid.UUID
	GatewayAccountID uuid.UUID
	PaymentMethod    PaymentMethod
	PayerIP          string
	PayerUserAgent   string
	IdempotencyKey   string
}

type InitiateSessionResult struct {
	PaymentIntentID  uuid.UUID
	GatewaySessionID string
}

type InitiateAuthRequest struct {
	ProjectID       uuid.UUID
	InvoiceID       uuid.UUID
	PaymentIntentID uuid.UUID
}

type InitiateAuthResult struct {
	TransactionID string
	NextStep      string
}

type BrowserDetails struct {
	ThreeDSecureChallengeWindowSize string `json:"3DSecureChallengeWindowSize,omitempty"`
	AcceptHeaders                   string `json:"acceptHeaders,omitempty"`
	ColorDepth                      int    `json:"colorDepth"`
	JavaEnabled                     bool   `json:"javaEnabled"`
	Language                        string `json:"language"`
	ScreenHeight                    int    `json:"screenHeight"`
	ScreenWidth                     int    `json:"screenWidth"`
	TimeZone                        int    `json:"timeZone"`
}

type ProcessAuthRequest struct {
	ProjectID       uuid.UUID
	InvoiceID       uuid.UUID
	PaymentIntentID uuid.UUID
	BrowserDetails  BrowserDetails
	PayerIP         string
	UserAgent       string
	AcceptHeaders   string
}

type ProcessAuthResult struct {
	NextStep     string
	RedirectHTML string
}

type FinalizePaymentRequest struct {
	ProjectID       uuid.UUID
	InvoiceID       uuid.UUID
	PaymentIntentID uuid.UUID
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
