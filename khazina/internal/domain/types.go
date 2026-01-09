package domain

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
	InvoiceStatusPending    InvoiceStatus = "pending"
	InvoiceStatusProcessing InvoiceStatus = "processing"
	InvoiceStatusPaid       InvoiceStatus = "paid"
	InvoiceStatusFailed     InvoiceStatus = "failed"
)

type PaymentSessionStatus string

const (
	PaymentSessionStatusCreated        PaymentSessionStatus = "created"
	PaymentSessionStatusAuthenticating PaymentSessionStatus = "authenticating"
	PaymentSessionStatusAuthenticated  PaymentSessionStatus = "authenticated"
	PaymentSessionStatusPaying         PaymentSessionStatus = "paying"
	PaymentSessionStatusCompleted      PaymentSessionStatus = "completed"
	PaymentSessionStatusFailed         PaymentSessionStatus = "failed"
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
