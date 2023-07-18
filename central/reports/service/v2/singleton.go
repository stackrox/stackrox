package v2

import (
	metadataDataStore "github.com/stackrox/rox/central/reports/metadata/datastore"
	snapshotDataStore "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	metadataDS := metadataDataStore.Singleton()
	snapshotDS := snapshotDataStore.Singleton()
	svc = New(metadataDS, snapshotDS)
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return svc
}
