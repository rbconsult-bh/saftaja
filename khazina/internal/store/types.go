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

type GatewayOperationType string

const (
	GatewayOperationTypeInitiateAuth      GatewayOperationType = "initiate_authentication"
	GatewayOperationTypeAuthenticatePayer GatewayOperationType = "authenticate_payer"
	GatewayOperationTypePay               GatewayOperationType = "pay"
)

type GatewayOperationStatus string

const (
	GatewayOperationStatusPending GatewayOperationStatus = "pending"
	GatewayOperationStatusSuccess GatewayOperationStatus = "success"
	GatewayOperationStatusFailed  GatewayOperationStatus = "failed"
)
