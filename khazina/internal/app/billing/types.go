package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var ErrInvalidArgument = errors.New("invalid argument")

type Service interface {
	GetInvoice(ctx context.Context, r GetInvoiceRequest) (*GetInvoiceResponse, error)
	ListPaymentOptions(ctx context.Context, r ListPaymentOptionsRequest) (*ListPaymentOptionsResponse, error)

	StartPayment(ctx context.Context, r StartPaymentRequest) (*StartPaymentResponse, error)
	VerifyCard(ctx context.Context, r VerifyCardRequest) (*VerifyCardResponse, error)
	ChallengeCard(ctx context.Context, r ChallengeCardRequest) (*ChallengeCardResponse, error)
	CapturePayment(ctx context.Context, r CapturePaymentRequest) (*CapturePaymentResponse, error)
}

type InvoiceStatus string

const (
	InvoiceStatusUnkown  InvoiceStatus = "unknown"
	InvoiceStatusPending InvoiceStatus = "pending"
	InvoiceStatusPaid    InvoiceStatus = "paid"
	InvoiceStatusFailed  InvoiceStatus = "failed"
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
	PaymentOption struct{}
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
	ListPaymentOptionsRequest  struct{}
	ListPaymentOptionsResponse struct{}
)

type (
	StartPaymentRequest  struct{}
	StartPaymentResponse struct{}
)

type (
	VerifyCardRequest  struct{}
	VerifyCardResponse struct{}
)

type (
	ChallengeCardRequest  struct{}
	ChallengeCardResponse struct{}
)

type (
	CapturePaymentRequest  struct{}
	CapturePaymentResponse struct{}
)
