package reporter

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/integrationhealth/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

var (
	integrationWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	once     sync.Once
	reporter integrationhealth.Reporter
)

// DatastoreBasedIntegrationHealthReporter updates the integration health in the central database
type DatastoreBasedIntegrationHealthReporter struct {
	healthUpdates chan *storage.IntegrationHealth
	healthRemoval chan string

	stopSig              concurrency.Signal
	latestDBTimestampMap map[string]*types.Timestamp
	integrationDS        datastore.DataStore
}

// New returns a new datastore based integration health reporter
func New(datastore datastore.DataStore) *DatastoreBasedIntegrationHealthReporter {
	d := &DatastoreBasedIntegrationHealthReporter{
		healthUpdates:        make(chan *storage.IntegrationHealth, 5),
		healthRemoval:        make(chan string, 5),
		stopSig:              concurrency.NewSignal(),
		latestDBTimestampMap: make(map[string]*types.Timestamp),
		integrationDS:        datastore,
	}
	go d.processIntegrationHealthUpdates()
	return d
}

// Register registers the integration health for an integration, if it doesn't exist
func (d *DatastoreBasedIntegrationHealthReporter) Register(id, name string, typ storage.IntegrationHealth_Type) error {
	_, exists, err := d.integrationDS.GetIntegrationHealth(integrationWriteCtx, id)
	if err != nil {
		log.Errorf("Error getting health for integration %s (%s): %v", name, id, err)
		return err
	}

	if exists {
		// Do nothing
		return nil
	}

	now := types.TimestampNow()
	// Integration health does not exist yet, initialize it.
	d.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
		Id:            id,
		Name:          name,
		Type:          typ,
		Status:        storage.IntegrationHealth_UNINITIALIZED,
		LastTimestamp: now,
		ErrorMessage:  "",
	})

	return nil
}

// RemoveIntegrationHealth removes the health entry corresponding to the integration
func (d *DatastoreBasedIntegrationHealthReporter) RemoveIntegrationHealth(id string) error {
	if err := d.integrationDS.RemoveIntegrationHealth(integrationWriteCtx, id); err != nil {
		return errors.Wrapf(err, "Error removing health for integration %s", id)
	}
	select {
	case d.healthRemoval <- id:
		return nil
	case <-d.stopSig.Done():
		return nil
	}
}

// UpdateIntegrationHealthAsync updates the health of the integration
func (d *DatastoreBasedIntegrationHealthReporter) UpdateIntegrationHealthAsync(health *storage.IntegrationHealth) {
	select {
	case d.healthUpdates <- health:
		return
	case <-d.stopSig.Done():
		return
	}
}

// RetrieveIntegrationHealths retrieves the integration healths for a specific type.
func (d *DatastoreBasedIntegrationHealthReporter) RetrieveIntegrationHealths(typ storage.IntegrationHealth_Type) ([]*storage.IntegrationHealth, error) {
	switch typ {
	case storage.IntegrationHealth_DECLARATIVE_CONFIG:
		return d.integrationDS.GetDeclarativeConfigs(integrationWriteCtx)
	case storage.IntegrationHealth_IMAGE_INTEGRATION:
		return d.integrationDS.GetRegistriesAndScanners(integrationWriteCtx)
	case storage.IntegrationHealth_BACKUP:
		return d.integrationDS.GetBackupPlugins(integrationWriteCtx)
	case storage.IntegrationHealth_NOTIFIER:
		return d.integrationDS.GetNotifierPlugins(integrationWriteCtx)
	}
	return nil, errox.InvalidArgs.Newf("type %s is not supported", typ)
}

func (d *DatastoreBasedIntegrationHealthReporter) processIntegrationHealthUpdates() {
	for {
		select {
		case health := <-d.healthUpdates:
			if health.Status == storage.IntegrationHealth_UNINITIALIZED {
				d.latestDBTimestampMap[health.Id] = health.LastTimestamp
				if err := d.integrationDS.UpsertIntegrationHealth(integrationWriteCtx, health); err != nil {
					log.Errorf("Error updating health for integration %s (%s): %v", health.Name, health.Id, err)
				}
			} else if health.LastTimestamp.Compare(d.latestDBTimestampMap[health.Id]) > 0 {
				d.latestDBTimestampMap[health.Id] = health.LastTimestamp
				_, exists, err := d.integrationDS.GetIntegrationHealth(integrationWriteCtx, health.Id)
				if err != nil {
					log.Errorf("Error reading health for integration %s (%s): %v", health.Name, health.Id, err)
					continue
				} else if !exists {
					// Ignore health update since integration has possibly been deleted.
					continue
				}
				if err := d.integrationDS.UpsertIntegrationHealth(integrationWriteCtx, health); err != nil {
					log.Errorf("Error updating health for integration %s (%s): %v", health.Name, health.Id, err)
				}
			}
		case id := <-d.healthRemoval:
			delete(d.latestDBTimestampMap, id)

		case <-d.stopSig.Done():
			return
		}
	}
}

// Singleton returns an instance of the integration health reporter
func Singleton() integrationhealth.Reporter {
	once.Do(func() {
		reporter = New(datastore.Singleton())
	})
	return reporter
}
