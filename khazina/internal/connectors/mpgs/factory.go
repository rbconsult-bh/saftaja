package mpgs

import mpgsclient "github.com/rbconsult-bh/saftaja/internal/clients/mpgs"

func NewClient(creds *Credentials) mpgsclient.Client {
	return mpgsclient.New(creds.BaseURL, creds.MerchantID, creds.APIPassword)
}
