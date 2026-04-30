package service

import (
	imagecvev2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	srv Service
)

func initialize() {
	// imageDS.Singleton() is the mapping datastore used by the v1 image service.
	// It handles both the legacy images table and the imagev2 table transparently,
	// so the service works regardless of the ROX_FLATTEN_IMAGE_DATA feature flag.
	srv = New(imageDS.Singleton(), imagecvev2DS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return srv
}
