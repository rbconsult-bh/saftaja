package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
		AmountMinor:      invoice.AmountMinor,
		Currency:         invoice.Currency,
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
			Amount:    invoice.AmountMinor,
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
	if err := r.Validate(); err != nil {
		log.Ctx(ctx).Info().Err(err).Msg("validation failed")
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to begin capture tx")
		return nil, fmt.Errorf("failed to begin capture tx: %w", err)
	}
	defer tx.Rollback(ctx)

	queriesWithTx := s.queries.WithTx(tx)
	invoice, err := queriesWithTx.GetInvoiceByIDAndProjectForNoKeyUpdate(ctx, store.GetInvoiceByIDAndProjectForNoKeyUpdateParams{
		ID:        r.InvoiceID,
		ProjectID: r.ProjectID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to lock invoice: %w", err)
	}

	paymentIntent, err := queriesWithTx.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdate(
		ctx,
		store.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdateParams{
			ID:        r.PaymentIntentID,
			ProjectID: r.ProjectID,
			InvoiceID: r.InvoiceID,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to lock payment intent: %w", err)
	}

	currentStatus, err := mapStorePaymentIntentStatusToPaymentIntentStatus(paymentIntent.Status)
	if err != nil {
		return nil, err
	}
	switch currentStatus {
	case PaymentIntentStatusSucceeded:
		return &CapturePaymentIntentResponse{NextStep: CapturePaymentIntentNextStepComplete}, nil
	case PaymentIntentStatusFailed:
		return &CapturePaymentIntentResponse{NextStep: CapturePaymentIntentNextStepCantContinue}, nil
	case PaymentIntentStatusCapturing:
		// TODO: Reconcile capturing payment intents with a durable River job before production. Until that worker retrieves the existing gateway transaction and applies its final result, this intent remains stuck in capturing.
		return processingCapturePaymentIntentResponse(), nil
	case PaymentIntentStatusReadyToCapture:
		// Start capture below.
	default:
		return nil, fmt.Errorf(
			"%w: CapturePaymentIntent requires %s, got %s",
			ErrPaymentIntentInvalidState,
			PaymentIntentStatusReadyToCapture,
			currentStatus,
		)
	}

	switch invoice.Status {
	case store.InvoiceStatusPending:
		// Invoice can be paid.
	case store.InvoiceStatusPaid:
		return nil, ErrInvoiceAlreadyPaid
	case store.InvoiceStatusCancelled:
		return nil, ErrInvoiceCancelled
	default:
		return nil, fmt.Errorf("%w: unknown invoice status %q", ErrInvoiceInvalidState, invoice.Status)
	}

	if paymentIntent.ExpiresAt.Before(time.Now()) {
		return nil, ErrPaymentIntentExpired
	}
	if paymentIntent.GatewaySetupReference == nil || *paymentIntent.GatewaySetupReference == "" {
		return nil, fmt.Errorf("%w: payment intent is missing payment method reference", ErrPaymentIntentInvalidState)
	}

	paymentMethod, err := mapStorePaymentMethodToPaymentMethod(paymentIntent.PaymentMethod)
	if err != nil {
		return nil, err
	}
	if paymentMethod != PaymentMethodCard {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedPaymentMethod, paymentMethod)
	}

	hasOtherCapturingIntent, err := queriesWithTx.InvoiceHasOtherCapturingPaymentIntent(
		ctx,
		store.InvoiceHasOtherCapturingPaymentIntentParams{
			InvoiceID: r.InvoiceID,
			ID:        r.PaymentIntentID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check invoice capture: %w", err)
	}
	if hasOtherCapturingIntent {
		return nil, ErrInvoicePaymentInProgress
	}

	authenticationOperation, err := queriesWithTx.GetCompletedAuthenticateCardholderGatewayOperation(ctx, paymentIntent.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf(
				"%w: completed authenticate cardholder gateway operation is missing",
				ErrPaymentIntentInvalidState,
			)
		}
		return nil, fmt.Errorf("failed to get authenticate cardholder gateway operation: %w", err)
	}
	if authenticationOperation.GatewayReference == "" {
		return nil, fmt.Errorf(
			"%w: authenticate cardholder gateway operation is missing gateway reference",
			ErrPaymentIntentInvalidState,
		)
	}

	gatewayAccount, err := queriesWithTx.GetGatewayAccountByIDAndProject(ctx, store.GetGatewayAccountByIDAndProjectParams{
		ID:        paymentIntent.GatewayAccountID,
		ProjectID: paymentIntent.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway account: %w", err)
	}
	cardGateway, err := s.gatewayResolver.CardGateway(gatewayAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve card gateway: %w", err)
	}

	prepared, err := cardGateway.CaptureCardPayment(ctx, CaptureCardPaymentGatewayRequest{
		InvoiceID:               invoice.ID,
		PaymentReference:        paymentIntent.ID,
		Amount:                  paymentIntent.AmountMinor,
		Currency:                paymentIntent.Currency,
		PaymentMethodReference:  PaymentMethodReference(*paymentIntent.GatewaySetupReference),
		AuthenticationReference: AuthenticationReference(authenticationOperation.GatewayReference),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to prepare capture card payment: %w", err)
	}

	gatewayOperation, err := queriesWithTx.CreateGatewayOperation(ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  paymentIntent.ID,
		InvoiceID:        invoice.ID,
		ProjectID:        paymentIntent.ProjectID,
		GatewayAccountID: paymentIntent.GatewayAccountID,
		OperationType:    store.GatewayOperationTypeCapturePayment,
		GatewayReference: paymentIntent.ID.String(),
		RawRequest:       prepared.RawRequest,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create capture gateway operation: %w", err)
	}

	if err := currentStatus.ValidatePaymentIntentTransition(paymentMethod, PaymentIntentStatusCapturing); err != nil {
		return nil, err
	}
	if err := queriesWithTx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
		ID:     paymentIntent.ID,
		Status: store.PaymentIntentStatusCapturing,
	}); err != nil {
		return nil, fmt.Errorf("failed to mark payment intent capturing: %w", err)
	}

	// TODO: Enqueue a durable payment-capture reconciliation job whenever a payment intent enters or remains in capturing. The worker must retrieve the existing gateway transaction using the payment-intent ID and must never create a new gateway transaction for reconciliation.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit capture start: %w", err)
	}

	gatewayResponse, gatewayErr := prepared.Send(ctx)
	if gatewayErr != nil {
		log.Ctx(ctx).Error().Err(gatewayErr).Msg("card capture outcome is unknown")
		if err := s.finishCardCapture(
			ctx,
			r.PaymentIntentRef,
			gatewayOperation.ID,
			store.GatewayOperationStatusErrored,
			CaptureCardPaymentGatewayResultUnknown,
			rawGatewayResponse(gatewayErr),
		); err != nil {
			return nil, err
		}
		return processingCapturePaymentIntentResponse(), nil
	}

	if err := s.finishCardCapture(
		ctx,
		r.PaymentIntentRef,
		gatewayOperation.ID,
		store.GatewayOperationStatusCompleted,
		gatewayResponse.Result,
		gatewayResponse.RawResponse,
	); err != nil {
		return nil, err
	}

	switch gatewayResponse.Result {
	case CaptureCardPaymentGatewayResultSucceeded:
		return &CapturePaymentIntentResponse{NextStep: CapturePaymentIntentNextStepComplete}, nil
	case CaptureCardPaymentGatewayResultDeclined:
		return &CapturePaymentIntentResponse{NextStep: CapturePaymentIntentNextStepCantContinue}, nil
	case CaptureCardPaymentGatewayResultPending:
		return processingCapturePaymentIntentResponse(), nil
	case CaptureCardPaymentGatewayResultUnknown:
		return processingCapturePaymentIntentResponse(), nil
	default:
		return nil, fmt.Errorf("unsupported capture card payment gateway result %q", gatewayResponse.Result)
	}
}

func (s *service) finishCardCapture(
	ctx context.Context,
	ref PaymentIntentRef,
	gatewayOperationID uuid.UUID,
	operationStatus store.GatewayOperationStatus,
	result CaptureCardPaymentGatewayResult,
	rawResponse []byte,
) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin capture result tx: %w", err)
	}
	defer tx.Rollback(ctx)

	queriesWithTx := s.queries.WithTx(tx)
	if _, err := queriesWithTx.GetInvoiceByIDAndProjectForNoKeyUpdate(ctx, store.GetInvoiceByIDAndProjectForNoKeyUpdateParams{
		ID:        ref.InvoiceID,
		ProjectID: ref.ProjectID,
	}); err != nil {
		return fmt.Errorf("failed to lock invoice for capture result: %w", err)
	}
	paymentIntent, err := queriesWithTx.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdate(
		ctx,
		store.GetPaymentIntentByIDAndProjectAndInvoiceForNoKeyUpdateParams{
			ID:        ref.PaymentIntentID,
			ProjectID: ref.ProjectID,
			InvoiceID: ref.InvoiceID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to lock payment intent for capture result: %w", err)
	}

	if err := queriesWithTx.UpdateGatewayOperationStatus(ctx, store.UpdateGatewayOperationStatusParams{
		ID:          gatewayOperationID,
		Status:      operationStatus,
		RawResponse: rawResponse,
	}); err != nil {
		return fmt.Errorf("failed to update capture gateway operation: %w", err)
	}

	currentStatus, err := mapStorePaymentIntentStatusToPaymentIntentStatus(paymentIntent.Status)
	if err != nil {
		return err
	}
	paymentMethod, err := mapStorePaymentMethodToPaymentMethod(paymentIntent.PaymentMethod)
	if err != nil {
		return err
	}

	var targetStatus PaymentIntentStatus
	switch result {
	case CaptureCardPaymentGatewayResultSucceeded:
		targetStatus = PaymentIntentStatusSucceeded
	case CaptureCardPaymentGatewayResultDeclined:
		targetStatus = PaymentIntentStatusFailed
	case CaptureCardPaymentGatewayResultPending, CaptureCardPaymentGatewayResultUnknown:
		return tx.Commit(ctx)
	default:
		return fmt.Errorf("unsupported capture card payment gateway result %q", result)
	}

	if err := currentStatus.ValidatePaymentIntentTransition(paymentMethod, targetStatus); err != nil {
		return err
	}
	targetStoreStatus, err := mapPaymentIntentStatusToStorePaymentIntentStatus(targetStatus)
	if err != nil {
		return err
	}
	if err := queriesWithTx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
		ID:     paymentIntent.ID,
		Status: targetStoreStatus,
	}); err != nil {
		return fmt.Errorf("failed to update captured payment intent: %w", err)
	}
	if result == CaptureCardPaymentGatewayResultSucceeded {
		if err := queriesWithTx.MarkInvoicePaid(ctx, ref.InvoiceID); err != nil {
			return fmt.Errorf("failed to mark invoice paid: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit capture result: %w", err)
	}
	return nil
}

func processingCapturePaymentIntentResponse() *CapturePaymentIntentResponse {
	return &CapturePaymentIntentResponse{
		NextStep: CapturePaymentIntentNextStepProcessing,
	}
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

	if currentStatus != PaymentIntentStatusCreated {
		err := fmt.Errorf(
			"%w: PrepareCardAuthentication requires %s, got %s",
			ErrPaymentIntentInvalidState,
			PaymentIntentStatusCreated,
			currentStatus,
		)
		log.Ctx(ctx).Info().Err(err).Msg("payment intent invalid status")
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
		Currency:               paymentIntent.Currency,
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
		OperationType:    store.GatewayOperationTypePrepareCardAuthentication,
		GatewayReference: string(prepared.AuthenticationReference),
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
		if statusErr := updateGatewayOperationStatus(store.GatewayOperationStatusErrored, nil); statusErr != nil {
			log.Ctx(ctx).Error().Err(statusErr).Msg("failed to update gateway operation status after prepare card authentication error")
		}
		return nil, err
	}

	if gatewayResp.Result != PrepareCardAuthenticationGatewayResultAvailable {
		// The gateway call completed with a valid response, but its decision was "can't continue."
		// Keep the payment intent in created so the customer can update the card and
		// retry PrepareCardAuthentication. The raw gateway decision is stored on this operation.
		if err := updateGatewayOperationStatus(store.GatewayOperationStatusCompleted, gatewayResp.RawResponse); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
			return nil, err
		}
		return &PrepareCardAuthenticationResponse{
			NextStep: PrepareCardAuthenticationNextStepCantContinue,
		}, nil
	}

	if err := updateGatewayOperationStatus(store.GatewayOperationStatusCompleted, gatewayResp.RawResponse); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
		return nil, err
	}

	if err := currentStatus.ValidatePaymentIntentTransition(paymentMethod, PaymentIntentStatusReadyToAuthenticate); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("payment intent invalid transition after preparing card authentication")
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

	prepareAuthenticationOperation, err := s.queries.GetCompletedPrepareCardAuthenticationGatewayOperation(ctx, paymentIntent.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Ctx(ctx).Error().Msg("completed prepare card authentication gateway operation not found")
			return nil, fmt.Errorf(
				"%w: completed prepare card authentication gateway operation is missing",
				ErrPaymentIntentInvalidState,
			)
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get completed prepare card authentication gateway operation")
		return nil, err
	}
	if prepareAuthenticationOperation.GatewayReference == "" {
		log.Ctx(ctx).Error().Msg("prepare card authentication gateway operation is missing gateway reference")
		return nil, fmt.Errorf(
			"%w: prepare card authentication gateway operation is missing gateway reference",
			ErrPaymentIntentInvalidState,
		)
	}

	prepared, err := cardGateway.AuthenticateCardholder(ctx, AuthenticateCardholderGatewayRequest{
		InvoiceID:               invoice.ID,
		Amount:                  paymentIntent.AmountMinor,
		Currency:                paymentIntent.Currency,
		PaymentMethodReference:  PaymentMethodReference(*paymentIntent.GatewaySetupReference),
		AuthenticationReference: AuthenticationReference(prepareAuthenticationOperation.GatewayReference),
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
		OperationType:    store.GatewayOperationTypeAuthenticateCardholder,
		GatewayReference: prepareAuthenticationOperation.GatewayReference,
		RawRequest:       prepared.RawRequest,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create authenticate cardholder gateway operation")
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
		if statusErr := updateGatewayOperationStatus(store.GatewayOperationStatusErrored, nil); statusErr != nil {
			log.Ctx(ctx).Error().Err(statusErr).Msg("failed to update gateway operation status after authenticate cardholder error")
		}
		return nil, err
	}

	if err := updateGatewayOperationStatus(store.GatewayOperationStatusCompleted, gatewayResp.RawResponse); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
		return nil, err
	}

	var (
		nextStep     AuthenticateCardholderNextStep
		targetStatus PaymentIntentStatus
		redirectHTML string
	)
	switch gatewayResp.Result {
	case AuthenticateCardholderGatewayResultChallengeRequired:
		nextStep = AuthenticateCardholderNextStepChallenge
		targetStatus = PaymentIntentStatusAwaitingAuthenticationResult
		redirectHTML = gatewayResp.RedirectHTML
	case AuthenticateCardholderGatewayResultSucceeded:
		nextStep = AuthenticateCardholderNextStepCapture
		targetStatus = PaymentIntentStatusReadyToCapture
	case AuthenticateCardholderGatewayResultFailed:
		nextStep = AuthenticateCardholderNextStepCantContinue
		targetStatus = PaymentIntentStatusFailed
	default:
		return nil, fmt.Errorf("unsupported authenticate cardholder gateway result %q", gatewayResp.Result)
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
		if errors.Is(err, pgx.ErrNoRows) {
			log.Ctx(ctx).Info().Msg("payment intent not found")
			return nil, ErrNotFound
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to lock payment intent")
		return nil, err
	}

	currentStatus, err := mapStorePaymentIntentStatusToPaymentIntentStatus(paymentIntent.Status)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment intent status")
		return nil, err
	}
	paymentMethod, err := mapStorePaymentMethodToPaymentMethod(paymentIntent.PaymentMethod)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to map store payment method")
		return nil, err
	}

	switch currentStatus {
	case PaymentIntentStatusReadyToCapture:
		return &VerifyCardAuthenticationResponse{
			NextStep: VerifyCardAuthenticationNextStepCapture,
		}, nil
	case PaymentIntentStatusFailed:
		return &VerifyCardAuthenticationResponse{
			NextStep: VerifyCardAuthenticationNextStepCantContinue,
		}, nil
	case PaymentIntentStatusAwaitingAuthenticationResult:
		// Retrieve the current authentication result below.
	default:
		err := fmt.Errorf(
			"%w: VerifyCardAuthentication requires %s, got %s",
			ErrPaymentIntentInvalidState,
			PaymentIntentStatusAwaitingAuthenticationResult,
			currentStatus,
		)
		log.Ctx(ctx).Info().Err(err).Msg("payment intent invalid status")
		return nil, err
	}

	if paymentIntent.ExpiresAt.Before(time.Now()) {
		log.Ctx(ctx).Info().Msg("payment intent expired")
		return nil, ErrPaymentIntentExpired
	}

	invoice, err := queriesWithTx.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        paymentIntent.InvoiceID,
		ProjectID: paymentIntent.ProjectID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get invoice")
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

	authenticationOperation, err := queriesWithTx.GetCompletedAuthenticateCardholderGatewayOperation(ctx, paymentIntent.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Ctx(ctx).Error().Msg("completed authenticate cardholder gateway operation not found")
			return nil, fmt.Errorf(
				"%w: completed authenticate cardholder gateway operation is missing",
				ErrPaymentIntentInvalidState,
			)
		}

		log.Ctx(ctx).Error().Err(err).Msg("failed to get completed authenticate cardholder gateway operation")
		return nil, err
	}
	if authenticationOperation.GatewayReference == "" {
		log.Ctx(ctx).Error().Msg("authenticate cardholder gateway operation is missing gateway reference")
		return nil, fmt.Errorf(
			"%w: authenticate cardholder gateway operation is missing gateway reference",
			ErrPaymentIntentInvalidState,
		)
	}

	cardGateway, err := s.gatewayResolver.CardGateway(gatewayAccount)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to resolve card gateway")
		return nil, err
	}

	prepared, err := cardGateway.GetCardAuthenticationResult(ctx, GetCardAuthenticationResultGatewayRequest{
		InvoiceID:               invoice.ID,
		Amount:                  paymentIntent.AmountMinor,
		Currency:                paymentIntent.Currency,
		AuthenticationReference: AuthenticationReference(authenticationOperation.GatewayReference),
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to prepare get card authentication result request")
		return nil, err
	}

	gatewayOperation, err := s.queries.CreateGatewayOperation(ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  paymentIntent.ID,
		InvoiceID:        invoice.ID,
		ProjectID:        paymentIntent.ProjectID,
		GatewayAccountID: paymentIntent.GatewayAccountID,
		OperationType:    store.GatewayOperationTypeGetCardAuthenticationResult,
		GatewayReference: authenticationOperation.GatewayReference,
		RawRequest:       prepared.RawRequest,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create get card authentication result gateway operation")
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
		log.Ctx(ctx).Error().Err(err).Msg("failed to get card authentication result")
		if statusErr := updateGatewayOperationStatus(store.GatewayOperationStatusErrored, nil); statusErr != nil {
			log.Ctx(ctx).Error().Err(statusErr).Msg("failed to update gateway operation after get card authentication result error")
		}
		return nil, err
	}

	var (
		nextStep     VerifyCardAuthenticationNextStep
		targetStatus PaymentIntentStatus
	)
	switch gatewayResp.Result {
	case CardAuthenticationResultSucceeded:
		nextStep = VerifyCardAuthenticationNextStepCapture
		targetStatus = PaymentIntentStatusReadyToCapture
	case CardAuthenticationResultPending:
		if err := updateGatewayOperationStatus(store.GatewayOperationStatusCompleted, gatewayResp.RawResponse); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to commit pending card authentication result")
			return nil, err
		}
		return &VerifyCardAuthenticationResponse{
			NextStep: VerifyCardAuthenticationNextStepPending,
		}, nil
	case CardAuthenticationResultFailed:
		nextStep = VerifyCardAuthenticationNextStepCantContinue
		targetStatus = PaymentIntentStatusFailed
	default:
		if statusErr := updateGatewayOperationStatus(store.GatewayOperationStatusErrored, gatewayResp.RawResponse); statusErr != nil {
			log.Ctx(ctx).Error().Err(statusErr).Msg("failed to update gateway operation after unsupported authentication result")
		}
		return nil, fmt.Errorf("unsupported card authentication result %q", gatewayResp.Result)
	}

	if err := updateGatewayOperationStatus(store.GatewayOperationStatusCompleted, gatewayResp.RawResponse); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to update gateway operation status")
		return nil, err
	}
	if err := currentStatus.ValidatePaymentIntentTransition(paymentMethod, targetStatus); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("payment intent invalid transition after verifying card authentication")
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
		log.Ctx(ctx).Error().Err(err).Msg("failed to commit verified card authentication result")
		return nil, err
	}

	return &VerifyCardAuthenticationResponse{
		NextStep: nextStep,
	}, nil
}
