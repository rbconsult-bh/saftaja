package gateway

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
)

type MPGSCredentials struct {
	MerchantID  string `json:"merchant_id"`
	BaseURL     string `json:"base_url"`
	APIPassword string `json:"api_password"`
}

func ParseCredentials(data []byte) (*MPGSCredentials, error) {
	var c MPGSCredentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid MPGS credentials JSON: %w", err)
	}
	if c.MerchantID == "" {
		return nil, errors.New("missing merchant_id in MPGS credentials")
	}
	if c.BaseURL == "" {
		return nil, errors.New("missing base_url in MPGS credentials")
	}
	if c.APIPassword == "" {
		return nil, errors.New("missing api_password in MPGS credentials")
	}
	return &c, nil
}

func ParseEncryptedCredentials(encryptedData []byte, key []byte) (*MPGSCredentials, error) {
	decrypted, err := crypto.Decrypt(encryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt MPGS credentials: %w", err)
	}
	return ParseCredentials(decrypted)
}
