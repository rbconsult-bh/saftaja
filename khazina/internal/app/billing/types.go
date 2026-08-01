package billing

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
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
	ErrInvoicePaymentInProgress       = errors.New("invoice payment in progress")
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

	CreatePaymentIntent(ctx context.Context, r CreatePaymentIntentRequest) (*CreatePaymentIntentResponse, error)
	CapturePaymentIntent(ctx context.Context, r CapturePaymentIntentRequest) (*CapturePaymentIntentResponse, error)

	PrepareCardAuthentication(ctx context.Context, r PrepareCardAuthenticationRequest) (*PrepareCardAuthenticationResponse, error)
	AuthenticateCardholder(ctx context.Context, r AuthenticateCardholderRequest) (*AuthenticateCardholderResponse, error)
	VerifyCardAuthentication(ctx context.Context, r VerifyCardAuthenticationRequest) (*VerifyCardAuthenticationResponse, error)
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
	PaymentIntentStatusCreated                      PaymentIntentStatus = "created"
	PaymentIntentStatusReadyToAuthenticate          PaymentIntentStatus = "ready_to_authenticate"
	PaymentIntentStatusAwaitingAuthenticationResult PaymentIntentStatus = "awaiting_authentication_result"
	PaymentIntentStatusReadyToCapture               PaymentIntentStatus = "ready_to_capture"
	PaymentIntentStatusCapturing                    PaymentIntentStatus = "capturing"
	PaymentIntentStatusSucceeded                    PaymentIntentStatus = "succeeded"
	PaymentIntentStatusFailed                       PaymentIntentStatus = "failed"
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
	CreatePaymentIntentRequest struct {
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
	CreatePaymentIntentResponse struct {
		PaymentIntentID        uuid.UUID
		PaymentMethodReference *PaymentMethodReference
	}
)

func (r *CreatePaymentIntentRequest) Validate() error {
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

type CapturePaymentIntentNextStep string

const (
	CapturePaymentIntentNextStepComplete     CapturePaymentIntentNextStep = "complete"
	CapturePaymentIntentNextStepProcessing   CapturePaymentIntentNextStep = "processing"
	CapturePaymentIntentNextStepCantContinue CapturePaymentIntentNextStep = "cant_continue"
)

type (
	CapturePaymentIntentRequest struct {
		PaymentIntentRef
	}
	CapturePaymentIntentResponse struct {
		NextStep CapturePaymentIntentNextStep
	}
)

func (r *CapturePaymentIntentRequest) Validate() error {
	return r.PaymentIntentRef.Validate()
}

type PrepareCardAuthenticationNextStep string

const (
	PrepareCardAuthenticationNextStepAuthenticate PrepareCardAuthenticationNextStep = "authenticate"
	PrepareCardAuthenticationNextStepCantContinue PrepareCardAuthenticationNextStep = "cant_continue"
)

type (
	PrepareCardAuthenticationRequest struct {
		// TODO: use PaymentIntentRef
		ProjectID       uuid.UUID
		InvoiceID       uuid.UUID
		PaymentIntentID uuid.UUID
	}
	PrepareCardAuthenticationResponse struct {
		NextStep PrepareCardAuthenticationNextStep
	}
)

func (r *PrepareCardAuthenticationRequest) Validate() error {
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

type AuthenticateCardholderNextStep string

const (
	AuthenticateCardholderNextStepChallenge    AuthenticateCardholderNextStep = "challenge"
	AuthenticateCardholderNextStepCapture      AuthenticateCardholderNextStep = "capture"
	AuthenticateCardholderNextStepCantContinue AuthenticateCardholderNextStep = "cant_continue"
)

type (
	AuthenticateCardholderRequest struct {
		PaymentIntentRef
		Browser            ThreeDSBrowser
		ChallengeReturnURL string
	}
	AuthenticateCardholderResponse struct {
		NextStep     AuthenticateCardholderNextStep
		RedirectHTML string
	}
)

func (r *AuthenticateCardholderRequest) Validate() error {
	if err := r.PaymentIntentRef.Validate(); err != nil {
		return err
	}
	if err := r.Browser.Validate(); err != nil {
		return err
	}
	challengeReturnURL, err := url.Parse(r.ChallengeReturnURL)
	if err != nil || challengeReturnURL.Scheme != "https" || challengeReturnURL.Host == "" {
		return fmt.Errorf("%w: ChallengeReturnURL must be an absolute HTTPS URL", ErrInvalidArgument)
	}

	return nil
}

type VerifyCardAuthenticationNextStep string

const (
	VerifyCardAuthenticationNextStepPending      VerifyCardAuthenticationNextStep = "pending"
	VerifyCardAuthenticationNextStepCapture      VerifyCardAuthenticationNextStep = "capture"
	VerifyCardAuthenticationNextStepCantContinue VerifyCardAuthenticationNextStep = "cant_continue"
)

type (
	VerifyCardAuthenticationRequest struct {
		PaymentIntentRef
	}
	VerifyCardAuthenticationResponse struct {
		NextStep VerifyCardAuthenticationNextStep
	}
)

func (r *VerifyCardAuthenticationRequest) Validate() error {
	return r.PaymentIntentRef.Validate()
}
