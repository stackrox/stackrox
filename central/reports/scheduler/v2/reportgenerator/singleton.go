package reportgenerator

import (
	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	rg   ReportGenerator
)

func initialize() {
	_, collectionQueryRes := collectionDS.Singleton()
	schema, err := graphql.ParseSchema(resolvers.Schema(), resolvers.New())
	utils.CrashOnError(err)
	rg = New(reportSnapshotDS.Singleton(),
		deploymentDS.Singleton(),
		watchedImageDS.Singleton(),
		collectionQueryRes,
		notifierProcessor.Singleton(),
		blobDS.Singleton(),
		schema,
	)
}

// Singleton returns a singleton instance of ReportGenerator
func Singleton() ReportGenerator {
	if !env.VulnReportingEnhancements.BooleanSetting() {
		return nil
	}
	once.Do(initialize)
	return rg
}
