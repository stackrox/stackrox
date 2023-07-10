package v2

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	reportMetadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	sched Scheduler
)

func initialize() {
	collectionDatastore, collectionQueryRes := collectionDS.Singleton()
	sched = New(
		reportConfigDS.Singleton(),
		reportMetadataDS.Singleton(),
		reportSnapshotDS.Singleton(),
		notifierDS.Singleton(),
		deploymentDS.Singleton(),
		watchedImageDS.Singleton(),
		collectionDatastore,
		collectionQueryRes,
		notifierProcessor.Singleton(),
	)
}

func Singleton() Scheduler {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return sched
}
