package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const dnrIntegrationBucket = "dnrintegration"

// Store provides storage functionality for DNR integrations.
type Store interface {
	GetDNRIntegration(id string) (*v1.DNRIntegration, bool, error)
	GetDNRIntegrations(request *v1.GetDNRIntegrationsRequest) ([]*v1.DNRIntegration, error)
	AddDNRIntegration(integration *v1.DNRIntegration) (string, error)
	UpdateDNRIntegration(integration *v1.DNRIntegration) error
	RemoveDNRIntegration(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucket(db, dnrIntegrationBucket)
	return &storeImpl{
		DB: db,
	}
}
