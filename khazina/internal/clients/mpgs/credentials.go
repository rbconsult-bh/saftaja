package mpgs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
)

type Credentials struct {
	MerchantID  string `json:"merchant_id"`
	BaseURL     string `json:"base_url"`
	APIPassword string `json:"api_password"`
}

func ParseCredentials(data []byte) (*Credentials, error) {
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
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

func ParseEncryptedCredentials(encrypted []byte, key []byte) (*Credentials, error) {
	decrypted, err := crypto.Decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}
	return ParseCredentials(decrypted)
}
