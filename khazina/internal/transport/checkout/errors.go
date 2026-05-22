package checkout

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/checkout/templfiles"
)

type ErrorCode string

const (
	ErrCodeInvalidRequest  ErrorCode = "INVALID_REQUEST"
	ErrCodeInvoiceNotFound ErrorCode = "INVOICE_NOT_FOUND"
	ErrCodeSessionExpired  ErrorCode = "SESSION_EXPIRED"
	ErrCodeSessionNotFound ErrorCode = "SESSION_NOT_FOUND"
	ErrCodeInvalidState    ErrorCode = "INVALID_STATE"
	ErrCodeAlreadyPaid     ErrorCode = "ALREADY_PAID"
	ErrCodeSessionMismatch ErrorCode = "SESSION_MISMATCH"
	ErrCodeGatewayError    ErrorCode = "GATEWAY_ERROR"
	ErrCodeInternalError   ErrorCode = "INTERNAL_ERROR"
)

type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func respondError(w http.ResponseWriter, r *http.Request, code ErrorCode, msg templfiles.LocalizedString, status int) {
	lang := templfiles.DetectLanguage(r)

	resp := APIError{
		Code:    code,
		Message: msg.Get(lang),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

func handlePaymentError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	var sessionExpired *payment.SessionExpiredError
	var invalidTransition *payment.InvalidStateTransitionError
	var alreadyPaid *payment.InvoiceAlreadyPaidError
	var mismatch *payment.SessionInvoiceMismatchError
	var gatewayErr *payment.GatewayError
	var notFound *payment.InvoiceNotFoundError

	switch {
	case errors.As(err, &notFound):
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
	case errors.As(err, &sessionExpired):
		respondError(w, r, ErrCodeSessionExpired, templfiles.MsgSessionExpired, http.StatusGone)
	case errors.As(err, &invalidTransition):
		respondError(w, r, ErrCodeInvalidState, templfiles.MsgInvalidState, http.StatusConflict)
	case errors.As(err, &alreadyPaid):
		respondError(w, r, ErrCodeAlreadyPaid, templfiles.MsgAlreadyPaid, http.StatusConflict)
	case errors.As(err, &mismatch):
		respondError(w, r, ErrCodeSessionMismatch, templfiles.MsgInvalidState, http.StatusForbidden)
	case errors.As(err, &gatewayErr):
		log.Ctx(ctx).Error().Err(err).Msg("gateway error")
		respondError(w, r, ErrCodeGatewayError, templfiles.MsgGatewayError, http.StatusBadGateway)
	default:
		log.Ctx(ctx).Error().Err(err).Msg("internal error")
		respondError(w, r, ErrCodeInternalError, templfiles.MsgInternalError, http.StatusInternalServerError)
	}
}

var PaymentResultMessages = map[payment.PaymentResultCode]templfiles.LocalizedString{
	payment.ResultSuccess:      templfiles.MsgPaymentSuccessful,
	payment.ResultDeclined:     templfiles.MsgPaymentDeclined,
	payment.ResultAuthFailed:   templfiles.MsgAuthenticationFailed,
	payment.ResultGatewayError: templfiles.MsgGatewayError,
}
