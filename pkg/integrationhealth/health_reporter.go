package integrationhealth

import (
	"github.com/stackrox/rox/generated/storage"
)

// Reporter is an interface to report integration health updates and deletes
//
//go:generate mockgen-wrapper
type Reporter interface {
	Register(id, name string, typ storage.IntegrationHealth_Type) error
	UpdateIntegrationHealthAsync(*storage.IntegrationHealth)
	RetrieveIntegrationHealths(typ storage.IntegrationHealth_Type) ([]*storage.IntegrationHealth, error)
	RemoveIntegrationHealth(id string) error
}
