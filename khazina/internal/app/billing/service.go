package billing

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
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

func (s *service) StartPayment(ctx context.Context, r StartPaymentRequest) (*StartPaymentResponse, error) {
	if err := r.Validate(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("validation failed")
		return nil, err
	}

	// TODO: do idempotency checks

	invoice, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        r.InvoiceID,
		ProjectID: r.ProjectID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Ctx(ctx).Info().Msg("invoice is not found")
			return nil, ErrInvoiceNotFound
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id and project")
		return nil, err
	}

	switch invoice.Status {
	case store.InvoiceStatusPending:
		// we are good :)
	case store.InvoiceStatusPaid:
		log.Ctx(ctx).Info().Msg("attempted to pay an already paid invoice")
		return nil, ErrInvoiceAlreadyPaid
	case store.InvoiceStatusCancelled:
		log.Ctx(ctx).Info().Msg("cannot pay cancelled invoice")
		return nil, ErrInvoiceCancelled
	default:
		return nil, fmt.Errorf("un-payable invoice status: %s", invoice.Status)
	}

	_, err = s.queries.GetGatewayAccountByIDAndProject(ctx, store.GetGatewayAccountByIDAndProjectParams{
		ID:        r.GatewayAccountID,
		ProjectID: r.ProjectID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Ctx(ctx).Info().Msg("gatewat account is not found")
			return nil, ErrGatewayAccountNotFound
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get gateway account by id and project")
		return nil, err
	}

	createPaymentIntentParams := store.CreatePaymentIntentParams{
		InvoiceID:        r.InvoiceID,
		ProjectID:        r.ProjectID,
		GatewayAccountID: r.GatewayAccountID,
		PayerIp:          r.PayerIP,
		PayerUserAgent:   r.PayerUserAgent,
		IdempotencyKey:   r.IdempotencyKey,
	}
	switch r.PaymentMethod {
	case PaymentMethodCard:
		createPaymentIntentParams.PaymentMethod = store.PaymentMethodCard
		createPaymentIntentParams.GatewaySessionID = new(string)
		// TODO: create mpgs session
	case PaymentMethodApplePay:
		createPaymentIntentParams.PaymentMethod = store.PaymentMethodApplePay
		// apple pay through does not need external services before we get the token from the user.
		// TODO: handle apple pay
		return nil, fmt.Errorf("%w: Apply Pay not yet supported", ErrUnsupportedPaymentMethod)
	default:
		return nil, ErrUnsupportedPaymentMethod
	}

	resp, err := s.queries.CreatePaymentIntent(ctx, createPaymentIntentParams)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create payment intent")
		return nil, err
	}

	return &StartPaymentResponse{
		PaymentIntentID:  resp.ID.String(),
		GatewaySessionID: resp.GatewaySessionID,
	}, nil
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
