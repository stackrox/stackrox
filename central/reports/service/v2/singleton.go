package v2

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/globaldb"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	scheduler := schedulerV2.Singleton()
	if !env.CentralWorkerEnabled.BooleanSetting() {
		go scheduler.Start()
	} else {
		log.Info("Report scheduling is managed by central-worker, skipping start in Central")
	}
	collectionDatastore, _ := collectionDS.Singleton()
	svc = New(reportConfigDS.Singleton(), snapshotDS.Singleton(), collectionDatastore, notifierDS.Singleton(), scheduler,
		blobDS.Singleton(), validation.Singleton(), globaldb.GetPostgres())
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
