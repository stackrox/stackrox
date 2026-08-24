package node

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	svc  Service
)

// Singleton returns the singleton instance of the node report service.
func Singleton() Service {
	once.Do(func() {
		collectionDatastore, _ := collectionDS.Singleton()
		svc = New(
			reportConfigDS.Singleton(),
			snapshotDS.Singleton(),
			collectionDatastore,
			notifierDS.Singleton(),
			schedulerV2.Singleton(),
			blobDS.Singleton(),
			validation.Singleton(),
		)
	})
	return svc
}
