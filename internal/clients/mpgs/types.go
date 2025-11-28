package mpgs

import (
	"context"
	"fmt"
)

// Client interface defines the MPGS API operations
type Client interface {
	// CreateSession creates a payment session that can be used to temporarily store request fields
	CreateSession(ctx context.Context, req *CreateSessionRequest) (*Response[CreateSessionResponse], error)
	UpdateSession(ctx context.Context, sessionID string, req *UpdateSessionRequest) (*Response[UpdateSessionResponse], error)

	// InitiateAuthentication(ctx context.Context, req *InitiateAuthenticationRequest) (*Response[InitiateAuthenticationResponse], error)
	// AuthenticatePayer(ctx context.Context, req *AuthenticatePayerRequest) (*Response[AuthenticatePayerResponse], error)
}

// Response is a generic wrapper for all API responses
type Response[T any] struct {
	Data T
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	ErrorDetail *ErrorDetail `json:"error,omitempty"`
	Result      string       `json:"result,omitempty"` // ERROR
}

// Error implements the error interface for ErrorResponse
func (e ErrorResponse) Error() string {
	if e.ErrorDetail != nil {
		msg := fmt.Sprintf("MPGS API error: %s", e.ErrorDetail.Cause)
		if e.ErrorDetail.Explanation != "" {
			msg += fmt.Sprintf(" - %s", e.ErrorDetail.Explanation)
		}
		if e.ErrorDetail.Field != "" {
			msg += fmt.Sprintf(" (field: %s)", e.ErrorDetail.Field)
		}
		return msg
	}
	return "MPGS API error"
}

// ErrorDetail contains error details from the API
type ErrorDetail struct {
	// Cause broadly categorizes the cause of the error
	// Values: INVALID_REQUEST, REQUEST_REJECTED, SERVER_BUSY, SERVER_FAILED
	Cause string `json:"cause,omitempty"`

	// Explanation provides textual description of the error based on the cause
	// Returned only if cause is INVALID_REQUEST or SERVER_BUSY
	Explanation string `json:"explanation,omitempty"`

	// Field indicates the name of the field that failed validation
	// Returned only if cause is INVALID_REQUEST and a field level validation error occurred
	Field string `json:"field,omitempty"`

	// SupportCode helps support team identify the exact cause
	// Returned only if cause is SERVER_FAILED or REQUEST_REJECTED
	SupportCode string `json:"supportCode,omitempty"`

	// ValidationType indicates the type of field validation error
	// Values: INVALID, MISSING, UNSUPPORTED
	// Returned only if cause is INVALID_REQUEST and field validation error occurred
	ValidationType string `json:"validationType,omitempty"`
}

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	// CorrelationID is a transient identifier that can be used to match response to request
	// Not validated, does not persist, returned as provided
	// Min: 1, Max: 100 characters
	CorrelationID string `json:"correlationId,omitempty"`

	// Session contains session configuration
	Session *CreateSessionRequestSession `json:"session,omitempty"`
}

// CreateSessionRequestSession contains session configuration for creation
type CreateSessionRequestSession struct {
	// AuthenticationLimit is the number of operations that may be submitted using this session ID as password
	// Used when browser/mobile app issues operations from device using session ID
	// Limits exposure to risk of unauthorized requests
	// Min: 0, Max: 25
	AuthenticationLimit *int32 `json:"authenticationLimit,omitempty"`
}

// CreateSessionResponse represents the response from creating a session
type CreateSessionResponse struct {
	// CorrelationID echoes back the transient identifier from request
	CorrelationID string `json:"correlationId,omitempty"`

	// LineOfBusiness if merchant profile supports multiple lines
	// Each can have different payment parameters (bank account, supported cards, etc)
	LineOfBusiness string `json:"lineOfBusiness,omitempty"`

	// Merchant is the unique identifier issued by payment provider
	// Up to 12 characters, alphanumeric + '-', '_'
	Merchant string `json:"merchant"`

	// Result is the high level outcome
	// Values: SUCCESS, FAILURE, PENDING, UNKNOWN
	Result string `json:"result"`

	// Session contains the created session details
	Session *SessionDetails `json:"session"`
}

// SessionDetails contains the session information returned in responses
type SessionDetails struct {
	// AES256Key to decrypt sensitive data passed via payer's browser/device
	// Base64 encoded AES256 key, unique for this session
	// Should never be exposed to payer environment
	// Length: exactly 44 characters
	AES256Key string `json:"aes256Key"`

	// AuthenticationLimit number of operations allowed using session ID as password
	// Min: 0, Max: 9999
	AuthenticationLimit int32 `json:"authenticationLimit"`

	// ID is the identifier for the payment session
	// Can be used with Hosted Payment Form, wallet provider, or Update Session
	// Length: 31-35 characters
	ID string `json:"id"`

	// UpdateStatus summary of last attempt to modify session
	// Must be SUCCESS to perform operations using this session
	// Values: SUCCESS, FAILURE, NO_UPDATE
	UpdateStatus string `json:"updateStatus"`

	// Version for optimistic locking of session content
	// Record when making decisions, pass when submitting operations
	// Length: exactly 10 characters
	Version string `json:"version"`
}

type (
	UpdateSessionOrder struct {
		Amount   string `json:"amount,omitempty"`
		Currency string `json:"currency,omitempty"`
		ID       string `json:"id,omitempty"`
	}
	UpdateSessionRequest struct {
		Order UpdateSessionOrder `json:"order"`
	}
	UpdateSessionResponse struct{}
)
