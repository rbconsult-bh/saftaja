package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/resendlabs/resend-go"
	"github.com/rs/zerolog/log"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/auth"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/billing"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/gateway"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/membership"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/payment"
	"github.com/rbconsult-bh/saftaja/khazina/internal/app/tenant"
	"github.com/rbconsult-bh/saftaja/khazina/internal/clients/email"
	"github.com/rbconsult-bh/saftaja/khazina/internal/config"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/jwt"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/dashboard"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/dashboard/authv1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/dashboard/projectv1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/dashboard/workspacev1"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/auth/v1/authpbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/project/v1/projectpbv1connect"
	"github.com/rbconsult-bh/saftaja/khazina/internal/transport/proto/saftaja/dashboard/workspace/v1/workspacepbv1connect"
)

type dependencies struct {
	dbPool                *pgxpool.Pool
	tenantSvc             tenant.Service
	paymentSvc            payment.Service
	billingSvc            billing.Service
	gatewaySvc            gateway.Service
	authSvc               auth.AuthService
	dashboardAuthSvc      authpbv1connect.AuthServiceHandler
	membershipSvc         membership.MembershipService
	dashboardWorkspaceSvc workspacepbv1connect.WorkspaceServiceHandler
	dashboardProjectSvc   projectpbv1connect.ProjectServiceHandler
	authInterceptor       connect.UnaryInterceptorFunc
	emailer               email.Emailer
	emailTemplates        email.Templates
	encryptionKey         []byte
	cfg                   *config.Config
}

func (d *dependencies) Cleanup() {
	d.dbPool.Close()
}

func InitDependencies(ctx context.Context, cfg *config.Config) (*dependencies, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBDatabase)

	log.Info().Msg("running database migrations...")
	if err := store.RunMigrations(dsn); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Info().Msg("migrations completed")

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse db config: %w", err)
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	queries := store.NewTransactionQuerier(dbPool)

	encryptionKey, err := cfg.GetEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	gatewayResolver := billing.NewGatewayResolver(encryptionKey)

	tenantSvc := tenant.NewService(queries)
	paymentSvc := payment.NewService(dbPool, queries, encryptionKey)
	billingSvc := billing.New(dbPool, queries, gatewayResolver)
	gatewaySvc := gateway.New(queries, encryptionKey)

	emailer, err := initEmailer(cfg)
	if err != nil {
		return nil, err
	}

	emailTemplates := email.NewTemplates(cfg.Domain)

	jwtIssuer, jwtVerifier, err := jwt.ParseJWTPrivateKey(cfg.JWTPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jwt private key: %w", err)
	}

	authInterceptor := dashboard.NewAuthTokenInterceptor(jwtVerifier)
	authSvc := auth.New(dbPool, queries, emailer, emailTemplates, jwtIssuer, jwtVerifier)
	dashboardAuthSvc := authv1.New(authSvc)

	membershipSvc := membership.New(dbPool, queries)
	dashboardWorkspaceSvc := workspacev1.New(membershipSvc)

	dashboardProjectSvc := projectv1.New(gatewaySvc, membershipSvc)

	return &dependencies{
		dbPool:                dbPool,
		tenantSvc:             tenantSvc,
		paymentSvc:            paymentSvc,
		billingSvc:            billingSvc,
		gatewaySvc:            gatewaySvc,
		authSvc:               authSvc,
		dashboardAuthSvc:      dashboardAuthSvc,
		membershipSvc:         membershipSvc,
		dashboardWorkspaceSvc: dashboardWorkspaceSvc,
		dashboardProjectSvc:   dashboardProjectSvc,
		authInterceptor:       authInterceptor,
		emailer:               emailer,
		emailTemplates:        emailTemplates,
		encryptionKey:         encryptionKey,
		cfg:                   cfg,
	}, nil
}

func initEmailer(cfg *config.Config) (email.Emailer, error) {
	switch cfg.Emailer {
	case string(email.EmailerName_Resend):
		resendCli := resend.NewClient(cfg.ResendAPIKey)
		return email.NewResendEmailer(resendCli), nil
	case string(email.EmailerName_Stdout):
		return email.NewStdoutEmailer(), nil
	default:
		return nil, fmt.Errorf("emailer is not expected: %v", cfg.Emailer)
	}
}
