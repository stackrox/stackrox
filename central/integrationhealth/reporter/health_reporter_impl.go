package reporter

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/integrationhealth/datastore"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/integrationhealth"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
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
	if err := d.integrationDS.RemoveIntegrationHealth(allAccessCtx, id); err != nil {
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

func (d *DatastoreBasedIntegrationHealthReporter) processIntegrationHealthUpdates() {
	for {
		select {
		case health := <-d.healthUpdates:
			if health.Status == storage.IntegrationHealth_UNINITIALIZED {
				d.latestDBTimestampMap[health.Id] = health.LastTimestamp
				if err := d.integrationDS.UpdateIntegrationHealth(allAccessCtx, health); err != nil {
					log.Errorf("Error updating health for integration %s (%s): %v", health.Name, health.Id, err)
				}
			} else if health.LastTimestamp.Compare(d.latestDBTimestampMap[health.Id]) > 0 {
				d.latestDBTimestampMap[health.Id] = health.LastTimestamp
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
