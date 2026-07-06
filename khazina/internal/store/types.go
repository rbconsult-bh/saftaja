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
