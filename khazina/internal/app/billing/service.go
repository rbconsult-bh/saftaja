package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rs/zerolog/log"
)

type service struct {
	db              *pgxpool.Pool
	queries         store.TransactionQuerier
	gatewayResolver GatewayResolver
}

func New(db *pgxpool.Pool, queries store.TransactionQuerier, gatewayResolver GatewayResolver) Service {
	return &service{
		db:              db,
		queries:         queries,
		gatewayResolver: gatewayResolver,
	}
}

func (s *service) GetInvoice(ctx context.Context, r GetInvoiceRequest) (*GetInvoiceResponse, error) {
	if err := r.Validate(); err != nil {
		log.Ctx(ctx).Info().Err(err).Msg("validation failed")
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
		log.Ctx(ctx).Info().Err(err).Msg("validation failed")
		return nil, err
	}

	existingIntent, err := s.queries.GetPaymentIntentByIdempotencyKeyAndProject(ctx, store.GetPaymentIntentByIdempotencyKeyAndProjectParams{
		ProjectID:      r.ProjectID,
		IdempotencyKey: r.IdempotencyKey,
	})
	switch {
	case err == nil:
		existingPaymentMethod, err := mapStorePaymentMethodToPaymentMethod(existingIntent.PaymentMethod)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment method to payment method")
			return nil, err
		}

		if existingIntent.InvoiceID != r.InvoiceID ||
			existingIntent.GatewayAccountID != r.GatewayAccountID ||
			existingPaymentMethod != r.PaymentMethod {
			log.Ctx(ctx).Warn().Msg("idempotency key collision with different parameters")
			return nil, ErrIdempotencyMismatch
		}

		if time.Now().After(existingIntent.ExpiresAt) {
			log.Ctx(ctx).Error().Msg("payment intent already expired")
			return nil, ErrPaymentIntentExpired
		}

		log.Ctx(ctx).Info().Msg("returning idempotent response")
		return &StartPaymentResponse{
			PaymentIntentID:  existingIntent.ID,
			GatewaySessionID: existingIntent.GatewaySessionID,
		}, nil
	case errors.Is(err, pgx.ErrNoRows):
		// we are good :)
	default:
		log.Ctx(ctx).Error().Err(err).Msg("failed to get payment intent by idempotency key")
		return nil, err
	}

	invoice, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        r.InvoiceID,
		ProjectID: r.ProjectID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

	gatewayAccount, err := s.queries.GetGatewayAccountByIDAndProject(ctx, store.GetGatewayAccountByIDAndProjectParams{
		ID:        r.GatewayAccountID,
		ProjectID: r.ProjectID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Ctx(ctx).Info().Msg("gateway account is not found")
			return nil, ErrGatewayAccountNotFound
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get gateway account by id and project")
		return nil, err
	}

	createPaymentIntentParams := store.CreatePaymentIntentParams{
		InvoiceID:        invoice.ID,
		ProjectID:        invoice.ProjectID,
		GatewayAccountID: gatewayAccount.ID,
		PayerIp:          r.PayerIP,
		PayerUserAgent:   r.PayerUserAgent,
		IdempotencyKey:   r.IdempotencyKey,
	}
	switch r.PaymentMethod {
	case PaymentMethodCard:
		cardGateway, err := s.gatewayResolver.CardGateway(gatewayAccount)
		if err != nil {
			if errors.Is(err, ErrUnsupportedGateway) {
				log.Ctx(ctx).Info().
					Err(err).
					Str("connector_type", string(gatewayAccount.ConnectorType)).
					Str("payment_method", string(r.PaymentMethod)).
					Msg("gateway account does not support payment method")
				return nil, fmt.Errorf("%w: card is not supported by selected gateway account", ErrUnsupportedPaymentMethod)
			}

			log.Ctx(ctx).Error().
				Err(err).
				Str("gateway_account_id", gatewayAccount.ID.String()).
				Str("connector_type", string(gatewayAccount.ConnectorType)).
				Msg("failed to resolve card gateway")
			return nil, err
		}

		createPaymentIntentParams.PaymentMethod = store.PaymentMethodCard

		createSessionResp, err := cardGateway.CreateSession(ctx, CreateSessionRequest{
			InvoiceID: invoice.ID,
			Amount:    invoice.Amount,
			Currency:  invoice.Currency,
		})
		if err != nil {
			log.Ctx(ctx).Error().
				Err(err).
				Str("gateway_account_id", gatewayAccount.ID.String()).
				Str("invoice_id", invoice.ID.String()).
				Msg("failed to create card gateway session")
			return nil, err
		}

		createPaymentIntentParams.GatewaySessionID = &createSessionResp.GatewaySessionID

	case PaymentMethodApplePay:
		log.Ctx(ctx).Info().
			Str("payment_method", string(r.PaymentMethod)).
			Msg("payment method is not yet supported")
		return nil, fmt.Errorf("%w: Apple Pay not yet supported", ErrUnsupportedPaymentMethod)

	default:
		log.Ctx(ctx).Info().
			Str("payment_method", string(r.PaymentMethod)).
			Msg("unsupported payment method")
		return nil, ErrUnsupportedPaymentMethod
	}

	resp, err := s.queries.CreatePaymentIntent(ctx, createPaymentIntentParams)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create payment intent")
		return nil, err
	}

	return &StartPaymentResponse{
		PaymentIntentID:  resp.ID,
		GatewaySessionID: resp.GatewaySessionID,
	}, nil
}

func (s *service) VerifyCard(ctx context.Context, r VerifyCardRequest) (*VerifyCardResponse, error) {
	if err := r.Validate(); err != nil {
		log.Ctx(ctx).Info().Err(err).Msg("validation failed")
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to begin tx")
		return nil, errors.New("failed to begin tx")
	}
	defer tx.Rollback(ctx)

	queriesWithTx := s.queries.WithTx(tx)

	paymentIntent, err := queriesWithTx.GetPaymentIntentByIDAndProjectAndInvoiceForUpdate(
		ctx,
		store.GetPaymentIntentByIDAndProjectAndInvoiceForUpdateParams{
			ID:        r.PaymentIntentID,
			ProjectID: r.ProjectID,
			InvoiceID: r.InvoiceID,
		},
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Ctx(ctx).Info().Msg("payment intent not found")
			return nil, ErrNotFound
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get payment intent by id and project for update")
		return nil, err
	}

	if paymentIntent.ExpiresAt.Before(time.Now()) {
		log.Ctx(ctx).Info().Msg("payment intent expired")
		return nil, ErrPaymentIntentExpired
	}

	paymentIntentStatus, err := mapStorePaymentIntentStatusToPaymentIntentStatus(paymentIntent.Status)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment intent status to payment intent status")
		return nil, err
	}

	paymentMethod, err := mapStorePaymentMethodToPaymentMethod(paymentIntent.PaymentMethod)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment method to payment method")
		return nil, err
	}

	if err := validatePaymentIntentTransition(paymentMethod, paymentIntentStatus, PaymentIntentStatusVerifyingCard); err != nil {
		log.Ctx(ctx).Info().Err(err).Msg("payment intent invalid transition")
		return nil, err
	}

	// TODO: use gatewayResolver to verify card

	return &VerifyCardResponse{}, nil
}

func (s *service) ChallengeCard(ctx context.Context, r ChallengeCardRequest) (*ChallengeCardResponse, error) {
	return &ChallengeCardResponse{}, nil
}

func (s *service) CapturePayment(ctx context.Context, r CapturePaymentRequest) (*CapturePaymentResponse, error) {
	return &CapturePaymentResponse{}, nil
}
