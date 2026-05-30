package billing

import (
	"context"
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rs/zerolog/log"
)

type service struct {
	queries store.TransactionQuerier
}

func New(queries store.TransactionQuerier) Service {
	return &service{
		queries: queries,
	}
}

func (s *service) GetInvoice(ctx context.Context, r GetInvoiceRequest) (*GetInvoiceResponse, error) {
	if err := r.Validate(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("validation failed")
		return nil, err
	}

	rows, err := s.queries.GetInvoiceWithItemsByIDAndProjectID(ctx, store.GetInvoiceWithItemsByIDAndProjectIDParams{
		ID:        r.InvoiceID,
		ProjectID: r.ProjectID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed calling GetInvoiceWithItemsByIDAndProjectID")
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if len(rows) == 0 {
		log.Ctx(ctx).Info().Msg("invoice does not exist or belong to project")
		return nil, ErrNotFound
	}

	return &GetInvoiceResponse{
		Invoice: mapStoreInvoiceRowsToInvoice(rows),
	}, nil
}

func (s *service) ListPaymentMethods(ctx context.Context, r ListPaymentMethodsRequest) (*ListPaymentMethodsResponse, error) {
	return &ListPaymentMethodsResponse{}, nil
}

func (s *service) StartPayment(ctx context.Context, r StartPaymentRequest) (*StartPaymentResponse, error) {
	return &StartPaymentResponse{}, nil
}

func (s *service) VerifyCard(ctx context.Context, r VerifyCardRequest) (*VerifyCardResponse, error) {
	return &VerifyCardResponse{}, nil
}

func (s *service) ChallengeCard(ctx context.Context, r ChallengeCardRequest) (*ChallengeCardResponse, error) {
	return &ChallengeCardResponse{}, nil
}

func (s *service) CapturePayment(ctx context.Context, r CapturePaymentRequest) (*CapturePaymentResponse, error) {
	return &CapturePaymentResponse{}, nil
}
