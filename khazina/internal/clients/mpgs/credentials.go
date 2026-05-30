package mpgs

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/pkg/crypto"
)

// Config holds public gateway configuration stored in the config column (JSONB, plain text).
type Config struct {
	BaseURL    string `json:"base_url"`
	MerchantID string `json:"merchant_id"`
}

// Secret holds encrypted gateway credentials stored in the secret column (bytea, encrypted at rest).
type Secret struct {
	APIPassword string `json:"api_password"`
}

func ParseConfig(data []byte) (*Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if c.MerchantID == "" {
		return nil, errors.New("missing merchant_id in MPGS config")
	}
	if c.BaseURL == "" {
		return nil, errors.New("missing base_url in MPGS config")
	}
	return &c, nil
}

func DecryptSecret(encrypted []byte, key []byte) (*Secret, error) {
	decrypted, err := crypto.Decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}
	var s Secret
	if err := json.Unmarshal(decrypted, &s); err != nil {
		return nil, fmt.Errorf("failed to parse secret: %w", err)
	}
	if s.APIPassword == "" {
		return nil, errors.New("missing api_password in MPGS secret")
	}
	return &s, nil
}
