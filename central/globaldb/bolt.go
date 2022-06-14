package globaldb

import (
	"github.com/stackrox/stackrox/central/option"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/postgres"
	"github.com/stackrox/stackrox/pkg/sync"
	bolt "go.etcd.io/bbolt"
)

var (
	once sync.Once

	globalDB *bolt.DB
)

func initialize() {
	var err error
	globalDB, err = bolthelper.NewWithDefaults(option.CentralOptions.DBPathBase)
	if err != nil {
		panic(err)
	}
	go startMonitoring(globalDB)
}

// GetGlobalDB returns a pointer to the global db.
func GetGlobalDB() *bolt.DB {
	postgres.LogCallerOnPostgres("GetGlobalDB")
	once.Do(initialize)
	return globalDB
}
