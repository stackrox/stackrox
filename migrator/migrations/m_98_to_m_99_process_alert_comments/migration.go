package m98to99

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	bolt "go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 98,
		VersionAfter:   &storage.Version{SeqNum: 99},
		Run: func(databases *types.Databases) error {
			err := deleteProcessAndAlertCommentsBuckets(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "updating policies")
			}
			return nil
		},
	}

	alertCommentsBucket   = []byte("alertComments")
	processCommentsBucket = []byte("process_comments")
)

func deleteProcessAndAlertCommentsBuckets(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket(alertCommentsBucket) != nil {
			if err := tx.DeleteBucket(alertCommentsBucket); err != nil {
				return errors.Wrap(err, "failed to delete alert comments bucket")
			}
		}
		if tx.Bucket(processCommentsBucket) != nil {
			if err := tx.DeleteBucket(processCommentsBucket); err != nil {
				return errors.Wrap(err, "failed to delete process comments bucket")
			}
		}
		return nil
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
