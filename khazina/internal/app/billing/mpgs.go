package billing

import (
	"fmt"

	mpgsclient "github.com/rbconsult-bh/saftaja/khazina/internal/clients/mpgs"
	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

func (s *service) newMPGSClient(account store.GatewayAccount) (mpgsclient.Client, error) {
	cfg, err := mpgsclient.ParseConfig(account.Config)
	if err != nil {
		return nil, fmt.Errorf("invalud gateway config: %w", err)
	}

	sec, err := mpgsclient.DecryptSecret(account.Secret, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid gateway credentals: %w", err)
	}

	return mpgsclient.New(
		cfg.BaseURL,
		cfg.MerchantID,
		sec.APIPassword,
	), nil
}
