package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
	InitiateSession(ctx context.Context, req *InitiateSessionRequest) (*InitiateSessionResult, error)
	InitiateAuth(ctx context.Context, req *InitiateAuthRequest) (*InitiateAuthResult, error)
	ProcessAuth(ctx context.Context, req *ProcessAuthRequest) (*ProcessAuthResult, error)
	FinalizePayment(ctx context.Context, req *FinalizePaymentRequest) (*FinalizePaymentResult, error)
	VerifyDomain(ctx context.Context, domain string) (bool, error)
}

type service struct {
	pool          *pgxpool.Pool
	queries       store.TransactionQuerier
	encryptionKey []byte
}

func NewService(pool *pgxpool.Pool, queries store.TransactionQuerier, encryptionKey []byte) Service {
	return &service{
		pool:          pool,
		queries:       queries,
		encryptionKey: encryptionKey,
	}
}

func (s *service) InitiateSession(ctx context.Context, req *InitiateSessionRequest) (*InitiateSessionResult, error) {
	invoice, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        req.InvoiceID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &InvoiceNotFoundError{InvoiceID: req.InvoiceID.String()}
		}
		return nil, err
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		return nil, &InvoiceAlreadyPaidError{InvoiceID: req.InvoiceID.String()}
	}

	// Check for existing session with same idempotency key
	if req.IdempotencyKey != "" {
		existingSession, err := s.queries.GetPaymentIntentByIdempotencyKey(ctx, store.GetPaymentIntentByIdempotencyKeyParams{
			InvoiceID:      req.InvoiceID,
			IdempotencyKey: pgtype.Text{String: req.IdempotencyKey, Valid: true},
		})
		if err == nil {
			// Session already exists, return it
			return &InitiateSessionResult{
				PaymentIntentID:  existingSession.ID,
				GatewaySessionID: existingSession.GatewaySessionID.String,
			}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check idempotency: %w", err)
		}
	}

	account, err := s.queries.GetGatewayAccountByIDAndProject(ctx, store.GetGatewayAccountByIDAndProjectParams{
		ID:        req.GatewayAccountID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid gateway account: %w", err)
	}

	creds, err := mpgsclient.ParseEncryptedCredentials(account.Credentials, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgsclient.New(creds.BaseURL, creds.MerchantID, creds.APIPassword)

	resp, err := mpgsCli.CreateSession(ctx, &mpgsclient.CreateSessionRequest{
		Session: &mpgsclient.CreateSessionRequestSession{
			AuthenticationLimit: ptr.Ptr[int32](25),
		},
	})
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	var idempotencyKey pgtype.Text
	if req.IdempotencyKey != "" {
		idempotencyKey = pgtype.Text{String: req.IdempotencyKey, Valid: true}
	}

	dbSession, err := s.queries.CreatePaymentIntent(ctx, store.CreatePaymentIntentParams{
		InvoiceID:        invoice.ID,
		ProjectID:        invoice.ProjectID,
		GatewayAccountID: account.ID,
		GatewaySessionID: pgtype.Text{String: resp.Data.Session.ID, Valid: true},
		PaymentMethod:    req.PaymentMethod,
		PayerIp:          pgtype.Text{String: req.PayerIP, Valid: true},
		PayerUserAgent:   pgtype.Text{String: req.PayerUserAgent, Valid: req.PayerUserAgent != ""},
		IdempotencyKey:   idempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment session: %w", err)
	}

	_, err = mpgsCli.UpdateSession(ctx, resp.Data.Session.ID, &mpgsclient.UpdateSessionRequest{
		Order: mpgsclient.UpdateSessionOrder{
			Amount:   invoice.Amount.String(),
			Currency: invoice.Currency,
			ID:       invoice.ID.String(),
		},
	})
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	return &InitiateSessionResult{
		PaymentIntentID:  dbSession.ID,
		GatewaySessionID: resp.Data.Session.ID,
	}, nil
}

func (s *service) InitiateAuth(ctx context.Context, req *InitiateAuthRequest) (*InitiateAuthResult, error) {
	session, err := s.queries.GetPaymentIntentByIDAndProject(ctx, store.GetPaymentIntentByIDAndProjectParams{
		ID:        req.PaymentIntentID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment session not found")
		}
		return nil, err
	}

	if session.InvoiceID != req.InvoiceID {
		return nil, &SessionInvoiceMismatchError{
			SessionID: req.PaymentIntentID.String(),
			InvoiceID: req.InvoiceID.String(),
		}
	}

	if err := ValidateSessionTransition(session.Status, domain.PaymentIntentStatusAuthenticating); err != nil {
		return nil, err
	}

	if session.ExpiresAt.Valid && time.Now().After(session.ExpiresAt.Time) {
		return nil, &SessionExpiredError{SessionID: session.ID.String()}
	}

	invoice, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        req.InvoiceID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		return nil, err
	}

	account, err := s.queries.GetGatewayAccountByPaymentIntentID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	creds, err := mpgsclient.ParseEncryptedCredentials(account.Credentials, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgsclient.New(creds.BaseURL, creds.MerchantID, creds.APIPassword)
	gatewayTxID := uuid.New()

	mpgsReq := &mpgsclient.InitiateAuthenticationRequest{
		APIOperation: mpgsclient.OperationInitiateAuthentication,
		Authentication: mpgsclient.InitiateAuthenticationReqAuthentication{
			Channel: mpgsclient.ChannelPayerBrowser,
		},
		Order:   mpgsclient.InitiateAuthenticationOrder{Currency: invoice.Currency},
		Session: mpgsclient.InitiateAuthenticationSession{ID: session.GatewaySessionID.String},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	dbTx, err := s.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentIntentID:      session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      domain.TransactionTypeInitiateAuth,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
		RawRequest:           rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	resp, err := mpgsCli.InitiateAuthentication(ctx, invoice.ID.String(), gatewayTxID.String(), mpgsReq)
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	status := domain.TransactionStatusFailed
	if resp.Data.Result == mpgsclient.ResultSuccess {
		status = domain.TransactionStatusSuccess
	}

	if err := s.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
		ID:          dbTx.ID,
		Status:      status,
		RawResponse: resp.RawBody,
	}); err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	if status == domain.TransactionStatusSuccess {
		if err := s.queries.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
			ID:     session.ID,
			Status: domain.PaymentIntentStatusAuthenticating,
		}); err != nil {
			return nil, fmt.Errorf("failed to update session status: %w", err)
		}
	}

	var nextStep string
	switch resp.Data.Response.GatewayRecommendation {
	case mpgsclient.GatewayRecommendationProceed:
		nextStep = "authenticate"
	default:
		nextStep = "cant_continue"
	}

	return &InitiateAuthResult{
		TransactionID: gatewayTxID.String(),
		NextStep:      nextStep,
	}, nil
}

func (s *service) ProcessAuth(ctx context.Context, req *ProcessAuthRequest) (*ProcessAuthResult, error) {
	session, err := s.queries.GetPaymentIntentByIDAndProject(ctx, store.GetPaymentIntentByIDAndProjectParams{
		ID:        req.PaymentIntentID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment session not found")
		}
		return nil, err
	}

	if session.InvoiceID != req.InvoiceID {
		return nil, &SessionInvoiceMismatchError{
			SessionID: req.PaymentIntentID.String(),
			InvoiceID: req.InvoiceID.String(),
		}
	}

	if err := ValidateSessionTransition(session.Status, domain.PaymentIntentStatusAuthenticated); err != nil {
		return nil, err
	}

	if session.ExpiresAt.Valid && time.Now().After(session.ExpiresAt.Time) {
		return nil, &SessionExpiredError{SessionID: session.ID.String()}
	}

	invoice, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        req.InvoiceID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		return nil, err
	}

	lastTx, err := s.queries.GetLatestTransaction(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("no transaction found: %w", err)
	}

	if lastTx.Status != domain.TransactionStatusSuccess {
		return nil, fmt.Errorf("previous transaction not successful")
	}

	account, err := s.queries.GetGatewayAccountByPaymentIntentID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	project, err := s.queries.GetProjectByPaymentIntentID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	customDomain := project.CustomDomain.String
	if !project.CustomDomain.Valid || customDomain == "" {
		return nil, fmt.Errorf("custom domain not configured")
	}

	creds, err := mpgsclient.ParseEncryptedCredentials(account.Credentials, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgsclient.New(creds.BaseURL, creds.MerchantID, creds.APIPassword)

	mpgsReq := &mpgsclient.AuthenticatePayerRequest{
		APIOperation: mpgsclient.OperationAuthenticatePayer,
		Authentication: mpgsclient.AuthenticatePayerReqAuthentication{
			RedirectResponseURL: fmt.Sprintf("https://%s/checkout/%s/pay/card/%s/finalize", customDomain, req.InvoiceID, session.ID),
		},
		Device: mpgsclient.AuthenticatePayerReqDevice{
			Browser: req.UserAgent,
			BrowserDetails: &mpgsclient.AuthenticatePayerReqBrowserDetails{
				ThreeDSecureChallengeWindowSize: req.BrowserDetails.ThreeDSecureChallengeWindowSize,
				AcceptHeaders:                   req.BrowserDetails.AcceptHeaders,
				ColorDepth:                      req.BrowserDetails.ColorDepth,
				JavaEnabled:                     req.BrowserDetails.JavaEnabled,
				Language:                        req.BrowserDetails.Language,
				ScreenHeight:                    req.BrowserDetails.ScreenHeight,
				ScreenWidth:                     req.BrowserDetails.ScreenWidth,
				TimeZone:                        req.BrowserDetails.TimeZone,
			},
			IPAddress: req.PayerIP,
		},
		Order: mpgsclient.AuthenticatePayerReqOrder{
			Amount:   invoice.Amount.String(),
			Currency: invoice.Currency,
		},
		Session: mpgsclient.AuthenticatePayerReqSession{
			ID: session.GatewaySessionID.String,
		},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	dbTx, err := s.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentIntentID:      session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      domain.TransactionTypeAuthenticatePayer,
		GatewayTransactionID: lastTx.GatewayTransactionID,
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
		RawRequest:           rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	resp, err := mpgsCli.AuthenticatePayer(ctx, invoice.ID.String(), lastTx.GatewayTransactionID, mpgsReq)
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	status := domain.TransactionStatusFailed
	if resp.Data.Result == mpgsclient.ResultPending || resp.Data.Result == mpgsclient.ResultSuccess {
		status = domain.TransactionStatusSuccess
	}

	if err := s.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
		ID:          dbTx.ID,
		Status:      status,
		RawResponse: resp.RawBody,
	}); err != nil {
		return nil, fmt.Errorf("failed to update transaction: %w", err)
	}

	if status == domain.TransactionStatusSuccess {
		if err := s.queries.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
			ID:     session.ID,
			Status: domain.PaymentIntentStatusAuthenticated,
		}); err != nil {
			return nil, fmt.Errorf("failed to update session status: %w", err)
		}
	}

	var nextStep, html string
	if resp.Data.Authentication.Redirect.HTML != "" {
		nextStep = "3ds_challenge"
		html = resp.Data.Authentication.Redirect.HTML
	} else {
		nextStep = "pay"
	}

	return &ProcessAuthResult{
		NextStep:     nextStep,
		RedirectHTML: html,
	}, nil
}

func (s *service) FinalizePayment(ctx context.Context, req *FinalizePaymentRequest) (*FinalizePaymentResult, error) {
	session, err := s.queries.GetPaymentIntentByIDAndProject(ctx, store.GetPaymentIntentByIDAndProjectParams{
		ID:        req.PaymentIntentID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment session not found")
		}
		return nil, err
	}

	if session.InvoiceID != req.InvoiceID {
		return nil, &SessionInvoiceMismatchError{
			SessionID: req.PaymentIntentID.String(),
			InvoiceID: req.InvoiceID.String(),
		}
	}

	if err := ValidateSessionTransition(session.Status, domain.PaymentIntentStatusPaying); err != nil {
		return nil, err
	}

	if session.ExpiresAt.Valid && time.Now().After(session.ExpiresAt.Time) {
		return nil, &SessionExpiredError{SessionID: session.ID.String()}
	}

	invoice, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        req.InvoiceID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &InvoiceNotFoundError{InvoiceID: req.InvoiceID.String()}
		}
		return nil, err
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		return nil, &InvoiceAlreadyPaidError{InvoiceID: invoice.ID.String()}
	}

	existingTx, err := s.queries.GetPayTransactionByPaymentIntentID(ctx, session.ID)
	if err == nil && existingTx.Status == domain.TransactionStatusSuccess {
		return &FinalizePaymentResult{
			Success:     true,
			ResultCode:  ResultSuccess,
			CheckoutURL: fmt.Sprintf("/checkout/%s", req.InvoiceID),
		}, nil
	}

	authTx, err := s.queries.GetSuccessfulAuthTransaction(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("authentication missing: %w", err)
	}

	account, err := s.queries.GetGatewayAccountByPaymentIntentID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	creds, err := mpgsclient.ParseEncryptedCredentials(account.Credentials, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgsclient.New(creds.BaseURL, creds.MerchantID, creds.APIPassword)
	gatewayTxID := uuid.New()

	mpgsReq := &mpgsclient.ExecutePayRequest{
		APIOperation: mpgsclient.OperationPay,
		Authentication: mpgsclient.ExecutePayReqAuthentication{
			TransactionID: authTx.GatewayTransactionID,
		},
		Order: mpgsclient.ExecutePayReqOrder{
			Amount:    invoice.Amount.String(),
			Currency:  invoice.Currency,
			Reference: invoice.ID.String(),
		},
		Session: mpgsclient.ExecutePayReqSession{
			ID: session.GatewaySessionID.String,
		},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	dbTx, err := s.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentIntentID:      session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      domain.TransactionTypePay,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
		RawRequest:           rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	resp, err := mpgsCli.ExecutePay(ctx, invoice.ID.String(), gatewayTxID.String(), mpgsReq)
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	pgxTx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer pgxTx.Rollback(ctx)

	qtx := s.queries.WithTx(pgxTx)

	result := &FinalizePaymentResult{
		CheckoutURL: fmt.Sprintf("/checkout/%s", req.InvoiceID),
	}

	if resp.Data.Response.GatewayCode == mpgsclient.CodeApproved {
		if err := ValidateInvoiceTransition(invoice.Status, domain.InvoiceStatusPaid); err != nil {
			return nil, err
		}

		if err := qtx.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
			ID:          dbTx.ID,
			Status:      domain.TransactionStatusSuccess,
			RawResponse: resp.RawBody,
		}); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}

		if err := qtx.MarkInvoicePaid(ctx, invoice.ID); err != nil {
			return nil, fmt.Errorf("failed to update invoice: %w", err)
		}

		result.Success = true
		result.ResultCode = ResultSuccess

		if err := qtx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
			ID:     session.ID,
			Status: domain.PaymentIntentStatusCompleted,
		}); err != nil {
			return nil, fmt.Errorf("failed to update session status: %w", err)
		}
	} else {
		if err := qtx.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
			ID:          dbTx.ID,
			Status:      domain.TransactionStatusFailed,
			RawResponse: resp.RawBody,
		}); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}

		result.Success = false
		result.ResultCode = ResultDeclined

		if err := qtx.MarkInvoiceFailed(ctx, invoice.ID); err != nil {
			return nil, fmt.Errorf("failed to mark invoice as failed: %w", err)
		}

		if err := qtx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
			ID:     session.ID,
			Status: domain.PaymentIntentStatusFailed,
		}); err != nil {
			return nil, fmt.Errorf("failed to update session status: %w", err)
		}
	}

	if err := pgxTx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit db transaction: %w", err)
	}

	return result, nil
}

func (s *service) VerifyDomain(ctx context.Context, domain string) (bool, error) {
	_, err := s.queries.GetProjectByCustomDomain(ctx, pgtype.Text{String: domain, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
