package payment

import (
	"testing"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var gatewayTestEncryptionKey = []byte("12345678901234567890123456789012")

func TestGatewayResolver_CardGateway(t *testing.T) {
	account := validMPGSGatewayAccount(t, gatewayTestEncryptionKey, []byte(`{
		"base_url": "https://test.gateway.mastercard.com",
		"merchant_id": "TESTMERCHANT"
	}`), []byte(`{
		"api_password": "secret"
	}`))

	resolver := NewGatewayResolver(gatewayTestEncryptionKey)
	cardGateway, err := resolver.CardGateway(account)

	require.NoError(t, err)
	assert.IsType(t, &mpgsCardGateway{}, cardGateway)
}

func TestGatewayResolver_RejectsUnsupportedCardGateway(t *testing.T) {
	resolver := NewGatewayResolver(gatewayTestEncryptionKey)

	cardGateway, err := resolver.CardGateway(store.GatewayAccount{
		ConnectorType: "unsupported",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedGateway)
	assert.Nil(t, cardGateway)
}

func TestGatewayResolver_RejectsInvalidMPGSConfig(t *testing.T) {
	resolver := NewGatewayResolver(gatewayTestEncryptionKey)

	cardGateway, err := resolver.CardGateway(store.GatewayAccount{
		ConnectorType: store.ConnectorTypeMPGS,
		Config:        []byte(`{`),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mpgs config")
	assert.Nil(t, cardGateway)
}

func TestGatewayResolver_RejectsInvalidMPGSSecret(t *testing.T) {
	resolver := NewGatewayResolver(gatewayTestEncryptionKey)

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
