package m101tom102

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	buckets = []string{"licenseKeys", "telemetry", "transactions"}

	migration = types.Migration{
		StartingSeqNum: 101,
		VersionAfter:   &storage.Version{SeqNum: 102},
		Run: func(databases *types.Databases) error {
			if err := dropBuckets(databases.BoltDB); err != nil {
				return errors.Wrap(err, "error dropping buckets from Bolt")
			}
			return nil
		},
	}
)

func dropBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		for _, bucketName := range buckets {
			if tx.Bucket([]byte(bucketName)) != nil {
				if err := tx.DeleteBucket([]byte(bucketName)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
