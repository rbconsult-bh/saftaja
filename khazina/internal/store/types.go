package store

type ProjectEnvironment string

const (
	ProjectEnvironmentProduction ProjectEnvironment = "production"
	ProjectEnvironmentSandbox    ProjectEnvironment = "sandbox"
)

type ConnectorType string

const (
	ConnectorTypeMPGS ConnectorType = "mpgs"
)

type InvoiceStatus string

const (
	InvoiceStatusPending   InvoiceStatus = "pending"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
)

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

type GatewayOperationType string

const (
	GatewayOperationTypePrepareCardAuthentication   GatewayOperationType = "prepare_card_authentication"
	GatewayOperationTypeAuthenticateCardholder      GatewayOperationType = "authenticate_cardholder"
	GatewayOperationTypeGetCardAuthenticationResult GatewayOperationType = "get_card_authentication_result"
	GatewayOperationTypeCapturePayment              GatewayOperationType = "capture_payment"
)

type GatewayOperationStatus string

const (
	GatewayOperationStatusPending   GatewayOperationStatus = "pending"
	GatewayOperationStatusCompleted GatewayOperationStatus = "completed"
	GatewayOperationStatusErrored   GatewayOperationStatus = "errored"
)
