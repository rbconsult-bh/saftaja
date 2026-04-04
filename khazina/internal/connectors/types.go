package connectors

import "github.com/rbconsult-bh/saftaja/khazina/internal/domain"

type FieldType string

const (
	FieldTypeText     FieldType = "text"
	FieldTypePassword FieldType = "password"
	FieldTypeURL      FieldType = "url"
	FieldTypeSelect   FieldType = "select"
	FieldTypeCheckbox FieldType = "checkbox"
)

type Field struct {
	Key         string                 `json:"key"`
	Label       domain.LocalizedString `json:"label"`
	Placeholder domain.LocalizedString `json:"placeholder,omitempty"`
	Type        FieldType              `json:"type"`
	Required    bool                   `json:"required"`
}

type PaymentMethodMeta struct {
	Method domain.PaymentMethod   `json:"method"`
	Label  domain.LocalizedString `json:"label"`
	Icon   string                 `json:"icon,omitempty"`
}

type ConnectorMeta struct {
	Type             domain.ConnectorType   `json:"type"`
	Name             domain.LocalizedString `json:"name"`
	Description      domain.LocalizedString `json:"description,omitempty"`
	CredentialFields []Field                `json:"credential_fields"`
	SettingsFields   []Field                `json:"settings_fields,omitempty"`
	PaymentMethods   []PaymentMethodMeta    `json:"payment_methods"`
}
