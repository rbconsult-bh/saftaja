package payment

import (
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

func mapStoreInvoiceRowsToInvoice(rows []store.GetInvoiceWithItemsByIDAndProjectIDRow) (Invoice, error) {
	if len(rows) == 0 {
		return Invoice{}, nil
	}

	first := rows[0].Invoice
	status, err := mapStoreInvoiceStatusToInvoiceStatus(first.Status)
	if err != nil {
		return Invoice{}, err
	}

	invoice := Invoice{
		ID:            first.ID,
		ProjectID:     first.ProjectID,
		Amount:        first.Amount,
		Currency:      first.Currency,
		Status:        status,
		ExternalID:    ptr.Deref(first.ExternalID),
		CustomerEmail: ptr.Deref(first.CustomerEmail),
		CustomerName:  ptr.Deref(first.CustomerName),
		Description:   ptr.Deref(first.Description),
		PaidAt:        first.PaidAt,
		CreatedAt:     first.CreatedAt,
		UpdatedAt:     first.UpdatedAt,
		DeletedAt:     first.DeletedAt,
		Items:         make([]InvoiceItem, 0, len(rows)),
	}

	for _, r := range rows {
		item := InvoiceItem{
			ID:          r.InvoiceItem.ID,
			InvoiceID:   r.InvoiceItem.InvoiceID,
			Name:        r.InvoiceItem.Name,
			Description: ptr.Deref(r.InvoiceItem.Description),
			Quantity:    r.InvoiceItem.Quantity,
			UnitPrice:   r.InvoiceItem.UnitPrice,
			Amount:      r.InvoiceItem.Amount,
			CreatedAt:   r.InvoiceItem.CreatedAt,
		}

		invoice.Items = append(invoice.Items, item)
	}

	return invoice, nil
}

func mapStoreInvoiceStatusToInvoiceStatus(is store.InvoiceStatus) (InvoiceStatus, error) {
	switch is {
	case store.InvoiceStatusPending:
		return InvoiceStatusPending, nil
	case store.InvoiceStatusPaid:
		return InvoiceStatusPaid, nil
	case store.InvoiceStatusCancelled:
		return InvoiceStatusCancelled, nil
	default:
		return "", fmt.Errorf("%w: unknown invoice status %q", ErrInvoiceInvalidState, is)
	}
}

func mapStorePaymentMethodToPaymentMethod(spm store.PaymentMethod) (PaymentMethod, error) {
	switch spm {
	case store.PaymentMethodCard:
		return PaymentMethodCard, nil
	case store.PaymentMethodApplePay:
		return PaymentMethodApplePay, nil
	default:
		return "", fmt.Errorf("%w: unknown payment method %q", ErrPaymentIntentInvalidState, spm)
	}
}

func mapStorePaymentMethodReferenceToPaymentMethodReference(reference *string) *PaymentMethodReference {
	if reference == nil {
		return nil
	}

	paymentMethodReference := PaymentMethodReference(*reference)
	return &paymentMethodReference
}

func mapStorePaymentIntentStatusToPaymentIntentStatus(s store.PaymentIntentStatus) (PaymentIntentStatus, error) {
	switch s {
	case store.PaymentIntentStatusCreated:
		return PaymentIntentStatusCreated, nil
	case store.PaymentIntentStatusReadyToAuthenticate:
		return PaymentIntentStatusReadyToAuthenticate, nil
	case store.PaymentIntentStatusAwaitingAuthenticationResult:
		return PaymentIntentStatusAwaitingAuthenticationResult, nil
	case store.PaymentIntentStatusReadyToCapture:
		return PaymentIntentStatusReadyToCapture, nil
	case store.PaymentIntentStatusCapturing:
		return PaymentIntentStatusCapturing, nil
	case store.PaymentIntentStatusSucceeded:
		return PaymentIntentStatusSucceeded, nil
	case store.PaymentIntentStatusFailed:
		return PaymentIntentStatusFailed, nil
	default:
		return "", fmt.Errorf("%w: unknown payment intent status %q", ErrPaymentIntentInvalidState, s)
	}
}

func mapPaymentIntentStatusToStorePaymentIntentStatus(s PaymentIntentStatus) (store.PaymentIntentStatus, error) {
	switch s {
	case PaymentIntentStatusCreated:
		return store.PaymentIntentStatusCreated, nil
	case PaymentIntentStatusReadyToAuthenticate:
		return store.PaymentIntentStatusReadyToAuthenticate, nil
	case PaymentIntentStatusAwaitingAuthenticationResult:
		return store.PaymentIntentStatusAwaitingAuthenticationResult, nil
	case PaymentIntentStatusReadyToCapture:
		return store.PaymentIntentStatusReadyToCapture, nil
	case PaymentIntentStatusCapturing:
		return store.PaymentIntentStatusCapturing, nil
	case PaymentIntentStatusSucceeded:
		return store.PaymentIntentStatusSucceeded, nil
	case PaymentIntentStatusFailed:
		return store.PaymentIntentStatusFailed, nil
	default:
		return "", fmt.Errorf("%w: unknown payment intent status %q", ErrPaymentIntentInvalidState, s)
	}
}
