package store

type ProjectEnvironment string

const (
	ProjectEnvironmentProduction ProjectEnvironment = "production"
	ProjectEnvironmentSandbox    ProjectEnvironment = "sandbox"
)

type GatewayAccountConnectorType string

const (
	GatewayAccountConnectorTypeMPGS GatewayAccountConnectorType = "mpgs"
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

type PaymentSessionPaymentMethod string

const (
	PaymentSessionPaymentMethodCard     PaymentSessionPaymentMethod = "card"
	PaymentSessionPaymentMethodApplePay PaymentSessionPaymentMethod = "apple_pay"
)

type TransactionTransactionType string

const (
	TransactionTransactionTypeInitiateAuthentication TransactionTransactionType = "initiate_authentication"
	TransactionTransactionTypeAuthenticatePayer      TransactionTransactionType = "authenticate_payer"
	TransactionTransactionTypePay                    TransactionTransactionType = "pay"
)

type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "pending"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)
