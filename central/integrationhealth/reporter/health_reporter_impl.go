package reporter

import (
	"context"

	"github.com/google/martian/log"
	"github.com/stackrox/rox/central/integrationhealth/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	allAccessCtx = sac.WithAllAccess(context.Background())
	once         sync.Once
	reporter     IntegrationHealthReporter
)

type datastoreBasedIntegrationHealthReporter struct {
	integrationDS datastore.DataStore
}

func newDatastoreBasedIntegrationHealthReporter(datastore datastore.DataStore) *datastoreBasedIntegrationHealthReporter {
	return &datastoreBasedIntegrationHealthReporter{
		integrationDS: datastore,
	}
}

func (d *datastoreBasedIntegrationHealthReporter) UpdateIntegrationHealth(health *storage.IntegrationHealth) {
	if err := d.integrationDS.UpdateIntegrationHealth(allAccessCtx, health); err != nil {
		log.Errorf("Error updating health for integration %s (%s): %v", health.Name, health.Id, err)
	}
}

// Singleton returns an instance of the integration health reporter
func Singleton() IntegrationHealthReporter {
	once.Do(func() {
		reporter = newDatastoreBasedIntegrationHealthReporter(datastore.Singleton())
	})
	return reporter
}
