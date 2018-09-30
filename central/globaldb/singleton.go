package globaldb

import (
	"sync"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/env"
)

var (
	once sync.Once

	globalDB *bolt.DB
)

func initialize() {
	var err error
	globalDB, err = bolthelper.NewWithDefaults(env.DBPath.Setting())
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
		log.Errorf("unable to close bolt db: %s", err)
	}
}
