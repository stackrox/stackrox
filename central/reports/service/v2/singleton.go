package v2

import (
	metadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	metadataStore := metadataDS.Singleton()
	snapshotDatastore := snapshotDS.Singleton()
	scheduler := schedulerV2.Singleton()

	// Start() also queues previously pending reports and scheduled reports, so running it in a separate routine to prevent
	// blocking main routine
	go scheduler.Start()
	svc = New(metadataStore, snapshotDatastore, scheduler)
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return svc
}
