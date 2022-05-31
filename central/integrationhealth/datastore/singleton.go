package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/integrationhealth/store"
	"github.com/stackrox/rox/central/integrationhealth/store/postgres"
	"github.com/stackrox/rox/central/integrationhealth/store/rocksdb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var storage store.Store
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(context.TODO(), globaldb.GetPostgres())
	} else {
		storage = rocksdb.New(globaldb.GetRocksDB())
	}
	ad = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
