package globaldb

import (
	"sync"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	once sync.Once

	globalDB *bolt.DB
)

func initialize() {
	var err error
	globalDB, err = bolthelper.NewWithDefaults()
	if err != nil {
		panic(err)
	}
	go startMonitoring(globalDB)
}

// GetGlobalDB returns a pointer to the global db.
func GetGlobalDB() *bolt.DB {
	once.Do(initialize)
	return globalDB
}

// Close closes the global db. Should only be used at central shutdown time.
func Close() {
	once.Do(initialize)
	if err := globalDB.Close(); err != nil {
		logger.Errorf("unable to close bolt db: %s", err)
	}
}
