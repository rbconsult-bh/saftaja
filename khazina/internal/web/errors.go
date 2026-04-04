package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/payment"
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

func respondError(w http.ResponseWriter, r *http.Request, code ErrorCode, msg domain.LocalizedString, status int) {
	lang := domain.DetectLanguage(r)

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
		respondError(w, r, ErrCodeInvoiceNotFound, domain.MsgInvoiceNotFound, http.StatusNotFound)
	case errors.As(err, &sessionExpired):
		respondError(w, r, ErrCodeSessionExpired, domain.MsgSessionExpired, http.StatusGone)
	case errors.As(err, &invalidTransition):
		respondError(w, r, ErrCodeInvalidState, domain.MsgInvalidState, http.StatusConflict)
	case errors.As(err, &alreadyPaid):
		respondError(w, r, ErrCodeAlreadyPaid, domain.MsgAlreadyPaid, http.StatusConflict)
	case errors.As(err, &mismatch):
		respondError(w, r, ErrCodeSessionMismatch, domain.MsgInvalidState, http.StatusForbidden)
	case errors.As(err, &gatewayErr):
		log.Ctx(ctx).Error().Err(err).Msg("gateway error")
		respondError(w, r, ErrCodeGatewayError, domain.MsgGatewayError, http.StatusBadGateway)
	default:
		log.Ctx(ctx).Error().Err(err).Msg("internal error")
		respondError(w, r, ErrCodeInternalError, domain.MsgInternalError, http.StatusInternalServerError)
	}
}

var PaymentResultMessages = map[payment.PaymentResultCode]domain.LocalizedString{
	payment.ResultSuccess:      domain.MsgPaymentSuccessful,
	payment.ResultDeclined:     domain.MsgPaymentDeclined,
	payment.ResultAuthFailed:   domain.MsgAuthenticationFailed,
	payment.ResultGatewayError: domain.MsgGatewayError,
}
