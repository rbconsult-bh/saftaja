package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	mpgsclient "github.com/rbconsult-bh/saftaja/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/internal/connectors/mpgs"
	"github.com/rbconsult-bh/saftaja/internal/domain"
	"github.com/rbconsult-bh/saftaja/internal/store"
	"github.com/rbconsult-bh/saftaja/internal/utils"
)

type Service interface {
	GetCheckoutData(ctx context.Context, invoiceID uuid.UUID) (*CheckoutData, error)
	InitiateSession(ctx context.Context, req *InitiateSessionRequest) (*InitiateSessionResult, error)
	InitiateAuth(ctx context.Context, req *InitiateAuthRequest) (*InitiateAuthResult, error)
	ProcessAuth(ctx context.Context, req *ProcessAuthRequest) (*ProcessAuthResult, error)
	FinalizePayment(ctx context.Context, req *FinalizePaymentRequest) (*FinalizePaymentResult, error)
}

type service struct {
	pool    *pgxpool.Pool
	queries *store.Queries
}

func NewService(pool *pgxpool.Pool, queries *store.Queries) Service {
	return &service{
		pool:    pool,
		queries: queries,
	}
}

func (s *service) GetCheckoutData(ctx context.Context, invoiceID uuid.UUID) (*CheckoutData, error) {
	invoice, err := s.queries.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		return &CheckoutData{
			Invoice: InvoiceInfo{
				ID:       invoice.ID,
				Amount:   invoice.Amount.String(),
				Currency: invoice.Currency,
			},
			IsPaid: true,
		}, nil
	}

	items, _ := s.queries.GetInvoiceItems(ctx, invoiceID)
	itemInfos := make([]ItemInfo, len(items))
	for i, item := range items {
		itemInfos[i] = ItemInfo{
			Name:      item.Name,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice.String(),
			Amount:    item.Amount.String(),
		}
	}

	accounts, err := s.queries.ListActiveGatewayAccounts(ctx, invoice.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list gateway accounts: %w", err)
	}

	var options []PaymentOption
	var mpgsConfig *MPGSConfig

	for _, acc := range accounts {
		if acc.ConnectorType == domain.ConnectorTypeMPGS {
			creds, err := mpgs.ParseCredentials(acc.Credentials)
			if err != nil {
				return nil, fmt.Errorf("invalid gateway credentials: %w", err)
			}

			mpgsConfig = &MPGSConfig{
				BaseURL:    creds.BaseURL,
				MerchantID: creds.MerchantID,
				APIVersion: mpgsclient.APIVersion,
			}

			options = append(options, PaymentOption{
				GatewayAccountID: acc.ID,
				Method:           domain.PaymentMethodCard,
				Label:            "Credit / Debit Card",
			})
			options = append(options, PaymentOption{
				GatewayAccountID: acc.ID,
				Method:           domain.PaymentMethodApplePay,
				Label:            "Apple Pay",
			})
		}
	}

	return &CheckoutData{
		Invoice: InvoiceInfo{
			ID:            invoice.ID,
			Amount:        invoice.Amount.String(),
			Currency:      invoice.Currency,
			Description:   invoice.Description.String,
			CustomerEmail: invoice.CustomerEmail.String,
			CustomerName:  invoice.CustomerName.String,
		},
		Items:          itemInfos,
		PaymentOptions: options,
		MPGSConfig:     mpgsConfig,
		IsPaid:         false,
	}, nil
}

func (s *service) InitiateSession(ctx context.Context, req *InitiateSessionRequest) (*InitiateSessionResult, error) {
	invoice, err := s.queries.GetInvoiceByID(ctx, req.InvoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		return nil, &InvoiceAlreadyPaidError{InvoiceID: req.InvoiceID.String()}
	}

	account, err := s.queries.GetGatewayAccount(ctx, req.GatewayAccountID)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway account: %w", err)
	}

	creds, err := mpgs.ParseCredentials(account.Credentials)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgs.NewClient(creds)

	resp, err := mpgsCli.CreateSession(ctx, &mpgsclient.CreateSessionRequest{
		Session: &mpgsclient.CreateSessionRequestSession{
			AuthenticationLimit: utils.Ptr[int32](25),
		},
	})
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	dbSession, err := s.queries.CreatePaymentSession(ctx, store.CreatePaymentSessionParams{
		InvoiceID:        invoice.ID,
		ProjectID:        invoice.ProjectID,
		GatewayAccountID: account.ID,
		GatewaySessionID: resp.Data.Session.ID,
		PaymentMethod:    req.PaymentMethod,
		PayerIp:          pgtype.Text{String: req.PayerIP, Valid: true},
		PayerUserAgent:   pgtype.Text{String: req.PayerUserAgent, Valid: req.PayerUserAgent != ""},
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
		PaymentSessionID: dbSession.ID,
		GatewaySessionID: resp.Data.Session.ID,
	}, nil
}

func (s *service) InitiateAuth(ctx context.Context, req *InitiateAuthRequest) (*InitiateAuthResult, error) {
	session, err := s.queries.GetPaymentSessionByID(ctx, req.PaymentSessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment session not found")
		}
		return nil, err
	}

	if session.InvoiceID != req.InvoiceID {
		return nil, &SessionInvoiceMismatchError{
			SessionID: req.PaymentSessionID.String(),
			InvoiceID: req.InvoiceID.String(),
		}
	}

	invoice, err := s.queries.GetInvoiceByID(ctx, req.InvoiceID)
	if err != nil {
		return nil, err
	}

	account, err := s.queries.GetGatewayAccountByPaymentSessionID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	creds, err := mpgs.ParseCredentials(account.Credentials)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgs.NewClient(creds)
	gatewayTxID := uuid.New()

	dbTx, err := s.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentSessionID:     session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      domain.TransactionTypeInitiateAuth,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	resp, err := mpgsCli.InitiateAuthentication(ctx, invoice.ID.String(), gatewayTxID.String(), &mpgsclient.InitiateAuthenticationRequest{
		APIOperation: mpgsclient.OperationInitiateAuthentication,
		Authentication: mpgsclient.InitiateAuthenticationReqAuthentication{
			Channel: mpgsclient.ChannelPayerBrowser,
		},
		Order:   mpgsclient.InitiateAuthenticationOrder{Currency: invoice.Currency},
		Session: mpgsclient.InitiateAuthenticationSession{ID: session.GatewaySessionID},
	})
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
	session, err := s.queries.GetPaymentSessionByID(ctx, req.PaymentSessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment session not found")
		}
		return nil, err
	}

	if session.InvoiceID != req.InvoiceID {
		return nil, &SessionInvoiceMismatchError{
			SessionID: req.PaymentSessionID.String(),
			InvoiceID: req.InvoiceID.String(),
		}
	}

	invoice, err := s.queries.GetInvoiceByID(ctx, req.InvoiceID)
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

	account, err := s.queries.GetGatewayAccountByPaymentSessionID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	project, err := s.queries.GetProjectByPaymentSessionID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	customDomain := project.CustomDomain.String
	if !project.CustomDomain.Valid || customDomain == "" {
		return nil, fmt.Errorf("custom domain not configured")
	}

	creds, err := mpgs.ParseCredentials(account.Credentials)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgs.NewClient(creds)
	gatewayTxID := uuid.New()

	dbTx, err := s.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentSessionID:     session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      domain.TransactionTypeAuthenticatePayer,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	browserDetails := mpgsclient.AuthenticatePayerReqBrowserDetails{
		AcceptHeaders: req.BrowserDetails.AcceptHeaders,
		ColorDepth:    req.BrowserDetails.ColorDepth,
		JavaEnabled:   req.BrowserDetails.JavaEnabled,
		Language:      req.BrowserDetails.Language,
		ScreenHeight:  req.BrowserDetails.ScreenHeight,
		ScreenWidth:   req.BrowserDetails.ScreenWidth,
		TimeZone:      req.BrowserDetails.TimeZone,
	}

	resp, err := mpgsCli.AuthenticatePayer(ctx, invoice.ID.String(), lastTx.GatewayTransactionID, &mpgsclient.AuthenticatePayerRequest{
		APIOperation: mpgsclient.OperationAuthenticatePayer,
		Authentication: mpgsclient.AuthenticatePayerReqAuthentication{
			RedirectResponseURL: fmt.Sprintf("https://%s/checkout/%s/pay/card/%s/finalize", customDomain, req.InvoiceID, session.ID),
		},
		Device: mpgsclient.AuthenticatePayerReqDevice{
			Browser:        req.UserAgent,
			BrowserDetails: &browserDetails,
			IPAddress:      req.PayerIP,
		},
		Order: mpgsclient.AuthenticatePayerReqOrder{
			Amount:   invoice.Amount.String(),
			Currency: invoice.Currency,
		},
		Session: mpgsclient.AuthenticatePayerReqSession{
			ID: session.GatewaySessionID,
		},
	})
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
	session, err := s.queries.GetPaymentSessionByID(ctx, req.PaymentSessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment session not found")
		}
		return nil, err
	}

	if session.InvoiceID != req.InvoiceID {
		return nil, &SessionInvoiceMismatchError{
			SessionID: req.PaymentSessionID.String(),
			InvoiceID: req.InvoiceID.String(),
		}
	}

	if session.ExpiresAt.Valid && time.Now().After(session.ExpiresAt.Time) {
		return nil, &SessionExpiredError{SessionID: session.ID.String()}
	}

	invoice, err := s.queries.GetInvoiceByID(ctx, req.InvoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invoice not found")
		}
		return nil, err
	}

	if invoice.Status == domain.InvoiceStatusPaid {
		return nil, &InvoiceAlreadyPaidError{InvoiceID: invoice.ID.String()}
	}

	authTx, err := s.queries.GetSuccessfulAuthTransaction(ctx, session.ID)
	if err != nil {
		return nil, fmt.Errorf("authentication missing: %w", err)
	}

	account, err := s.queries.GetGatewayAccountByPaymentSessionID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	creds, err := mpgs.ParseCredentials(account.Credentials)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentials: %w", err)
	}

	mpgsCli := mpgs.NewClient(creds)
	gatewayTxID := uuid.New()

	dbTx, err := s.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		PaymentSessionID:     session.ID,
		InvoiceID:            invoice.ID,
		ProjectID:            invoice.ProjectID,
		TransactionType:      domain.TransactionTypePay,
		GatewayTransactionID: gatewayTxID.String(),
		Amount:               invoice.Amount,
		Currency:             invoice.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	resp, err := mpgsCli.ExecutePay(ctx, invoice.ID.String(), gatewayTxID.String(), &mpgsclient.ExecutePayRequest{
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
			ID: session.GatewaySessionID,
		},
	})
	if err != nil {
		return nil, &GatewayError{Gateway: "mpgs", Err: err}
	}

	result := &FinalizePaymentResult{
		CheckoutURL: fmt.Sprintf("/checkout/%s", req.InvoiceID),
	}

	if resp.Data.Response.GatewayCode == mpgsclient.CodeApproved {
		if err := s.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
			ID:          dbTx.ID,
			Status:      domain.TransactionStatusSuccess,
			RawResponse: resp.RawBody,
		}); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}

		if err := s.queries.UpdateInvoiceStatus(ctx, store.UpdateInvoiceStatusParams{
			ID:     invoice.ID,
			Status: domain.InvoiceStatusPaid,
		}); err != nil {
			return nil, fmt.Errorf("failed to update invoice: %w", err)
		}

		result.Success = true
		result.ResultCode = ResultSuccess
	} else {
		if err := s.queries.UpdateTransactionStatus(ctx, store.UpdateTransactionStatusParams{
			ID:          dbTx.ID,
			Status:      domain.TransactionStatusFailed,
			RawResponse: resp.RawBody,
		}); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}

		result.Success = false
		result.ResultCode = ResultDeclined
	}

	return result, nil
}
