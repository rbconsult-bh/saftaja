package membership

import "github.com/rbconsult-bh/saftaja/khazina/internal/store"

func mapEnvironmentFromStore(env store.ProjectEnvironment) ProjectEnvironment {
	switch env {
	case store.ProjectEnvironmentSandbox:
		return EnvironmentSandbox
	case store.ProjectEnvironmentProduction:
		return EnvironmentProduction
	default:
		return EnvironmentUnknown
	}
}
