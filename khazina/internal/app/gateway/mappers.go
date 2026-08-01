package gateway

import (
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/store"
)

func mapConnectorTypeFromStore(ct store.ConnectorType) (ConnectorType, error) {
	switch ct {
	case store.ConnectorTypeMPGS:
		return ConnectorTypeMPGS, nil
	default:
		return "", fmt.Errorf("%w: unknown store connector type %q", ErrUnsupportedConnectorType, ct)
	}
}

func mapConnectorTypeToStore(ct ConnectorType) (store.ConnectorType, error) {
	switch ct {
	case ConnectorTypeMPGS:
		return store.ConnectorTypeMPGS, nil
	default:
		return "", fmt.Errorf("%w: unknown connector type %q", ErrUnsupportedConnectorType, ct)
	}
}
