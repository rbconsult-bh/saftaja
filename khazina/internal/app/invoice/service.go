package invoice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
	GetByID(ctx context.Context, invoiceID, projectID uuid.UUID) (*InvoiceData, error)
}

type service struct {
	queries store.TransactionQuerier
}

func New(queries store.TransactionQuerier) Service {
	return &service{queries: queries}
}

func (s *service) GetByID(ctx context.Context, invoiceID, projectID uuid.UUID) (*InvoiceData, error) {
	inv, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        invoiceID,
		ProjectID: projectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if inv.Status == domain.InvoiceStatusPaid {
		return &InvoiceData{
			Invoice: InvoiceInfo{
				ID:        inv.ID,
				ProjectID: inv.ProjectID,
				Amount:    inv.Amount.String(),
				Currency:  inv.Currency,
			},
			IsPaid: true,
		}, nil
	}

	items, err := s.queries.GetInvoiceItems(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice items: %w", err)
	}

	itemInfos := make([]ItemInfo, len(items))
	for i, item := range items {
		itemInfos[i] = ItemInfo{
			Name:      item.Name,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice.String(),
			Amount:    item.Amount.String(),
		}
	}

	return &InvoiceData{
		Invoice: InvoiceInfo{
			ID:            inv.ID,
			ProjectID:     inv.ProjectID,
			Amount:        inv.Amount.String(),
			Currency:      inv.Currency,
			Description:   inv.Description.String,
			CustomerEmail: inv.CustomerEmail.String,
			CustomerName:  inv.CustomerName.String,
		},
		Items:  itemInfos,
		IsPaid: false,
	}, nil
}
