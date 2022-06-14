package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/globaldb"
	policyDataStore "github.com/stackrox/stackrox/central/policy/datastore"
	"github.com/stackrox/stackrox/central/signatureintegration/store"
	"github.com/stackrox/stackrox/central/signatureintegration/store/postgres"
	"github.com/stackrox/stackrox/central/signatureintegration/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		var storage store.SignatureIntegrationStore
		if features.PostgresDatastore.Enabled() {
			storage = postgres.New(globaldb.GetPostgres())
		} else {
			var err error
			storage, err = rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(errors.Wrap(err, "unable to create rocksdb store for signature integrations"))
		}
		instance = New(storage, policyDataStore.Singleton())
	})
	return instance
}
