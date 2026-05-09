package invoice

import (
	"github.com/google/uuid"
)

type InvoiceData struct {
	Invoice InvoiceInfo
	Items   []ItemInfo
	IsPaid  bool
}

type InvoiceInfo struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	Amount        string
	Currency      string
	Description   string
	CustomerEmail string
	CustomerName  string
}

type ItemInfo struct {
	Name      string
	Quantity  int32
	UnitPrice string
	Amount    string
}
