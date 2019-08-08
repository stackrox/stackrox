package globaldb

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	badgerDBInit sync.Once
	badgerDB     *badger.DB

	gcDiscardRatio = 0.7
	gcInterval     = 5 * time.Minute

	registeredBuckets []registeredBucket

	log = logging.LoggerForModule()
)

type registeredBucket struct {
	badgerPrefix []byte
	prefixString string
	objType      string
}

// RegisterBucket registers a bucket to have metrics pulled from it
func RegisterBucket(bucketName []byte, objType string) {
	registeredBuckets = append(registeredBuckets, registeredBucket{
		prefixString: string(bucketName),
		badgerPrefix: badgerhelper.GetBucketKey(bucketName, nil),
		objType:      objType,
	})
}

// GetGlobalBadgerDB returns the global BadgerDB instance.
func GetGlobalBadgerDB() *badger.DB {
	badgerDBInit.Do(func() {
		var err error
		badgerDB, err = badgerhelper.NewWithDefaults()
		if err != nil {
			log.Panicf("Could not initialize badger DB: %v", err)
		}
		go badgerhelper.RunGC(badgerDB, gcDiscardRatio, gcInterval)
		go startMonitoringBadger(badgerDB)
	})
	return badgerDB
}

func startMonitoringBadger(db *badger.DB) {
	ticker := time.NewTicker(gatherFrequency)
	for range ticker.C {
		for _, bucket := range registeredBuckets {
			badgerhelper.UpdateBadgerPrefixSizeMetric(db, bucket.badgerPrefix, bucket.prefixString, bucket.objType)
		}
	}
}
