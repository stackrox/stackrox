package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
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
	if features.PostgresDatastore.Enabled() {
		ad = New(postgres.New(globaldb.GetPostgres()))
	} else {
		ad = New(rocksdb.New(globaldb.GetRocksDB()))
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
