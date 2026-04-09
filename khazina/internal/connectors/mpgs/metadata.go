package mpgs

import (
	"golang.org/x/text/language"

	"github.com/rbconsult-bh/saftaja/khazina/internal/app/domain"
	"github.com/rbconsult-bh/saftaja/khazina/internal/connectors"
)

func init() {
	connectors.Register(Metadata)
}

var Metadata = connectors.ConnectorMeta{
	Type: domain.ConnectorTypeMPGS,
	Name: domain.LocalizedString{
		language.English: "Mastercard Payment Gateway",
		language.Arabic:  "بوابة ماستركارد للدفع",
	},
	Description: domain.LocalizedString{
		language.English: "Accept card payments via MPGS",
		language.Arabic:  "قبول المدفوعات بالبطاقة عبر MPGS",
	},
	CredentialFields: []connectors.Field{
		{
			Key:      "merchant_id",
			Label:    domain.LocalizedString{language.English: "Merchant ID", language.Arabic: "معرف التاجر"},
			Type:     connectors.FieldTypeText,
			Required: true,
		},
		{
			Key:      "base_url",
			Label:    domain.LocalizedString{language.English: "API Base URL", language.Arabic: "رابط API الأساسي"},
			Type:     connectors.FieldTypeURL,
			Required: true,
		},
		{
			Key:      "api_password",
			Label:    domain.LocalizedString{language.English: "API Password", language.Arabic: "كلمة مرور API"},
			Type:     connectors.FieldTypePassword,
			Required: true,
		},
	},
	PaymentMethods: []connectors.PaymentMethodMeta{
		{
			Method: domain.PaymentMethodCard,
			Label:  domain.LocalizedString{language.English: "Credit/Debit Card", language.Arabic: "بطاقة ائتمان/خصم"},
			Icon:   "credit-card",
		},
		{
			Method: domain.PaymentMethodApplePay,
			Label:  domain.LocalizedString{language.English: "Apple Pay", language.Arabic: "أبل باي"},
			Icon:   "apple",
		},
	},
}
