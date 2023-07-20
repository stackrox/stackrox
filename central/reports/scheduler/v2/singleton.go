package v2

import (
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	reportMetadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	sched Scheduler
)

func initialize() {
	collectionDatastore, _ := collectionDS.Singleton()
	sched = New(
		reportConfigDS.Singleton(),
		reportMetadataDS.Singleton(),
		collectionDatastore,
		reportGen.Singleton(),
	)
}

// Singleton will return a singleton instance of the v2 report scheduler
func Singleton() Scheduler {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return sched
}
