package v2

import (
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
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
		reportSnapshotDS.Singleton(),
		collectionDatastore,
		notifierDS.Singleton(),
		reportGen.Singleton(),
		validation.Singleton(),
	)
}

// Singleton will return a singleton instance of the v2 report scheduler
func Singleton() Scheduler {
	once.Do(initialize)
	return sched
}
