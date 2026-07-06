package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/ptr"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

type Service interface {
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

	if err := ValidateSessionTransition(session.Status, PaymentIntentStatusReadyToCapture); err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, &SessionExpiredError{SessionID: session.ID.String()}
	}

	invoice, err := s.queries.GetInvoiceByIDAndProject(ctx, store.GetInvoiceByIDAndProjectParams{
		ID:        req.InvoiceID,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		return nil, err
	}

	lastOp, err := s.queries.GetLatestGatewayOperation(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("no gateway operation found: %w", err)
	}

	if lastOp.Status != TransactionStatusSuccess {
		return nil, fmt.Errorf("previous gateway operation not successful")
	}

	account, err := s.queries.GetGatewayAccountByPaymentIntentID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	project, err := s.queries.GetProjectByPaymentIntentID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	customDomain := project.CustomDomain
	if customDomain == nil {
		return nil, fmt.Errorf("custom domain not configured")
	}

	cfg, err := mpgsclient.ParseConfig(account.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway config: %w", err)
	}
	sec, err := mpgsclient.DecryptSecret(account.Secret, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgsclient.New(cfg.BaseURL, cfg.MerchantID, sec.APIPassword)

	mpgsReq := &mpgsclient.AuthenticatePayerRequest{
		APIOperation: mpgsclient.OperationAuthenticatePayer,
		Authentication: mpgsclient.AuthenticatePayerReqAuthentication{
			RedirectResponseURL: fmt.Sprintf("https://%s/checkout/%s/pay/card/%s/finalize", *customDomain, req.InvoiceID, session.ID),
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
			ID: ptr.Deref(session.GatewaySetupReference),
		},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	dbOp, err := s.queries.CreateGatewayOperation(ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  session.ID,
		InvoiceID:        invoice.ID,
		ProjectID:        invoice.ProjectID,
		GatewayAccountID: account.GatewayAccountID,
		OperationType:    TransactionTypeAuthenticatePayer,
		GatewayReference: lastOp.GatewayReference,
		Amount:           invoice.Amount,
		Currency:         invoice.Currency,
		RawRequest:       rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway operation: %w", err)
	}

	resp, err := mpgsCli.AuthenticatePayer(ctx, invoice.ID.String(), lastOp.GatewayReference, mpgsReq)
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	status := TransactionStatusFailed
	if resp.Data.Result == mpgsclient.ResultPending || resp.Data.Result == mpgsclient.ResultSuccess {
		status = TransactionStatusSuccess
	}

	if err := s.queries.UpdateGatewayOperationStatus(ctx, store.UpdateGatewayOperationStatusParams{
		ID:          dbOp.ID,
		Status:      status,
		RawResponse: resp.RawBody,
	}); err != nil {
		return nil, fmt.Errorf("failed to update gateway operation: %w", err)
	}

	if status == TransactionStatusSuccess {
		if err := s.queries.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
			ID:     session.ID,
			Status: PaymentIntentStatusReadyToCapture,
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

	if err := ValidateSessionTransition(session.Status, PaymentIntentStatusProcessingPayment); err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
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

	if invoice.Status == InvoiceStatusPaid {
		return nil, &InvoiceAlreadyPaidError{InvoiceID: invoice.ID.String()}
	}

	existingOp, err := s.queries.GetPayGatewayOperationByPaymentIntentID(ctx, session.ID)
	if err == nil && existingOp.Status == TransactionStatusSuccess {
		return &FinalizePaymentResult{
			Success:     true,
			ResultCode:  ResultSuccess,
			CheckoutURL: fmt.Sprintf("/checkout/%s", req.InvoiceID),
		}, nil
	}

	authOp, err := s.queries.GetSuccessfulAuthGatewayOperation(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("authentication missing: %w", err)
	}

	account, err := s.queries.GetGatewayAccountByPaymentIntentID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	cfg, err := mpgsclient.ParseConfig(account.Config)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway config: %w", err)
	}
	sec, err := mpgsclient.DecryptSecret(account.Secret, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgsclient.New(cfg.BaseURL, cfg.MerchantID, sec.APIPassword)
	gatewayTxID := uuid.New()

	mpgsReq := &mpgsclient.ExecutePayRequest{
		APIOperation: mpgsclient.OperationPay,
		Authentication: mpgsclient.ExecutePayReqAuthentication{
			TransactionID: authOp.GatewayReference,
		},
		Order: mpgsclient.ExecutePayReqOrder{
			Amount:    invoice.Amount.String(),
			Currency:  invoice.Currency,
			Reference: invoice.ID.String(),
		},
		Session: mpgsclient.ExecutePayReqSession{
			ID: ptr.Deref(session.GatewaySetupReference),
		},
	}

	rawReq, err := json.Marshal(mpgsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	dbOp, err := s.queries.CreateGatewayOperation(ctx, store.CreateGatewayOperationParams{
		PaymentIntentID:  session.ID,
		InvoiceID:        invoice.ID,
		ProjectID:        invoice.ProjectID,
		GatewayAccountID: account.GatewayAccountID,
		OperationType:    TransactionTypePay,
		GatewayReference: gatewayTxID.String(),
		Amount:           invoice.Amount,
		Currency:         invoice.Currency,
		RawRequest:       rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway operation: %w", err)
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
		if err := ValidateInvoiceTransition(invoice.Status, InvoiceStatusPaid); err != nil {
			return nil, err
		}

		if err := qtx.UpdateGatewayOperationStatus(ctx, store.UpdateGatewayOperationStatusParams{
			ID:          dbOp.ID,
			Status:      TransactionStatusSuccess,
			RawResponse: resp.RawBody,
		}); err != nil {
			return nil, fmt.Errorf("failed to update gateway operation: %w", err)
		}

		if err := qtx.MarkInvoicePaid(ctx, invoice.ID); err != nil {
			return nil, fmt.Errorf("failed to update invoice: %w", err)
		}

		result.Success = true
		result.ResultCode = ResultSuccess

		if err := qtx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
			ID:     session.ID,
			Status: PaymentIntentStatusCompleted,
		}); err != nil {
			return nil, fmt.Errorf("failed to update session status: %w", err)
		}
	} else {
		if err := qtx.UpdateGatewayOperationStatus(ctx, store.UpdateGatewayOperationStatusParams{
			ID:          dbOp.ID,
			Status:      TransactionStatusFailed,
			RawResponse: resp.RawBody,
		}); err != nil {
			return nil, fmt.Errorf("failed to update gateway operation: %w", err)
		}

		result.Success = false
		result.ResultCode = ResultDeclined

		if err := qtx.MarkInvoiceFailed(ctx, invoice.ID); err != nil {
			return nil, fmt.Errorf("failed to mark invoice as failed: %w", err)
		}

		if err := qtx.UpdatePaymentIntentStatus(ctx, store.UpdatePaymentIntentStatusParams{
			ID:     session.ID,
			Status: PaymentIntentStatusFailed,
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
	_, err := s.queries.GetProjectByCustomDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
