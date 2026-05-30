package gateway

import "github.com/rbconsult-bh/saftaja/khazina/internal/store"

func mapConnectorTypeFromStore(ct store.ConnectorType) ConnectorType {
	switch ct {
	case store.ConnectorTypeMPGS:
		return ConnectorTypeMPGS
	default:
		return ConnectorTypeUnknown
	}
}

func mapConnectorTypeToStore(ct ConnectorType) store.ConnectorType {
	switch ct {
	case ConnectorTypeMPGS:
		return store.ConnectorTypeMPGS
	default:
		return ""
	}
}
