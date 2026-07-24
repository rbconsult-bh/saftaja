package billing

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidArgument                = errors.New("invalid argument")
	ErrNotFound                       = errors.New("not found")
	ErrUnsupportedPaymentMethod       = errors.New("unsupported payment method")
	ErrInvoiceAlreadyPaid             = errors.New("invoice already paid")
	ErrInvoiceCancelled               = errors.New("invoice cancelled")
	ErrInvoiceNotFound                = errors.New("invoice not found")
	ErrInvoiceInvalidState            = errors.New("invoice invalid state")
	ErrGatewayAccountNotFound         = errors.New("gateway account not found")
	ErrIdempotencyMismatch            = errors.New("idempotency key already used with different parameters")
	ErrPaymentIntentExpired           = errors.New("payment intent expired")
	ErrPaymentIntentInvalidState      = errors.New("payment intent invalid state")
	ErrPaymentIntentInvalidTransition = errors.New("payment intent invalid transition")
	ErrUnsupportedGateway             = errors.New("unsupported gateway")
)

type Service interface {
	GetInvoice(ctx context.Context, r GetInvoiceRequest) (*GetInvoiceResponse, error)

	StartPayment(ctx context.Context, r StartPaymentRequest) (*StartPaymentResponse, error)
	CapturePayment(ctx context.Context, r CapturePaymentRequest) (*CapturePaymentResponse, error)

	PrepareCardChallenge(ctx context.Context, r PrepareCardChallengeRequest) (*PrepareCardChallengeResponse, error)
	StartCardChallenge(ctx context.Context, r StartCardChallengeRequest) (*StartCardChallengeResponse, error)
	CompleteCardChallenge(ctx context.Context, r CompleteCardChallengeRequest) (*CompleteCardChallengeResponse, error)
}

type PaymentIntentRef struct {
	ProjectID       uuid.UUID
	InvoiceID       uuid.UUID
	PaymentIntentID uuid.UUID
}

func (r *PaymentIntentRef) Validate() error {
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

type PaymentIntentStatus string

const (
	PaymentIntentStatusCreated                     PaymentIntentStatus = "created"
	PaymentIntentStatusReadyToStartChallenge       PaymentIntentStatus = "ready_to_start_challenge"
	PaymentIntentStatusAwaitingChallengeCompletion PaymentIntentStatus = "awaiting_challenge_completion"
	PaymentIntentStatusReadyToCapture              PaymentIntentStatus = "ready_to_capture"
	PaymentIntentStatusCapturingPayment            PaymentIntentStatus = "capturing_payment"
	PaymentIntentStatusSucceeded                   PaymentIntentStatus = "succeeded"
	PaymentIntentStatusFailed                      PaymentIntentStatus = "failed"
)

type PaymentMethod string

const (
	PaymentMethodCard     PaymentMethod = "card"
	PaymentMethodApplePay PaymentMethod = "apple_pay"
)

type InvoiceStatus string

const (
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
		InvoiceID      uuid.UUID
		ProjectID      uuid.UUID
		IdempotencyKey string
		// TODO: remove this, this is not a concern of consumer of this api,
		// they should provide only payment method they want to use.
		// We should build a gateway account resolver based on payment method.
		GatewayAccountID uuid.UUID
		PaymentMethod    PaymentMethod
		PayerIP          string
		PayerUserAgent   string
	}
	StartPaymentResponse struct {
		PaymentIntentID       uuid.UUID
		GatewaySetupReference *string
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
	CapturePaymentRequest  struct{}
	CapturePaymentResponse struct{}
)

type PrepareCardChallengeNextStep string

const (
	PrepareCardChallengeNextStepStartChallenge PrepareCardChallengeNextStep = "start_challenge"
	PrepareCardChallengeNextStepCantContinue   PrepareCardChallengeNextStep = "cant_continue"
)

type (
	PrepareCardChallengeRequest struct {
		ProjectID       uuid.UUID
		InvoiceID       uuid.UUID
		PaymentIntentID uuid.UUID
	}
	PrepareCardChallengeResponse struct {
		NextStep PrepareCardChallengeNextStep
	}
)

func (r *PrepareCardChallengeRequest) Validate() error {
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

type ThreeDSChallengeWindowSize string

const (
	ThreeDSChallengeWindowSize250x400    ThreeDSChallengeWindowSize = "250_X_400"
	ThreeDSChallengeWindowSize390x400    ThreeDSChallengeWindowSize = "390_X_400"
	ThreeDSChallengeWindowSize500x600    ThreeDSChallengeWindowSize = "500_X_600"
	ThreeDSChallengeWindowSize600x400    ThreeDSChallengeWindowSize = "600_X_400"
	ThreeDSChallengeWindowSizeFullScreen ThreeDSChallengeWindowSize = "FULL_SCREEN"
)

type ThreeDSBrowser struct {
	IPAddress           string
	UserAgent           string
	AcceptHeader        string
	ChallengeWindowSize ThreeDSChallengeWindowSize
	ColorDepth          int
	JavaEnabled         bool
	Language            string
	ScreenHeight        int
	ScreenWidth         int
	TimeZone            int
}

func (r ThreeDSBrowser) Validate() error {
	if strings.TrimSpace(r.IPAddress) == "" {
		return fmt.Errorf("%w: IPAddress is required", ErrInvalidArgument)
	}
	if _, err := netip.ParseAddr(r.IPAddress); err != nil {
		return fmt.Errorf("%w: IPAddress is invalid", ErrInvalidArgument)
	}
	if strings.TrimSpace(r.UserAgent) == "" {
		return fmt.Errorf("%w: UserAgent is required", ErrInvalidArgument)
	}
	if utf8.RuneCountInString(r.UserAgent) > 2048 {
		return fmt.Errorf("%w: UserAgent must not exceed 2048 characters", ErrInvalidArgument)
	}
	if strings.TrimSpace(r.AcceptHeader) == "" {
		return fmt.Errorf("%w: AcceptHeader is required", ErrInvalidArgument)
	}
	if utf8.RuneCountInString(r.AcceptHeader) > 2048 {
		return fmt.Errorf("%w: AcceptHeader must not exceed 2048 characters", ErrInvalidArgument)
	}

	switch r.ChallengeWindowSize {
	case ThreeDSChallengeWindowSize250x400,
		ThreeDSChallengeWindowSize390x400,
		ThreeDSChallengeWindowSize500x600,
		ThreeDSChallengeWindowSize600x400,
		ThreeDSChallengeWindowSizeFullScreen:
	default:
		return fmt.Errorf("%w: unsupported ChallengeWindowSize %q", ErrInvalidArgument, r.ChallengeWindowSize)
	}

	if r.ColorDepth < 1 || r.ColorDepth > 48 {
		return fmt.Errorf("%w: ColorDepth must be between 1 and 48", ErrInvalidArgument)
	}
	if strings.TrimSpace(r.Language) == "" {
		return fmt.Errorf("%w: Language is required", ErrInvalidArgument)
	}
	if utf8.RuneCountInString(r.Language) > 8 {
		return fmt.Errorf("%w: Language must not exceed 8 characters", ErrInvalidArgument)
	}
	if r.ScreenHeight < 1 || r.ScreenHeight > 999999 {
		return fmt.Errorf("%w: ScreenHeight must be between 1 and 999999", ErrInvalidArgument)
	}
	if r.ScreenWidth < 1 || r.ScreenWidth > 999999 {
		return fmt.Errorf("%w: ScreenWidth must be between 1 and 999999", ErrInvalidArgument)
	}
	if r.TimeZone < -840 || r.TimeZone > 840 {
		return fmt.Errorf("%w: TimeZone must be between -840 and 840", ErrInvalidArgument)
	}

	return nil
}

type StartCardChallengeNextStep string

const (
	StartCardChallengeNextStepCompleteChallenge StartCardChallengeNextStep = "complete_challenge"
	StartCardChallengeNextStepCapture           StartCardChallengeNextStep = "capture"
	StartCardChallengeNextStepCantContinue      StartCardChallengeNextStep = "cant_continue"
)

type (
	StartCardChallengeRequest struct {
		PaymentIntentRef
		Browser ThreeDSBrowser
	}
	StartCardChallengeResponse struct {
		NextStep StartCardChallengeNextStep
	}
)

func (r *StartCardChallengeRequest) Validate() error {
	if err := r.PaymentIntentRef.Validate(); err != nil {
		return err
	}
	if err := r.Browser.Validate(); err != nil {
		return err
	}

	return nil
}

type (
	CompleteCardChallengeRequest  struct{}
	CompleteCardChallengeResponse struct{}
)
