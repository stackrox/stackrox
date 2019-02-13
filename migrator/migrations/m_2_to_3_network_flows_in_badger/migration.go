package m2to3

import (
	"fmt"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	boltBucket      = []byte("clustersWithFlowsBucket")
	badgerKeyPrefix = []byte("networkFlows")
)

type flowEntry struct {
	bucket, key, value []byte
}

func readFromBolt(db *bolt.DB, entryC chan<- flowEntry, badgerErrC <-chan error) error {
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(boltBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(bucketKey, v []byte) error {
			if v != nil {
				return nil
			}
			clusterBucket := bucket.Bucket(bucketKey)
			if clusterBucket == nil {
				return nil
			}
			return clusterBucket.ForEach(func(valueKey, v []byte) error {
				if v == nil {
					return nil
				}
				select {
				case entryC <- flowEntry{bucket: bucketKey, key: valueKey, value: v}:
				case err := <-badgerErrC:
					return fmt.Errorf("badger write goroutine reported error: %v", err)
				}
				return nil
			})
		})
	})
	close(entryC)
	return err
}

func writeToBadgerAsync(badgerDB *badger.DB, kvC <-chan flowEntry, errC chan<- error) {
	err := badgerDB.Update(func(txn *badger.Txn) error {
		for kv := range kvC {
			keyStr := fmt.Sprintf("%s\x00%s\x00%s", string(badgerKeyPrefix), string(kv.bucket), string(kv.key))
			if err := txn.Set([]byte(keyStr), kv.value); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		errC <- err
	}
	close(errC)
}

func deleteFromBolt(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(boltBucket)
	})
}

func migrate(boltDB *bolt.DB, badgerDB *badger.DB) error {
	badgerErrC := make(chan error, 1)
	kvC := make(chan flowEntry)

	go writeToBadgerAsync(badgerDB, kvC, badgerErrC)

	if err := readFromBolt(boltDB, kvC, badgerErrC); err != nil {
		return fmt.Errorf("reading from bolt: %v", err)
	}
	if err := <-badgerErrC; err != nil {
		return fmt.Errorf("writing to badger: %v", err)
	}

	if err := deleteFromBolt(boltDB); err != nil {
		return fmt.Errorf("deleting from bolt: %v", err)
	}
	return nil
}

var (
	networkFlowsMigration = types.Migration{
		StartingSeqNum: 2,
		VersionAfter:   storage.Version{SeqNum: 3},
		Run:            migrate,
	}
)

func init() {
	migrations.MustRegisterMigration(networkFlowsMigration)
}
