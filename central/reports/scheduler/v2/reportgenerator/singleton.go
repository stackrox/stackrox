package reportgenerator

import (
	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	reportMetadataDS "github.com/stackrox/rox/central/reports/metadata/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	rg   ReportGenerator
)

func initialize() {
	collectionDatastore, collectionQueryRes := collectionDS.Singleton()
	schema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	utils.CrashOnError(err)
	rg = New(reportConfigDS.Singleton(),
		reportMetadataDS.Singleton(),
		reportSnapshotDS.Singleton(),
		deploymentDS.Singleton(),
		watchedImageDS.Singleton(),
		collectionDatastore,
		collectionQueryRes,
		notifierDS.Singleton(),
		notifierProcessor.Singleton(),
		blobDS.Singleton(),
		schema,
	)
}

// Singleton returns a singleton instance of ReportGenerator
func Singleton() ReportGenerator {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return rg
}
