package reportgenerator

import (
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	imageCVE2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globaldb"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	rg   ReportGenerator
)

func initialize() {
	_, collectionQueryRes := collectionDS.Singleton()
	rg = New(globaldb.GetPostgres(),
		reportSnapshotDS.Singleton(),
		deploymentDS.Singleton(),
		watchedImageDS.Singleton(),
		collectionQueryRes,
		notifierProcessor.Singleton(),
		blobDS.Singleton(),
		clusterDS.Singleton(),
		namespaceDS.Singleton(),
		imageCVE2DS.Singleton(),
	)
}

// Singleton returns a singleton instance of ReportGenerator
func Singleton() ReportGenerator {
	once.Do(initialize)
	return rg
}
