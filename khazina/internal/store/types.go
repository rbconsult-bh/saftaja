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
	PaymentIntentStatusCreated        PaymentIntentStatus = "created"
	PaymentIntentStatusAuthenticating PaymentIntentStatus = "authenticating"
	PaymentIntentStatusAuthenticated  PaymentIntentStatus = "authenticated"
	PaymentIntentStatusPaying         PaymentIntentStatus = "paying"
	PaymentIntentStatusCompleted      PaymentIntentStatus = "completed"
	PaymentIntentStatusFailed         PaymentIntentStatus = "failed"
)

type PaymentMethod string

const (
	PaymentMethodCard     PaymentMethod = "card"
	PaymentMethodApplePay PaymentMethod = "apple_pay"
)

type TransactionType string

const (
	TransactionTypeInitiateAuth      TransactionType = "initiate_authentication"
	TransactionTypeAuthenticatePayer TransactionType = "authenticate_payer"
	TransactionTypePay               TransactionType = "pay"
)

type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "pending"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)
