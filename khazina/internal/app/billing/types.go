package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidArgument           = errors.New("invalid argument")
	ErrNotFound                  = errors.New("not found")
	ErrUnsupportedPaymentMethod  = errors.New("unsupported payment method")
	ErrInvoiceAlreadyPaid        = errors.New("invoice already paid")
	ErrInvoiceCancelled          = errors.New("invoice cancelled")
	ErrInvoiceNotFound           = errors.New("invoice not found")
	ErrGatewayAccountNotFound    = errors.New("gateway account not found")
	ErrIdempotencyMismatch       = errors.New("idempotency key already used with different parameters")
	ErrPaymentIntentExpired      = errors.New("payment intent expired")
	ErrPaymentIntentInvalidState = errors.New("payment intent invalid state")
	ErrUnsupportedGateway        = errors.New("unsupported gateway")
)

type Service interface {
	GetInvoice(ctx context.Context, r GetInvoiceRequest) (*GetInvoiceResponse, error)

	StartPayment(ctx context.Context, r StartPaymentRequest) (*StartPaymentResponse, error)
	VerifyCard(ctx context.Context, r VerifyCardRequest) (*VerifyCardResponse, error)
	ChallengeCard(ctx context.Context, r ChallengeCardRequest) (*ChallengeCardResponse, error)
	CapturePayment(ctx context.Context, r CapturePaymentRequest) (*CapturePaymentResponse, error)
}

type PaymentIntentStatus string

const (
	PaymentIntentStatusCreated           PaymentIntentStatus = "created"
	PaymentIntentStatusVerifyingCard     PaymentIntentStatus = "verifying_card"
	PaymentIntentStatusCardVerified      PaymentIntentStatus = "card_verified"
	PaymentIntentStatusProcessingPayment PaymentIntentStatus = "processing_payment"
	PaymentIntentStatusCompleted         PaymentIntentStatus = "completed"
	PaymentIntentStatusFailed            PaymentIntentStatus = "failed"
)

type PaymentMethod string

const (
	PaymentMethodCard     PaymentMethod = "card"
	PaymentMethodApplePay PaymentMethod = "apple_pay"
)

type InvoiceStatus string

const (
	InvoiceStatusUnknown   InvoiceStatus = "unknown"
	InvoiceStatusPending   InvoiceStatus = "pending"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

type (
	Invoice struct {
		ID            uuid.UUID
		ProjectID     uuid.UUID
		Amount        decimal.Decimal
		Currency      string
		Status        InvoiceStatus
		ExternalID    string
		CustomerEmail string
		CustomerName  string
		Description   string
		PaidAt        *time.Time
		CreatedAt     time.Time
		UpdatedAt     time.Time
		DeletedAt     *time.Time
		Items         []InvoiceItem
	}
	InvoiceItem struct {
		ID          uuid.UUID
		InvoiceID   uuid.UUID
		Name        string
		Description string
		Quantity    int32
		UnitPrice   decimal.Decimal
		Amount      decimal.Decimal
		CreatedAt   time.Time
	}
)

type (
	GetInvoiceRequest struct {
		InvoiceID uuid.UUID
		ProjectID uuid.UUID
	}
	GetInvoiceResponse struct {
		Invoice Invoice
	}
)

func (r *GetInvoiceRequest) Validate() error {
	if r.InvoiceID == uuid.Nil {
		return fmt.Errorf("%w: InvoiceID is required", ErrInvalidArgument)
	}
	if r.ProjectID == uuid.Nil {
		return fmt.Errorf("%w: ProjectID is required", ErrInvalidArgument)
	}

	return nil
}

type (
	StartPaymentRequest struct {
		InvoiceID        uuid.UUID
		ProjectID        uuid.UUID
		IdempotencyKey   string
		GatewayAccountID uuid.UUID
		PaymentMethod    PaymentMethod
		PayerIP          string
		PayerUserAgent   string
	}
	StartPaymentResponse struct {
		PaymentIntentID  uuid.UUID
		GatewaySessionID *string
	}
)

func (r *StartPaymentRequest) Validate() error {
	if r.InvoiceID == uuid.Nil {
		return fmt.Errorf("%w: InvoiceID is required", ErrInvalidArgument)
	}
	if r.ProjectID == uuid.Nil {
		return fmt.Errorf("%w: ProjectID is required", ErrInvalidArgument)
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("%w: IdempotencyKey is required", ErrInvalidArgument)
	}
	if r.GatewayAccountID == uuid.Nil {
		return fmt.Errorf("%w: GatewayAccountID is required", ErrInvalidArgument)
	}
	switch r.PaymentMethod {
	case PaymentMethodCard, PaymentMethodApplePay:
	default:
		return fmt.Errorf("%w: unsupported PaymentMethod '%s'", ErrInvalidArgument, r.PaymentMethod)
	}
	if r.PayerIP == "" {
		return fmt.Errorf("%w: PayerIP is required", ErrInvalidArgument)
	}
	if r.PayerUserAgent == "" {
		return fmt.Errorf("%w: PayerUserAgent is required", ErrInvalidArgument)
	}

	return nil
}

type (
	VerifyCardRequest struct {
		ProjectID       uuid.UUID
		InvoiceID       uuid.UUID
		PaymentIntentID uuid.UUID
	}
	VerifyCardResponse struct{}
)

func (r *VerifyCardRequest) Validate() error {
	if r.ProjectID == uuid.Nil {
		return fmt.Errorf("%w: ProjectID is required", ErrInvalidArgument)
	}
	if r.InvoiceID == uuid.Nil {
		return fmt.Errorf("%w: InvoiceID is required", ErrInvalidArgument)
	}
	if r.PaymentIntentID == uuid.Nil {
		return fmt.Errorf("%w: PaymentIntentID is required", ErrInvalidArgument)
	}

	return nil
}

type (
	ChallengeCardRequest  struct{}
	ChallengeCardResponse struct{}
)

type (
	CapturePaymentRequest  struct{}
	CapturePaymentResponse struct{}
)
