package billing

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	mpgsmocks "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs/mocks"
	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMPGSCardGateway_CreateSession(t *testing.T) {
	ctx := context.Background()
	invoiceID := uuid.MustParse("00000000-0000-0000-0000-000000003001")

	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		CreateSession(mock.Anything, mock.MatchedBy(func(req *mpgsclient.CreateSessionRequest) bool {
			return req.Session != nil &&
				req.Session.AuthenticationLimit != nil &&
				*req.Session.AuthenticationLimit == 25
		})).
		Return(&mpgsclient.Response[mpgsclient.CreateSessionResponse]{
			Data: mpgsclient.CreateSessionResponse{
				Session: &mpgsclient.SessionDetails{ID: "SESSION123"},
			},
		}, nil)

	mpgsClient.EXPECT().
		UpdateSession(mock.Anything, "SESSION123", mock.MatchedBy(func(req *mpgsclient.UpdateSessionRequest) bool {
			return req.Order.Amount == "15" &&
				req.Order.Currency == "BHD" &&
				req.Order.ID == invoiceID.String()
		})).
		Return(&mpgsclient.Response[mpgsclient.UpdateSessionResponse]{}, nil)

	gateway := &mpgsCardGateway{client: mpgsClient}
	resp, err := gateway.CreateSession(ctx, CreateSessionRequest{
		InvoiceID: invoiceID,
		Amount:    decimal.RequireFromString("15.000"),
		Currency:  "BHD",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "SESSION123", resp.GatewaySessionID)
}

func TestMPGSCardGateway_ReturnsCreateSessionError(t *testing.T) {
	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		CreateSession(mock.Anything, mock.Anything).
		Return(nil, errors.New("create failed"))

	gateway := &mpgsCardGateway{client: mpgsClient}
	resp, err := gateway.CreateSession(context.Background(), validCreateSessionRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create mpgs session")
	assert.Nil(t, resp)
}

func TestMPGSCardGateway_ReturnsMissingSessionIDError(t *testing.T) {
	tests := []struct {
		name string
		resp *mpgsclient.Response[mpgsclient.CreateSessionResponse]
	}{
		{
			name: "nil_response",
			resp: nil,
		},
		{
			name: "nil_session",
			resp: &mpgsclient.Response[mpgsclient.CreateSessionResponse]{
				Data: mpgsclient.CreateSessionResponse{},
			},
		},
		{
			name: "empty_session_id",
			resp: &mpgsclient.Response[mpgsclient.CreateSessionResponse]{
				Data: mpgsclient.CreateSessionResponse{
					Session: &mpgsclient.SessionDetails{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpgsClient := mpgsmocks.NewMockClient(t)
			mpgsClient.EXPECT().
				CreateSession(mock.Anything, mock.Anything).
				Return(tt.resp, nil)

			gateway := &mpgsCardGateway{client: mpgsClient}
			resp, err := gateway.CreateSession(context.Background(), validCreateSessionRequest())

			require.Error(t, err)
			assert.Contains(t, err.Error(), "missing session id")
			assert.Nil(t, resp)
		})
	}
}

func TestMPGSCardGateway_ReturnsUpdateSessionError(t *testing.T) {
	mpgsClient := mpgsmocks.NewMockClient(t)
	mpgsClient.EXPECT().
		CreateSession(mock.Anything, mock.Anything).
		Return(&mpgsclient.Response[mpgsclient.CreateSessionResponse]{
			Data: mpgsclient.CreateSessionResponse{
				Session: &mpgsclient.SessionDetails{ID: "SESSION123"},
			},
		}, nil)

	mpgsClient.EXPECT().
		UpdateSession(mock.Anything, "SESSION123", mock.Anything).
		Return(nil, errors.New("update failed"))

	gateway := &mpgsCardGateway{client: mpgsClient}
	resp, err := gateway.CreateSession(context.Background(), validCreateSessionRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update mpgs session")
	assert.Nil(t, resp)
}

func TestGatewayResolver_CardGateway(t *testing.T) {
	encryptionKey := []byte("12345678901234567890123456789012")
	account := validMPGSGatewayAccount(t, encryptionKey, []byte(`{
		"base_url": "https://test.gateway.mastercard.com",
		"merchant_id": "TESTMERCHANT"
	}`), []byte(`{
		"api_password": "secret"
	}`))

	resolver := NewGatewayResolver(encryptionKey)
	cardGateway, err := resolver.CardGateway(account)

	require.NoError(t, err)
	assert.IsType(t, &mpgsCardGateway{}, cardGateway)
}

func TestGatewayResolver_RejectsUnsupportedCardGateway(t *testing.T) {
	resolver := NewGatewayResolver(testEncryptionKey)

	cardGateway, err := resolver.CardGateway(store.GatewayAccount{
		ConnectorType: "unsupported",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedGateway)
	assert.Nil(t, cardGateway)
}

func TestGatewayResolver_RejectsInvalidMPGSConfig(t *testing.T) {
	resolver := NewGatewayResolver(testEncryptionKey)

	cardGateway, err := resolver.CardGateway(store.GatewayAccount{
		ConnectorType: store.ConnectorTypeMPGS,
		Config:        []byte(`{`),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mpgs config")
	assert.Nil(t, cardGateway)
}

func TestGatewayResolver_RejectsInvalidMPGSSecret(t *testing.T) {
	resolver := NewGatewayResolver(testEncryptionKey)

	cardGateway, err := resolver.CardGateway(store.GatewayAccount{
		ConnectorType: store.ConnectorTypeMPGS,
		Config: []byte(`{
			"base_url": "https://test.gateway.mastercard.com",
			"merchant_id": "TESTMERCHANT"
		}`),
		Secret: []byte("not encrypted"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mpgs secret")
	assert.Nil(t, cardGateway)
}

func validCreateSessionRequest() CreateSessionRequest {
	return CreateSessionRequest{
		InvoiceID: uuid.MustParse("00000000-0000-0000-0000-000000003001"),
		Amount:    decimal.RequireFromString("15.000"),
		Currency:  "BHD",
	}
}

func validMPGSGatewayAccount(t *testing.T, encryptionKey, config, secret []byte) store.GatewayAccount {
	t.Helper()

	encryptedSecret, err := crypto.Encrypt(secret, encryptionKey)
	require.NoError(t, err)

	return store.GatewayAccount{
		ConnectorType: store.ConnectorTypeMPGS,
		Config:        config,
		Secret:        encryptedSecret,
	}
}
