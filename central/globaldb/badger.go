package globaldb

import (
	"sync"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	badgerDBInit sync.Once
	badgerDB     *badger.DB

	gcDiscardRatio = 0.7
	gcInterval     = 5 * time.Minute

	log = logging.LoggerForModule()
)

// GetGlobalBadgerDB returns the global BadgerDB instance.
func GetGlobalBadgerDB() *badger.DB {
	badgerDBInit.Do(func() {
		var err error
		badgerDB, err = badgerhelper.NewWithDefaults()
		if err != nil {
			log.Panicf("Could not initialize badger DB: %v", err)
		}
		go badgerhelper.RunGC(badgerDB, gcDiscardRatio, gcInterval)
	})
	return badgerDB
}
