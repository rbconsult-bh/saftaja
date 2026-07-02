package billing

import (
	"time"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

func mapStoreInvoiceRowsToInvoice(rows []store.GetInvoiceWithItemsByIDAndProjectIDRow) Invoice {
	if len(rows) == 0 {
		return Invoice{}
	}

	first := rows[0].Invoice

	var paidAt *time.Time
	if first.PaidAt.Valid {
		paidAt = &first.PaidAt.Time
	}

	var deletedAt *time.Time
	if first.DeletedAt.Valid {
		deletedAt = &first.DeletedAt.Time
	}

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
		PaidAt:        paidAt,
		CreatedAt:     first.CreatedAt.Time,
		UpdatedAt:     first.UpdatedAt.Time,
		DeletedAt:     deletedAt,
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
			CreatedAt:   r.InvoiceItem.CreatedAt.Time,
		}

		invoice.Items = append(invoice.Items, item)
	}

	return invoice
}

// TODO: use store type once you make it.
func mapStoreInvoiceStatusToInvoiceStatus(is store.InvoiceStatus) InvoiceStatus {
	switch is {
	case store.InvoiceStatusPending:
		return InvoiceStatusPending
	case store.InvoiceStatusPaid:
		return InvoiceStatusPaid
	case store.InvoiceStatusFailed:
		return InvoiceStatusFailed
	default:
		return InvoiceStatusUnknown
	}
}
