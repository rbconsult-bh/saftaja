package billing

import (
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

func mapStoreInvoiceRowsToInvoice(rows []store.GetInvoiceWithItemsByIDAndProjectIDRow) Invoice {
	if len(rows) == 0 {
		return Invoice{}
	}

	first := rows[0].Invoice

	invoice := Invoice{
		ID:            first.ID,
		ProjectID:     first.ProjectID,
		Amount:        first.Amount,
		Currency:      first.Currency,
		Status:        mapStoreInvoiceStatusToInvoiceStatus(first.Status),
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

	return invoice
}

func mapStoreInvoiceStatusToInvoiceStatus(is store.InvoiceStatus) InvoiceStatus {
	switch is {
	case store.InvoiceStatusPending:
		return InvoiceStatusPending
	case store.InvoiceStatusPaid:
		return InvoiceStatusPaid
	case store.InvoiceStatusCancelled:
		return InvoiceStatusCancelled
	default:
		return InvoiceStatusUnknown
	}
}

func mapStorePaymentMethodToPaymentMethod(spm store.PaymentMethod) PaymentMethod {
	switch spm {
	case store.PaymentMethodCard:
		return PaymentMethodCard
	case store.PaymentMethodApplePay:
		return PaymentMethodApplePay
	default:
		return PaymentMethodUnkown
	}
}

func mapStorePaymentIntentStatusToPaymentIntentStatus(s store.PaymentIntentStatus) (PaymentIntentStatus, error) {
	switch s {
	case store.PaymentIntentStatusCreated:
		return PaymentIntentStatusCreated, nil
	case store.PaymentIntentStatusVerifyingCard:
		return PaymentIntentStatusVerifyingCard, nil
	case store.PaymentIntentStatusCardVerified:
		return PaymentIntentStatusCardVerified, nil
	case store.PaymentIntentStatusProcessingPayment:
		return PaymentIntentStatusProcessingPayment, nil
	case store.PaymentIntentStatusCompleted:
		return PaymentIntentStatusCompleted, nil
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
	case PaymentIntentStatusVerifyingCard:
		return store.PaymentIntentStatusVerifyingCard, nil
	case PaymentIntentStatusCardVerified:
		return store.PaymentIntentStatusCardVerified, nil
	case PaymentIntentStatusProcessingPayment:
		return store.PaymentIntentStatusProcessingPayment, nil
	case PaymentIntentStatusCompleted:
		return store.PaymentIntentStatusCompleted, nil
	case PaymentIntentStatusFailed:
		return store.PaymentIntentStatusFailed, nil
	default:
		return "", fmt.Errorf("%w: unknown payment intent status %q", ErrPaymentIntentInvalidState, s)
	}
}
