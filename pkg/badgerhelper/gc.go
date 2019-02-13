package badgerhelper

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// RunGC runs the value log garbage collection every gcInterval with a discardRatio, until the DB is closed (determined
// by `RunValueLogGC` returning an `ErrRejected` -- note that this means that value log garbage collection must not
// be triggered in other ways, as this might lead to spurious `ErrRejected` errors).
// The garbage collection is exhausted, meaning value log garbage collection is invoked in a tight loop until no more
// rewrites are performed.
func RunGC(db *badger.DB, discardRatio float64, gcInterval time.Duration) {
	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()

	for range ticker.C {
		var err error
		for err = db.RunValueLogGC(discardRatio); err == nil; err = db.RunValueLogGC(discardRatio) {
		}
		switch err {
		case badger.ErrNoRewrite:
			continue
		case badger.ErrRejected:
			// There should be no concurrent garbage collections, hence assume the DB has been closed.
			return
		default:
			log.Errorf("Error triggering BadgerDB garbage collection: %v", err)
		}
	}
}
