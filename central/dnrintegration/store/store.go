package store

import (
	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
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
	bolthelper.RegisterBucketOrPanic(db, dnrIntegrationBucket)
	return &storeImpl{
		DB: db,
	}
}
