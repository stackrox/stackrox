package reporter

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/integrationhealth/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

var (
	allAccessCtx = sac.WithAllAccess(context.Background())
	once         sync.Once
	reporter     integrationhealth.Reporter
)

// DatastoreBasedIntegrationHealthReporter updates the integration health in the central database
type DatastoreBasedIntegrationHealthReporter struct {
	healthUpdates        chan *storage.IntegrationHealth
	stopSig              concurrency.Signal
	latestDBTimestampMap map[string]*types.Timestamp
	lock                 sync.RWMutex
	integrationDS        datastore.DataStore
}

// New returns a new datastore based integration health reporter
func New(datastore datastore.DataStore) *DatastoreBasedIntegrationHealthReporter {
	d := &DatastoreBasedIntegrationHealthReporter{
		healthUpdates:        make(chan *storage.IntegrationHealth, 5),
		stopSig:              concurrency.NewSignal(),
		latestDBTimestampMap: make(map[string]*types.Timestamp),
		integrationDS:        datastore,
	}
	go d.processIntegrationHealthUpdates()
	return d
}

// Register registers the integration health for an integration, if it doesn't exist
func (d *DatastoreBasedIntegrationHealthReporter) Register(id, name string, typ storage.IntegrationHealth_Type) error {
	_, exists, err := d.integrationDS.GetIntegrationHealth(allAccessCtx, id)
	if err != nil {
		log.Errorf("Error getting health for integration %s (%s): %v", name, id, err)
		return err
	}

	if exists {
		// Do nothing
		return nil
	}

	now := types.TimestampNow()
	// integration health for said integration does not exists, initialize it
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
	err := d.integrationDS.RemoveIntegrationHealth(allAccessCtx, id)
	if err == nil {
		d.lock.Lock()
		defer d.lock.Unlock()
		delete(d.latestDBTimestampMap, id)
	}
	return err
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

func (d *DatastoreBasedIntegrationHealthReporter) processIntegrationHealthUpdates() {
	for {
		select {
		case health := <-d.healthUpdates:
			if health.Status == storage.IntegrationHealth_UNINITIALIZED {
				d.updateTimestampInCache(health)
				if err := d.integrationDS.UpdateIntegrationHealth(allAccessCtx, health); err != nil {
					log.Errorf("Error updating health for integration %s (%s): %v", health.Name, health.Id, err)
				}
			} else if health.LastTimestamp.Compare(d.getTimestampInCache(health)) > 0 {
				d.updateTimestampInCache(health)
				_, exists, err := d.integrationDS.GetIntegrationHealth(allAccessCtx, health.Id)
				if err != nil {
					log.Errorf("Error reading health for integration %s (%s): %v", health.Name, health.Id, err)
					continue
				} else if !exists {
					// ignore health update since integration has possibly been deleted
					continue
				}
				if err := d.integrationDS.UpdateIntegrationHealth(allAccessCtx, health); err != nil {
					log.Errorf("Error updating health for integration %s (%s): %v", health.Name, health.Id, err)
				}
			}

		case <-d.stopSig.Done():
			return
		}
	}
}

func (d *DatastoreBasedIntegrationHealthReporter) updateTimestampInCache(health *storage.IntegrationHealth) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.latestDBTimestampMap[health.Id] = health.LastTimestamp
}

func (d *DatastoreBasedIntegrationHealthReporter) getTimestampInCache(health *storage.IntegrationHealth) *types.Timestamp {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.latestDBTimestampMap[health.Id]
}

// Singleton returns an instance of the integration health reporter
func Singleton() integrationhealth.Reporter {
	once.Do(func() {
		reporter = New(datastore.Singleton())
	})
	return reporter
}
