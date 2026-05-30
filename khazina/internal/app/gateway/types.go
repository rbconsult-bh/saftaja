package gateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

var ErrInvalidArgument = errors.New("invalid argument")

type Service interface {
	ListActiveByProject(ctx context.Context, r ListActiveByProjectRequest) (*ListActiveByProjectResponse, error)
	ListPaymentMethods(ctx context.Context, r ListPaymentMethodsRequest) (*ListPaymentMethodsResponse, error)
	Create(ctx context.Context, req CreateGatewayRequest) (*GatewayAccount, error)
}

type PaymentMethodType string

const (
	PaymentMethodTypeCard     PaymentMethodType = "card"
	PaymentMethodTypeApplePay PaymentMethodType = "apple_pay"
)

type PaymentMethod struct {
	Type             PaymentMethodType
	GatewayAccountID uuid.UUID
	MPGSBaseURL      string
	MPGSMerchantID   string
	MPGSApiVersion   string
}

type ListPaymentMethodsRequest struct {
	ProjectID uuid.UUID
}

func (r *ListPaymentMethodsRequest) Validate() error {
	if r.ProjectID == uuid.Nil {
		return fmt.Errorf("%w: ProjectID is required", ErrInvalidArgument)
	}
	return nil
}

type ListPaymentMethodsResponse struct {
	PaymentMethods []PaymentMethod
}

type ListActiveByProjectRequest struct {
	ProjectID uuid.UUID
}

type ListActiveByProjectResponse struct {
	Gateways []GatewayCredentials
}

type GatewayCredentials struct {
	GatewayAccountID uuid.UUID
	AccountName      string
	ConnectorType    domain.ConnectorType
	BaseURL          string
	MerchantID       string
}

type CreateGatewayRequest struct {
	ProjectID     uuid.UUID
	AccountName   string
	ConnectorType domain.ConnectorType
	Config        *mpgsclient.Config
	Secret        *mpgsclient.Secret
}

type GatewayAccount struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	AccountName   string
	ConnectorType domain.ConnectorType
	IsActive      bool
}
