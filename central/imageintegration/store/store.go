package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const imageIntegrationBucket = "imageintegrations"

// Store provides storage functionality for alerts.
type Store interface {
	GetImageIntegration(id string) (*storage.ImageIntegration, bool, error)
	GetImageIntegrations(integration *v1.GetImageIntegrationsRequest) ([]*storage.ImageIntegration, error)
	AddImageIntegration(integration *storage.ImageIntegration) (string, error)
	UpdateImageIntegration(integration *storage.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, imageIntegrationBucket)
	return &storeImpl{
		DB: db,
	}
}
