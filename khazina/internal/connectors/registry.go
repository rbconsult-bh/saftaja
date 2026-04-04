package connectors

import (
	"fmt"

	"github.com/rbconsult-bh/saftaja/khazina/internal/domain"
)

var Registry = make(map[domain.ConnectorType]ConnectorMeta)

func Register(meta ConnectorMeta) {
	Registry[meta.Type] = meta
}

func Get(t domain.ConnectorType) (ConnectorMeta, error) {
	meta, ok := Registry[t]
	if !ok {
		return ConnectorMeta{}, fmt.Errorf("unknown connector type: %s", t)
	}
	return meta, nil
}

func Available() []ConnectorMeta {
	result := make([]ConnectorMeta, 0, len(Registry))
	for _, meta := range Registry {
		result = append(result, meta)
	}
	return result
}
