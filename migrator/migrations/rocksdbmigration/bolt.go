package rocksdbmigration

import (
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
)

var (
	boltBucketsToMigrate = []string{
		"risk",
		"processWhitelists2",
		"service_accounts",
		"k8sroles",
		"rolebindings",
		"secrets",
		"namespaces",
		"processWhitelistResults",
	}

	separator = []byte("\x00")
)

// MigrateBoltBucket migrates a bolt bucket to RocksDB. Note: the bolt bucket must be flat and cannot
// have nested buckets
func MigrateBoltBucket(boltDB *bbolt.DB, rocksDB *gorocksdb.DB, prefix []byte) (int, error) {
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()

	var count int
	err := boltDB.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(prefix)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			newKey := make([]byte, 0, len(k)+len(prefix)+len(separator))
			newKey = append(newKey, prefix...)
			newKey = append(newKey, separator...)
			newKey = append(newKey, k...)

			newValue := make([]byte, len(v))
			copy(newValue, v)

			count++
			rocksWriteBatch.Put(newKey, newValue)
			return nil
		})
	})
	if err != nil {
		return 0, err
	}
	err = rocksDB.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func migrateBolt(databases *types.Databases) error {
	for _, bucket := range boltBucketsToMigrate {
		count, err := MigrateBoltBucket(databases.BoltDB, databases.RocksDB, []byte(bucket))
		if err != nil {
			return err
		}
		log.WriteToStderrf("Rewrote %d keys from Bolt Bucket %s", count, bucket)
	}
	return nil
}

// GetPrefixedKey returns fully-qualified key for rocksDB
func GetPrefixedKey(prefix []byte, id []byte) []byte {
	newKey := make([]byte, 0, len(id)+len(prefix)+len(separator))
	newKey = append(newKey, prefix...)
	newKey = append(newKey, separator...)
	newKey = append(newKey, id...)
	return newKey
}

// GetIDFromPrefixedKey removes prefix from key and returns it
func GetIDFromPrefixedKey(prefix []byte, key []byte) []byte {
	return key[len(prefix)+len(separator):]
}
