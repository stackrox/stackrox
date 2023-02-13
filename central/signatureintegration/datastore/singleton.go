package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	pgStore "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	"github.com/stackrox/rox/central/signatureintegration/store/rocksdb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		var storage store.SignatureIntegrationStore
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			storage = pgStore.New(globaldb.GetPostgres())
		} else {
			var err error
			storage, err = rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(errors.Wrap(err, "unable to create rocksdb store for signature integrations"))
		}
		instance = New(storage, policyDataStore.Singleton())
	})
	return instance
}
