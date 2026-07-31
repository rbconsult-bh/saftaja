package mpgs

const (
	APIVersion = "100"
)

// Result enum values
const (
	ResultSuccess = "SUCCESS"
	ResultFailure = "FAILURE"
	ResultPending = "PENDING"
	ResultUnknown = "UNKNOWN"
	ResultError   = "ERROR"
)

// UpdateStatus enum values
const (
	UpdateStatusSuccess  = "SUCCESS"
	UpdateStatusFailure  = "FAILURE"
	UpdateStatusNoUpdate = "NO_UPDATE"
)

// Error cause enum values
const (
	ErrorCauseInvalidRequest  = "INVALID_REQUEST"
	ErrorCauseRequestRejected = "REQUEST_REJECTED"
	ErrorCauseServerBusy      = "SERVER_BUSY"
	ErrorCauseServerFailed    = "SERVER_FAILED"
)

// ValidationError type enum values
const (
	ValidationTypeInvalid     = "INVALID"
	ValidationTypeMissing     = "MISSING"
	ValidationTypeUnsupported = "UNSUPPORTED"
)

type (
	APIOperation          string
	AuthChannel           string
	AuthStatus            string
	GatewayRecommendation string

	AuthenticationMethod string
	PayerInteraction     string
	TransactionStatus    string

	GatewayCode     string
	TransactionType string
)

const (
	OperationInitiateAuthentication APIOperation = "INITIATE_AUTHENTICATION"
	OperationAuthenticatePayer      APIOperation = "AUTHENTICATE_PAYER"
	OperationPay                    APIOperation = "PAY"
	OperationAuthorize              APIOperation = "AUTHORIZE"

	ChannelPayerBrowser AuthChannel = "PAYER_BROWSER"
	ChannelMerchant     AuthChannel = "MERCHANT_REQUESTED"

	AuthStatusInitiated  AuthStatus = "AUTHENTICATION_INITIATED"
	AuthStatusAvailable  AuthStatus = "AUTHENTICATION_AVAILABLE"
	AuthStatusAttempted  AuthStatus = "AUTHENTICATION_ATTEMPTED"
	AuthStatusPending    AuthStatus = "AUTHENTICATION_PENDING"
	AuthStatusSuccessful AuthStatus = "AUTHENTICATION_SUCCESSFUL"
	AuthStatusFailed     AuthStatus = "AUTHENTICATION_FAILED"

	GatewayRecommendationProceed                  GatewayRecommendation = "PROCEED"
	GatewayRecommendationDoNotProceed             GatewayRecommendation = "DO_NOT_PROCEED"
	GatewayRecommendationDoNotProceedAbandonOrder GatewayRecommendation = "DO_NOT_PROCEED_ABANDON_ORDER"
	GatewayRecommendationResubmitWithAltPay       GatewayRecommendation = "RESUBMIT_WITH_ALTERNATIVE_PAYMENT_DETAILS"

	MethodOutOfBand AuthenticationMethod = "OUT_OF_BAND"

	InteractionRequired PayerInteraction = "REQUIRED"

	TransStatusChallenge TransactionStatus = "C"
	TransStatusSuccess   TransactionStatus = "Y"
	TransStatusFailed    TransactionStatus = "N"

	CodeAborted                   GatewayCode = "ABORTED"
	CodeAcquirerSystemError       GatewayCode = "ACQUIRER_SYSTEM_ERROR"
	CodeApproved                  GatewayCode = "APPROVED"
	CodeApprovedAuto              GatewayCode = "APPROVED_AUTO"
	CodeApprovedPendingSettlement GatewayCode = "APPROVED_PENDING_SETTLEMENT"
	CodeAuthenticationFailed      GatewayCode = "AUTHENTICATION_FAILED"
	CodeAuthenticationInProgress  GatewayCode = "AUTHENTICATION_IN_PROGRESS"
	CodeBalanceAvailable          GatewayCode = "BALANCE_AVAILABLE"
	CodeBalanceUnknown            GatewayCode = "BALANCE_UNKNOWN"
	CodeBlocked                   GatewayCode = "BLOCKED"
	CodeCancelled                 GatewayCode = "CANCELLED"
	CodeDeclined                  GatewayCode = "DECLINED"
	CodeDeclinedAVS               GatewayCode = "DECLINED_AVS"
	CodeDeclinedAVSCSC            GatewayCode = "DECLINED_AVS_CSC"
	CodeDeclinedCSC               GatewayCode = "DECLINED_CSC"
	CodeDeclinedDoNotContact      GatewayCode = "DECLINED_DO_NOT_CONTACT"
	CodeDeclinedInvalidPIN        GatewayCode = "DECLINED_INVALID_PIN"
	CodeDeclinedPaymentPlan       GatewayCode = "DECLINED_PAYMENT_PLAN"
	CodeDeclinedPINRequired       GatewayCode = "DECLINED_PIN_REQUIRED"
	CodeDeferredTransaction       GatewayCode = "DEFERRED_TRANSACTION_RECEIVED"
	CodeDuplicateBatch            GatewayCode = "DUPLICATE_BATCH"
	CodeExceededRetryLimit        GatewayCode = "EXCEEDED_RETRY_LIMIT"
	CodeExpiredCard               GatewayCode = "EXPIRED_CARD"
	CodeInsufficientFunds         GatewayCode = "INSUFFICIENT_FUNDS"
	CodeInvalidCSC                GatewayCode = "INVALID_CSC"
	CodeLockFailure               GatewayCode = "LOCK_FAILURE"
	CodeNotEnrolled3DS            GatewayCode = "NOT_ENROLLED_3D_SECURE"
	CodeNotSupported              GatewayCode = "NOT_SUPPORTED"
	CodeNoBalance                 GatewayCode = "NO_BALANCE"
	CodePartiallyApproved         GatewayCode = "PARTIALLY_APPROVED"
	CodePending                   GatewayCode = "PENDING"
	CodeReferred                  GatewayCode = "REFERRED"
	CodeSubmitted                 GatewayCode = "SUBMITTED"
	CodeSystemError               GatewayCode = "SYSTEM_ERROR"
	CodeTimedOut                  GatewayCode = "TIMED_OUT"
	CodeUnknown                   GatewayCode = "UNKNOWN"
	CodeUnspecifiedFailure        GatewayCode = "UNSPECIFIED_FAILURE"

	TypeAuthentication      TransactionType = "AUTHENTICATION"
	TypeAuthorization       TransactionType = "AUTHORIZATION"
	TypeAuthorizationUpdate TransactionType = "AUTHORIZATION_UPDATE"
	TypeCapture             TransactionType = "CAPTURE"
	TypeChargeback          TransactionType = "CHARGEBACK"
	TypeDisbursement        TransactionType = "DISBURSEMENT"
	TypeFunding             TransactionType = "FUNDING"
	TypePayment             TransactionType = "PAYMENT"
	TypeRefund              TransactionType = "REFUND"
	TypeRefundRequest       TransactionType = "REFUND_REQUEST"
	TypeVerification        TransactionType = "VERIFICATION"
	TypeVoidAuthorization   TransactionType = "VOID_AUTHORIZATION"
	TypeVoidCapture         TransactionType = "VOID_CAPTURE"
	TypeVoidPayment         TransactionType = "VOID_PAYMENT"
	TypeVoidRefund          TransactionType = "VOID_REFUND"
)
