package store

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/utils"
	bolt "go.etcd.io/bbolt"
)

var imageIntegrationBucket = []byte("imageintegrations")

// Store provides storage functionality for alerts.
type Store interface {
	GetImageIntegration(id string) (*storage.ImageIntegration, bool, error)
	GetImageIntegrations() ([]*storage.ImageIntegration, error)
	AddImageIntegration(integration *storage.ImageIntegration) (string, error)
	UpdateImageIntegration(integration *storage.ImageIntegration) error
	RemoveImageIntegration(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, imageIntegrationBucket)
	si := &storeImpl{
		DB: db,
	}

	integrations, err := si.GetImageIntegrations()
	utils.CrashOnError(err)
	if !env.OfflineModeEnv.BooleanSetting() && len(integrations) == 0 {
		// Add default integrations
		for _, ii := range DefaultImageIntegrations {
			utils.Must(si.UpdateImageIntegration(ii))
		}
	}
	return si
}
