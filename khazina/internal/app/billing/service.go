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

	invoice, err := mapStoreInvoiceRowsToInvoice(rows)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map invoice")
		return nil, err
	}

	return &GetInvoiceResponse{
		Invoice: invoice,
	}, nil
}

func (s *service) CreatePaymentIntent(ctx context.Context, r CreatePaymentIntentRequest) (*CreatePaymentIntentResponse, error) {
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
		return &CreatePaymentIntentResponse{
			PaymentIntentID:        existingIntent.ID,
			PaymentMethodReference: mapStorePaymentMethodReferenceToPaymentMethodReference(existingIntent.GatewaySetupReference),
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
		log.Ctx(ctx).Error().Str("invoice_status", string(invoice.Status)).Msg("unknown invoice status")
		return nil, fmt.Errorf("%w: unknown invoice status %q", ErrInvoiceInvalidState, invoice.Status)
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

		setupResp, err := cardGateway.SetupCardPaymentMethod(ctx, SetupCardPaymentMethodGatewayRequest{
			InvoiceID: invoice.ID,
			Amount:    invoice.Amount,
			Currency:  invoice.Currency,
		})
		if err != nil {
			log.Ctx(ctx).Error().
				Err(err).
				Str("gateway_account_id", gatewayAccount.ID.String()).
				Str("invoice_id", invoice.ID.String()).
				Msg("failed to set up card payment method")
			return nil, err
		}
		storedPaymentMethodReference := string(setupResp.PaymentMethodReference)
		createPaymentIntentParams.GatewaySetupReference = &storedPaymentMethodReference

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

	return &CreatePaymentIntentResponse{
		PaymentIntentID:        resp.ID,
		PaymentMethodReference: mapStorePaymentMethodReferenceToPaymentMethodReference(resp.GatewaySetupReference),
	}, nil
}

func (s *service) CapturePaymentIntent(ctx context.Context, r CapturePaymentIntentRequest) (*CapturePaymentIntentResponse, error) {
	return &CapturePaymentIntentResponse{}, nil
}

func (s *service) PrepareCardAuthentication(ctx context.Context, r PrepareCardAuthenticationRequest) (*PrepareCardAuthenticationResponse, error) {
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

	paymentIntent, err := queriesWithTx.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdate(
		ctx,
		store.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdateParams{
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

		log.Ctx(ctx).Error().Err(err).Msg("failed to lock payment intent")
		return nil, err
	}

	currentStatus, err := mapStorePaymentIntentStatusToPaymentIntentStatus(paymentIntent.Status)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment intent status to payment intent status")
		return nil, err
	}

	paymentMethod, err := mapStorePaymentMethodToPaymentMethod(paymentIntent.PaymentMethod)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment method to payment method")
		return nil, err
	}

	if err := currentStatus.ValidatePaymentIntentTransition(paymentMethod, PaymentIntentStatusReadyToAuthenticate); err != nil {
		log.Ctx(ctx).Info().Err(err).Msg("payment intent invalid transition")
		return nil, err
	}

	if paymentIntent.ExpiresAt.Before(time.Now()) {
		log.Ctx(ctx).Info().Msg("payment intent expired")
		return nil, ErrPaymentIntentExpired
	}

	if paymentIntent.GatewaySetupReference == nil || *paymentIntent.GatewaySetupReference == "" {
		log.Ctx(ctx).Error().Msg("payment intent is missing payment method reference")
		return nil, fmt.Errorf("%w: payment intent is missing payment method reference", ErrPaymentIntentInvalidState)
	}

	invoice, err := queriesWithTx.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        paymentIntent.InvoiceID,
		ProjectID: paymentIntent.ProjectID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id and project")
		return nil, err
	}

	gatewayAccount, err := queriesWithTx.GetGatewayAccountByIDAndProject(ctx, store.GetGatewayAccountByIDAndProjectParams{
		ID:        paymentIntent.GatewayAccountID,
		ProjectID: paymentIntent.ProjectID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get gateway account")
		return nil, err
	}

	cardGateway, err := s.gatewayResolver.CardGateway(gatewayAccount)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to resolve card gateway by gateway account")
		return nil, err
	}

	prepared, err := cardGateway.PrepareCardAuthentication(ctx, PrepareCardAuthenticationGatewayRequest{
		InvoiceID:              invoice.ID,
		Currency:               invoice.Currency,
		PaymentMethodReference: PaymentMethodReference(*paymentIntent.GatewaySetupReference),
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to prepare card authentication")
		return nil, err
	}

	gatewayOperation, err := s.queries.CreateGatewayOperation(ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  paymentIntent.ID,
		InvoiceID:        invoice.ID,
		ProjectID:        paymentIntent.ProjectID,
		GatewayAccountID: paymentIntent.GatewayAccountID,
		OperationType:    store.GatewayOperationTypeInitiateAuth,
		GatewayReference: string(prepared.AuthenticationReference),
		Amount:           invoice.Amount,
		Currency:         invoice.Currency,
		RawRequest:       prepared.RawRequest,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create gateway operation")
		return nil, err
	}

	updateGatewayOperationStatus := func(status store.GatewayOperationStatus, rawResponse []byte) error {
		return s.queries.UpdateGatewayOperationStatus(ctx, store.UpdateGatewayOperationStatusParams{
			ID:          gatewayOperation.ID,
			Status:      status,
			RawResponse: rawResponse,
		})
	}

	gatewayResp, err := prepared.Send(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to prepare card authentication with gateway")
		if statusErr := updateGatewayOperationStatus(store.GatewayOperationStatusFailed, nil); statusErr != nil {
			log.Ctx(ctx).Error().Err(statusErr).Msg("failed to update gateway operation status after prepare card authentication error")
		}
		return nil, err
	}

	if gatewayResp.NextStep != PrepareCardAuthenticationGatewayNextStepAuthenticate {
		// The gateway call completed successfully, but its decision was "can't continue."
		// Keep the payment intent in created so the customer can update the card and
		// retry PrepareCardAuthentication. The raw gateway decision is stored on this operation.
		if err := updateGatewayOperationStatus(store.GatewayOperationStatusSuccess, gatewayResp.RawResponse); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
			return nil, err
		}
		return &PrepareCardAuthenticationResponse{
			NextStep: PrepareCardAuthenticationNextStepCantContinue,
		}, nil
	}

	if err := updateGatewayOperationStatus(store.GatewayOperationStatusSuccess, gatewayResp.RawResponse); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
		return nil, err
	}

	if err := queriesWithTx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
		ID:     paymentIntent.ID,
		Status: store.PaymentIntentStatusReadyToAuthenticate,
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update payment intent status")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to commit prepared card authentication")
		return nil, err
	}

	return &PrepareCardAuthenticationResponse{
		NextStep: PrepareCardAuthenticationNextStepAuthenticate,
	}, nil
}

func (s *service) AuthenticateCardholder(ctx context.Context, r AuthenticateCardholderRequest) (*AuthenticateCardholderResponse, error) {
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

	paymentIntent, err := queriesWithTx.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdate(
		ctx,
		store.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdateParams{
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

		log.Ctx(ctx).Error().Err(err).Msg("failed to lock payment intent")
		return nil, err
	}

	currentStatus, err := mapStorePaymentIntentStatusToPaymentIntentStatus(paymentIntent.Status)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment intent status to payment intent status")
		return nil, err
	}
	paymentMethod, err := mapStorePaymentMethodToPaymentMethod(paymentIntent.PaymentMethod)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment method to payment method")
		return nil, err
	}

	if currentStatus != PaymentIntentStatusReadyToAuthenticate {
		err := fmt.Errorf(
			"%w: AuthenticateCardholder requires %s, got %s",
			ErrPaymentIntentInvalidState,
			PaymentIntentStatusReadyToAuthenticate,
			currentStatus,
		)

		log.Ctx(ctx).Info().
			Err(err).
			Msg("payment intent invalid status")

		return nil, err
	}

	if paymentIntent.ExpiresAt.Before(time.Now()) {
		log.Ctx(ctx).Info().Msg("payment intent expired")
		return nil, ErrPaymentIntentExpired
	}

	if paymentIntent.GatewaySetupReference == nil || *paymentIntent.GatewaySetupReference == "" {
		log.Ctx(ctx).Error().Msg("payment intent is missing payment method reference")
		return nil, fmt.Errorf("%w: payment intent is missing payment method reference", ErrPaymentIntentInvalidState)
	}

	invoice, err := queriesWithTx.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        paymentIntent.InvoiceID,
		ProjectID: paymentIntent.ProjectID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice by id and project")
		return nil, err
	}

	gatewayAccount, err := queriesWithTx.GetGatewayAccountByIDAndProject(ctx, store.GetGatewayAccountByIDAndProjectParams{
		ID:        paymentIntent.GatewayAccountID,
		ProjectID: paymentIntent.ProjectID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get gateway account")
		return nil, err
	}

	cardGateway, err := s.gatewayResolver.CardGateway(gatewayAccount)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to resolve card gateway by gateway account")
		return nil, err
	}

	initiateAuthOperation, err := s.queries.GetSuccessfulInitiateAuthGatewayOperation(ctx, paymentIntent.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Ctx(ctx).Error().Msg("successful initiate authentication gateway operation not found")
			return nil, fmt.Errorf(
				"%w: successful initiate authentication gateway operation is missing",
				ErrPaymentIntentInvalidState,
			)
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get successful initiate authentication gateway operation")
		return nil, err
	}
	if initiateAuthOperation.GatewayReference == "" {
		log.Ctx(ctx).Error().Msg("initiate authentication gateway operation is missing gateway reference")
		return nil, fmt.Errorf(
			"%w: initiate authentication gateway operation is missing gateway reference",
			ErrPaymentIntentInvalidState,
		)
	}

	prepared, err := cardGateway.AuthenticateCardholder(ctx, AuthenticateCardholderGatewayRequest{
		InvoiceID:               invoice.ID,
		Amount:                  invoice.Amount,
		Currency:                invoice.Currency,
		PaymentMethodReference:  PaymentMethodReference(*paymentIntent.GatewaySetupReference),
		AuthenticationReference: AuthenticationReference(initiateAuthOperation.GatewayReference),
		ChallengeReturnURL:      r.ChallengeReturnURL,
		Browser:                 r.Browser,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to prepare authenticate cardholder request")
		return nil, err
	}

	gatewayOperation, err := s.queries.CreateGatewayOperation(ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  paymentIntent.ID,
		InvoiceID:        invoice.ID,
		ProjectID:        paymentIntent.ProjectID,
		GatewayAccountID: paymentIntent.GatewayAccountID,
		OperationType:    store.GatewayOperationTypeAuthenticatePayer,
		GatewayReference: initiateAuthOperation.GatewayReference,
		Amount:           invoice.Amount,
		Currency:         invoice.Currency,
		RawRequest:       prepared.RawRequest,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create authenticate payer gateway operation")
		return nil, err
	}

	updateGatewayOperationStatus := func(status store.GatewayOperationStatus, rawResponse []byte) error {
		return s.queries.UpdateGatewayOperationStatus(ctx, store.UpdateGatewayOperationStatusParams{
			ID:          gatewayOperation.ID,
			Status:      status,
			RawResponse: rawResponse,
		})
	}

	gatewayResp, err := prepared.Send(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to authenticate cardholder with gateway")
		if statusErr := updateGatewayOperationStatus(store.GatewayOperationStatusFailed, nil); statusErr != nil {
			log.Ctx(ctx).Error().Err(statusErr).Msg("failed to update gateway operation status after authenticate cardholder error")
		}
		return nil, err
	}

	if err := updateGatewayOperationStatus(store.GatewayOperationStatusSuccess, gatewayResp.RawResponse); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
		return nil, err
	}

	var (
		nextStep     AuthenticateCardholderNextStep
		targetStatus PaymentIntentStatus
		redirectHTML string
	)
	switch gatewayResp.NextStep {
	case AuthenticateCardholderGatewayNextStepChallenge:
		nextStep = AuthenticateCardholderNextStepChallenge
		targetStatus = PaymentIntentStatusAwaitingAuthenticationResult
		redirectHTML = gatewayResp.RedirectHTML
	case AuthenticateCardholderGatewayNextStepCapture:
		nextStep = AuthenticateCardholderNextStepCapture
		targetStatus = PaymentIntentStatusReadyToCapture
	case AuthenticateCardholderGatewayNextStepCantContinue:
		nextStep = AuthenticateCardholderNextStepCantContinue
		targetStatus = PaymentIntentStatusFailed
	default:
		return nil, fmt.Errorf("unsupported authenticate cardholder gateway next step %q", gatewayResp.NextStep)
	}

	if err := currentStatus.ValidatePaymentIntentTransition(paymentMethod, targetStatus); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("payment intent invalid transition after authenticating cardholder")
		return nil, err
	}
	targetStoreStatus, err := mapPaymentIntentStatusToStorePaymentIntentStatus(targetStatus)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map payment intent target status")
		return nil, err
	}
	if err := queriesWithTx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
		ID:     paymentIntent.ID,
		Status: targetStoreStatus,
	}); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update payment intent status")
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to commit authenticated cardholder result")
		return nil, err
	}

	return &AuthenticateCardholderResponse{
		NextStep:     nextStep,
		RedirectHTML: redirectHTML,
	}, nil
}

func (s *service) VerifyCardAuthentication(ctx context.Context, r VerifyCardAuthenticationRequest) (*VerifyCardAuthenticationResponse, error) {
	return &VerifyCardAuthenticationResponse{}, nil
}
