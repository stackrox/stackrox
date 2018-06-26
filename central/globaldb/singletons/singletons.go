package singletons

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/boltdb/bolt"
)

var (
	once sync.Once

	log = logging.LoggerForModule()

	globalDB *bolt.DB
)

func initialize() {
	var err error
	globalDB, err = bolthelper.NewWithDefaults(env.DBPath.Setting())
	if err != nil {
		panic(err)
	}
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
