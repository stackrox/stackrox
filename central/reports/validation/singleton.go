package validation

import (
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportSnapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	validator *Validator
)

func initialize() {
	collectionDatastore, _ := collectionDS.Singleton()
	validator = New(reportConfigDS.Singleton(), reportSnapshotDS.Singleton(), collectionDatastore, notifierDS.Singleton())
}

// Singleton returns a singleton instance of Validator
func Singleton() *Validator {
	if !features.VulnReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return validator
}
