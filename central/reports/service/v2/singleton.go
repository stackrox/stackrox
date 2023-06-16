package v2

import (
	metadataDataStore "github.com/stackrox/rox/central/reports/metadata/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	metadataDS := metadataDataStore.Singleton()
	svc = New(metadataDS)
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
