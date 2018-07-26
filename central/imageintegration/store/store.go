package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const imageIntegrationBucket = "imageintegrations"

// Store provides storage functionality for alerts.
type Store interface {
	GetImageIntegration(id string) (*v1.ImageIntegration, bool, error)
	GetImageIntegrations(integration *v1.GetImageIntegrationsRequest) ([]*v1.ImageIntegration, error)
	AddImageIntegration(integration *v1.ImageIntegration) (string, error)
	UpdateImageIntegration(integration *v1.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, imageIntegrationBucket)
	return &storeImpl{
		DB: db,
	}
}
