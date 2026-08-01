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
	ErrCodePaymentExpired  ErrorCode = "PAYMENT_INTENT_EXPIRED"
	ErrCodeInvalidState    ErrorCode = "INVALID_STATE"
	ErrCodeAlreadyPaid     ErrorCode = "ALREADY_PAID"
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

	switch {
	case errors.Is(err, payment.ErrInvalidArgument):
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
	case errors.Is(err, payment.ErrNotFound), errors.Is(err, payment.ErrInvoiceNotFound):
		respondError(w, r, ErrCodeInvoiceNotFound, templfiles.MsgInvoiceNotFound, http.StatusNotFound)
	case errors.Is(err, payment.ErrPaymentIntentExpired):
		respondError(w, r, ErrCodePaymentExpired, templfiles.MsgSessionExpired, http.StatusGone)
	case errors.Is(err, payment.ErrPaymentIntentInvalidState),
		errors.Is(err, payment.ErrPaymentIntentInvalidTransition),
		errors.Is(err, payment.ErrInvoiceInvalidState),
		errors.Is(err, payment.ErrInvoiceCancelled):
		respondError(w, r, ErrCodeInvalidState, templfiles.MsgInvalidState, http.StatusConflict)
	case errors.Is(err, payment.ErrInvoiceAlreadyPaid), errors.Is(err, payment.ErrInvoicePaymentInProgress):
		respondError(w, r, ErrCodeAlreadyPaid, templfiles.MsgAlreadyPaid, http.StatusConflict)
	case errors.Is(err, payment.ErrUnsupportedPaymentMethod):
		respondError(w, r, ErrCodeInvalidRequest, templfiles.MsgInvalidRequest, http.StatusBadRequest)
	default:
		log.Ctx(ctx).Error().Err(err).Msg("internal error")
		respondError(w, r, ErrCodeInternalError, templfiles.MsgInternalError, http.StatusInternalServerError)
	}
}
