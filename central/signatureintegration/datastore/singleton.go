package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/signatureintegration/store/rocksdb"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
)

func initialize() {
	storage, err := rocksdb.New(globaldb.GetRocksDB())
	utils.CrashOnError(errors.Wrap(err, "unable to create rocksdb store for signature integrations"))
	instance = New(storage)
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(initialize)
	return instance
}
