package globaldb

import (
	"github.com/stackrox/rox/central/option"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sync"
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
