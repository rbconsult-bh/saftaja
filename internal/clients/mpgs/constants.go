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
