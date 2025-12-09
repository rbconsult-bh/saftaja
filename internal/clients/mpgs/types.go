package mpgs

import (
	"context"
	"fmt"
)

// Client interface defines the MPGS API operations
type Client interface {
	CreateSession(ctx context.Context, req *CreateSessionRequest) (*Response[CreateSessionResponse], error)
	UpdateSession(ctx context.Context, sessionID string, req *UpdateSessionRequest) (*Response[UpdateSessionResponse], error)

	InitiateAuthentication(ctx context.Context, orderID, txID string, req *InitiateAuthenticationRequest) (*Response[InitiateAuthenticationResponse], error)
	AuthenticatePayer(ctx context.Context, orderID, txID string, req *AuthenticatePayerRequest) (*Response[AuthenticatePayerResponse], error)

	RetrieveTransaction(ctx context.Context, orderID, txID string) (*Response[RetrieveTransactionResponse], error)

	ExecutePay(ctx context.Context, orderID, txID string, req *ExecutePayRequest) (*Response[ExecutePayResponse], error)
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

type (
	InitiateAuthenticationReqAuthentication struct {
		Channel AuthChannel `json:"channel"`
	}
	InitiateAuthenticationOrder struct {
		Currency string `json:"currency"`
	}
	InitiateAuthenticationSession struct {
		ID string `json:"id"`
	}
	InitiateAuthenticationRequest struct {
		APIOperation   APIOperation                            `json:"apiOperation"`
		Authentication InitiateAuthenticationReqAuthentication `json:"authentication"`
		Order          InitiateAuthenticationOrder             `json:"order"`
		Session        InitiateAuthenticationSession           `json:"session"`
	}

	InitiateAuthenticationRespAuthentication struct {
		ThreeDS2       InitiateAuthenticationThreeDS2Data `json:"3ds2"`
		AcceptVersions string                             `json:"acceptVersions"`
		Channel        string                             `json:"channel"`
		Purpose        string                             `json:"purpose"`
		Redirect       InitiateAuthenticationRedirectData `json:"redirect"`
		Version        string                             `json:"version"`
	}
	InitiateAuthenticationThreeDS2Data struct {
		AuthenticationScheme string `json:"authenticationScheme"`
		DirectoryServerID    string `json:"directoryServerId"`
		MethodCompleted      bool   `json:"methodCompleted"`
		MethodSupported      string `json:"methodSupported"`
		ProtocolVersion      string `json:"protocolVersion"`
		RequestorID          string `json:"requestorId"`
		RequestorName        string `json:"requestorName"`
	}
	InitiateAuthenticationRedirectData struct {
		CustomizedHTML InitiateAuthenticationCustomizedHTML `json:"customizedHtml"`
		HTML           string                               `json:"html"`
	}
	InitiateAuthenticationCustomizedHTML struct {
		ThreeDS2Method InitiateAuthenticationThreeDS2Method `json:"3ds2"`
	}
	InitiateAuthenticationThreeDS2Method struct {
		MethodPostData string `json:"methodPostData"`
		MethodURL      string `json:"methodUrl"`
	}
	InitiateAuthenticationResponseOrder struct {
		AuthenticationStatus  string     `json:"authenticationStatus"`
		CreationTime          string     `json:"creationTime"`
		Currency              string     `json:"currency"`
		ID                    string     `json:"id"`
		LastUpdatedTime       string     `json:"lastUpdatedTime"`
		MerchantCategoryCode  string     `json:"merchantCategoryCode"`
		Status                AuthStatus `json:"status"`
		TotalAuthorizedAmount float64    `json:"totalAuthorizedAmount"`
		TotalCapturedAmount   float64    `json:"totalCapturedAmount"`
		TotalRefundedAmount   float64    `json:"totalRefundedAmount"`
	}
	InitiateAuthenticationGatewayResponse struct {
		GatewayCode           string                `json:"gatewayCode"`
		GatewayRecommendation GatewayRecommendation `json:"gatewayRecommendation"`
	}
	InitiateAuthenticationSourceOfFunds struct {
		Provided InitiateAuthenticationProvidedData `json:"provided"`
		Type     string                             `json:"type"`
	}
	InitiateAuthenticationProvidedData struct {
		Card InitiateAuthenticationCardData `json:"card"`
	}
	InitiateAuthenticationCardData struct {
		Brand         string                           `json:"brand"`
		Expiry        InitiateAuthenticationCardExpiry `json:"expiry"`
		FundingMethod string                           `json:"fundingMethod"`
		Number        string                           `json:"number"`
		Scheme        string                           `json:"scheme"`
	}
	InitiateAuthenticationCardExpiry struct {
		Month string `json:"month"`
		Year  string `json:"year"`
	}
	InitiateAuthenticationTransaction struct {
		Amount               float64 `json:"amount"`
		AuthenticationStatus string  `json:"authenticationStatus"`
		Currency             string  `json:"currency"`
		ID                   string  `json:"id"`
		Type                 string  `json:"type"`
	}
	InitiateAuthenticationResponse struct {
		Authentication   InitiateAuthenticationRespAuthentication `json:"authentication"`
		Merchant         string                                   `json:"merchant"`
		Order            InitiateAuthenticationOrder              `json:"order"`
		Response         InitiateAuthenticationGatewayResponse    `json:"response"`
		Result           string                                   `json:"result"`
		SourceOfFunds    InitiateAuthenticationSourceOfFunds      `json:"sourceOfFunds"`
		TimeOfLastUpdate string                                   `json:"timeOfLastUpdate"`
		TimeOfRecord     string                                   `json:"timeOfRecord"`
		Transaction      InitiateAuthenticationTransaction        `json:"transaction"`
		Version          string                                   `json:"version"`
	}
)

type (
	AuthenticatePayerReqAuthentication struct {
		RedirectResponseURL string `json:"redirectResponseUrl"`
	}
	AuthenticatePayerReqBrowserDetails struct {
		ThreeDSecureChallengeWindowSize string `json:"3DSecureChallengeWindowSize,omitempty"`
		AcceptHeaders                   string `json:"acceptHeaders,omitempty"`
		ColorDepth                      int    `json:"colorDepth,omitempty"`
		JavaEnabled                     bool   `json:"javaEnabled"`
		Language                        string `json:"language,omitempty"`
		ScreenHeight                    int    `json:"screenHeight,omitempty"`
		ScreenWidth                     int    `json:"screenWidth,omitempty"`
		TimeZone                        int    `json:"timeZone,omitempty"`
	}
	AuthenticatePayerReqDevice struct {
		Browser        string                              `json:"browser,omitempty"`
		BrowserDetails *AuthenticatePayerReqBrowserDetails `json:"browserDetails,omitempty"`
		IPAddress      string                              `json:"ipAddress,omitempty"`
	}
	AuthenticatePayerReqOrder struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}
	AuthenticatePayerReqSession struct {
		ID string `json:"id"`
	}
	AuthenticatePayerRequest struct {
		APIOperation   APIOperation                       `json:"apiOperation"`
		Authentication AuthenticatePayerReqAuthentication `json:"authentication"`
		Device         AuthenticatePayerReqDevice         `json:"device"`
		Order          AuthenticatePayerReqOrder          `json:"order"`
		Session        AuthenticatePayerReqSession        `json:"session"`
	}

	AuthenticatePayerRespThreeDS struct {
		TransactionID string `json:"transactionId"`
	}
	AuthenticatePayerRespThreeDS2 struct {
		ThreeDSServerTransactionID string `json:"3dsServerTransactionId"`
		ACSReference               string `json:"acsReference"`
		ACSTransactionID           string `json:"acsTransactionId"`
		AuthenticationScheme       string `json:"authenticationScheme"`
		DirectoryServerID          string `json:"directoryServerId"`
		DSReference                string `json:"dsReference"`
		DSTransactionID            string `json:"dsTransactionId"`
		MethodCompleted            bool   `json:"methodCompleted"`
		MethodSupported            string `json:"methodSupported"`
		ProtocolVersion            string `json:"protocolVersion"`
		RequestorID                string `json:"requestorId"`
		RequestorName              string `json:"requestorName"`
		TransactionStatus          string `json:"transactionStatus"`
	}
	AuthenticatePayerResp3DS2Data struct {
		ACSURL string `json:"acsUrl"`
		CReq   string `json:"cReq"`
	}
	AuthenticatePayerRespCustomizedHTML struct {
		ThreeDS2 AuthenticatePayerResp3DS2Data `json:"3ds2"`
	}
	AuthenticatePayerRespRedirect struct {
		CustomizedHTML AuthenticatePayerRespCustomizedHTML `json:"customizedHtml"`
		DomainName     string                              `json:"domainName"`
		HTML           string                              `json:"html"`
	}
	AuthenticatePayerRespAuthentication struct {
		ThreeDS          AuthenticatePayerRespThreeDS  `json:"3ds"`
		ThreeDS2         AuthenticatePayerRespThreeDS2 `json:"3ds2"`
		Amount           float64                       `json:"amount"`
		Method           string                        `json:"method"`
		PayerInteraction string                        `json:"payerInteraction"`
		Redirect         AuthenticatePayerRespRedirect `json:"redirect"`
		Time             string                        `json:"time"`
		Version          string                        `json:"version"`
	}
	AuthenticatePayerRespDevice struct {
		Browser   string `json:"browser"`
		IPAddress string `json:"ipAddress"`
	}
	AuthenticatePayerValueTransfer struct {
		AccountType string `json:"accountType"`
	}
	AuthenticatePayerRespOrder struct {
		Amount                float64                        `json:"amount"`
		AuthenticationStatus  string                         `json:"authenticationStatus"`
		CreationTime          string                         `json:"creationTime"`
		Currency              string                         `json:"currency"`
		ID                    string                         `json:"id"`
		LastUpdatedTime       string                         `json:"lastUpdatedTime"`
		MerchantCategoryCode  string                         `json:"merchantCategoryCode"`
		Status                AuthStatus                     `json:"status"`
		TotalAuthorizedAmount float64                        `json:"totalAuthorizedAmount"`
		TotalCapturedAmount   float64                        `json:"totalCapturedAmount"`
		TotalRefundedAmount   float64                        `json:"totalRefundedAmount"`
		ValueTransfer         AuthenticatePayerValueTransfer `json:"valueTransfer"`
	}
	AuthenticatePayerGatewayResponse struct {
		GatewayCode           string                `json:"gatewayCode"`
		GatewayRecommendation GatewayRecommendation `json:"gatewayRecommendation"`
	}
	AuthenticatePayerCardExpiry struct {
		Month string `json:"month"`
		Year  string `json:"year"`
	}
	AuthenticatePayerCardData struct {
		Brand         string                      `json:"brand"`
		Expiry        AuthenticatePayerCardExpiry `json:"expiry"`
		FundingMethod string                      `json:"fundingMethod"`
		NameOnCard    string                      `json:"nameOnCard"`
		Number        string                      `json:"number"`
		Scheme        string                      `json:"scheme"`
	}
	AuthenticatePayerProvidedData struct {
		Card AuthenticatePayerCardData `json:"card"`
	}
	AuthenticatePayerRespSourceOfFunds struct {
		Provided AuthenticatePayerProvidedData `json:"provided"`
		Type     string                        `json:"type"`
	}
	AuthenticatePayerAcquirer struct {
		MerchantID string `json:"merchantId"`
	}
	AuthenticatePayerRespTransaction struct {
		Acquirer             AuthenticatePayerAcquirer `json:"acquirer"`
		Amount               float64                   `json:"amount"`
		AuthenticationStatus string                    `json:"authenticationStatus"`
		Currency             string                    `json:"currency"`
		ID                   string                    `json:"id"`
		Type                 string                    `json:"type"`
	}
	AuthenticatePayerResponse struct {
		Authentication   AuthenticatePayerRespAuthentication `json:"authentication"`
		Device           AuthenticatePayerRespDevice         `json:"device"`
		Merchant         string                              `json:"merchant"`
		Order            AuthenticatePayerRespOrder          `json:"order"`
		Response         AuthenticatePayerGatewayResponse    `json:"response"`
		Result           string                              `json:"result"`
		SourceOfFunds    AuthenticatePayerRespSourceOfFunds  `json:"sourceOfFunds"`
		TimeOfLastUpdate string                              `json:"timeOfLastUpdate"`
		TimeOfRecord     string                              `json:"timeOfRecord"`
		Transaction      AuthenticatePayerRespTransaction    `json:"transaction"`
		Version          string                              `json:"version"`
	}
)

type (
	RetrieveTransactionAuthentication struct {
		Amount           float64 `json:"amount"`
		Method           string  `json:"method"`
		PayerInteraction string  `json:"payerInteraction"`
		Time             string  `json:"time"`
		Version          string  `json:"version"`
	}
	RetrieveTransactionDevice struct {
		Browser   string `json:"browser"`
		IPAddress string `json:"ipAddress"`
	}
	RetrieveTransactionOrder struct {
		Amount               float64 `json:"amount"`
		AuthenticationStatus string  `json:"authenticationStatus"`
		CreationTime         string  `json:"creationTime"`
		Currency             string  `json:"currency"`
		ID                   string  `json:"id"`
		Status               string  `json:"status"`
	}
	RetrieveTransactionGatewayResp struct {
		GatewayCode           string `json:"gatewayCode"`
		GatewayRecommendation string `json:"gatewayRecommendation"`
	}
	RetrieveTransactionSourceOfFunds struct {
		Provided struct {
			Card struct {
				Brand  string `json:"brand"`
				Number string `json:"number"`
				Scheme string `json:"scheme"`
			} `json:"card"`
		} `json:"provided"`
		Type string `json:"type"`
	}
	RetrieveTransactionTx struct {
		Acquirer struct {
			MerchantID string `json:"merchantId"`
		} `json:"acquirer"`
		Amount               float64 `json:"amount"`
		AuthenticationStatus string  `json:"authenticationStatus"`
		Currency             string  `json:"currency"`
		ID                   string  `json:"id"`
		Type                 string  `json:"type"`
	}
	RetrieveTransactionResponse struct {
		Authentication RetrieveTransactionAuthentication `json:"authentication"`
		Device         RetrieveTransactionDevice         `json:"device"`
		Merchant       string                            `json:"merchant"`
		Order          RetrieveTransactionOrder          `json:"order"`
		Response       RetrieveTransactionGatewayResp    `json:"response"`
		Result         string                            `json:"result"`
		SourceOfFunds  RetrieveTransactionSourceOfFunds  `json:"sourceOfFunds"`
		Transaction    RetrieveTransactionTx             `json:"transaction"`
		Version        string                            `json:"version"`
	}
)

type (
	ExecutePayReqAuthentication struct {
		TransactionID string `json:"transactionId"`
	}
	ExecutePayReqOrder struct {
		Amount    string `json:"amount"`
		Currency  string `json:"currency"`
		Reference string `json:"reference,omitempty"`
	}
	ExecutePayReqSession struct {
		ID string `json:"id"`
	}
	ExecutePayReqTransaction struct {
		Reference string `json:"reference,omitempty"`
	}
	ExecutePayRequest struct {
		APIOperation   APIOperation                `json:"apiOperation"`
		Authentication ExecutePayReqAuthentication `json:"authentication"`
		Order          ExecutePayReqOrder          `json:"order"`
		Session        ExecutePayReqSession        `json:"session"`
		Transaction    *ExecutePayReqTransaction   `json:"transaction,omitempty"`
	}

	ExecutePayGatewayResponse struct {
		GatewayCode GatewayCode `json:"gatewayCode"`
	}
	ExecutePayAcquirer struct {
		Batch          int    `json:"batch,omitempty"`
		Date           string `json:"date,omitempty"`
		ID             string `json:"id,omitempty"`
		MerchantID     string `json:"merchantId"`
		SettlementDate string `json:"settlementDate,omitempty"`
		TimeZone       string `json:"timeZone,omitempty"`
		TransactionID  string `json:"transactionId,omitempty"`
	}
	ExecutePayTransaction struct {
		Acquirer ExecutePayAcquirer `json:"acquirer"`
		Amount   float64            `json:"amount"`
		Currency string             `json:"currency"`
		ID       string             `json:"id"`
		Type     TransactionType    `json:"type"`
	}
	ExecutePayOrder struct {
		Amount                float64 `json:"amount"`
		AuthenticationStatus  string  `json:"authenticationStatus,omitempty"`
		CreationTime          string  `json:"creationTime,omitempty"`
		Currency              string  `json:"currency"`
		ID                    string  `json:"id,omitempty"`
		TotalAuthorizedAmount float64 `json:"totalAuthorizedAmount,omitempty"`
		TotalCapturedAmount   float64 `json:"totalCapturedAmount,omitempty"`
		TotalRefundedAmount   float64 `json:"totalRefundedAmount,omitempty"`
	}
	ExecutePayCardExpiry struct {
		Month string `json:"month"`
		Year  string `json:"year"`
	}
	ExecutePayCardData struct {
		Brand         string               `json:"brand"`
		Expiry        ExecutePayCardExpiry `json:"expiry"`
		FundingMethod string               `json:"fundingMethod"`
		Number        string               `json:"number"`
		Scheme        string               `json:"scheme"`
	}
	ExecutePayProvidedData struct {
		Card ExecutePayCardData `json:"card"`
	}
	ExecutePaySourceOfFunds struct {
		Provided ExecutePayProvidedData `json:"provided"`
		Type     string                 `json:"type"`
	}
	ExecutePayResponse struct {
		Merchant      string                    `json:"merchant"`
		Order         ExecutePayOrder           `json:"order"`
		Response      ExecutePayGatewayResponse `json:"response"`
		Result        string                    `json:"result"`
		SourceOfFunds *ExecutePaySourceOfFunds  `json:"sourceOfFunds,omitempty"`
		Transaction   ExecutePayTransaction     `json:"transaction"`
	}
)
