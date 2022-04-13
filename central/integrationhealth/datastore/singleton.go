package datastore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/integrationhealth/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	ad = New(rocksdb.New(globaldb.GetRocksDB()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
