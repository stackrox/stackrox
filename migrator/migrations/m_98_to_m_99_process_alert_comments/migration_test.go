package m98to99

import (
	"testing"

	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func assertBucketExistence(t *testing.T, db *bolt.DB, alertCommentsShouldExist, processCommentsShouldExist bool) {
	require.NoError(t, db.View(func(tx *bolt.Tx) error {
		assert.Equal(t, alertCommentsShouldExist, tx.Bucket(alertCommentsBucket) != nil)
		assert.Equal(t, processCommentsShouldExist, tx.Bucket(processCommentsBucket) != nil)
		return nil
	}))
}
func TestMigration(t *testing.T) {
	db := testutils.DBForT(t)
	defer testutils.TearDownDB(db)

	assertBucketExistence(t, db, false, false)
	require.NoError(t, deleteProcessAndAlertCommentsBuckets(db))
	assertBucketExistence(t, db, false, false)

	require.NoError(t, db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket(alertCommentsBucket)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucket(processCommentsBucket)
		if err != nil {
			return err
		}

		return nil
	}))
	assertBucketExistence(t, db, true, true)
	require.NoError(t, deleteProcessAndAlertCommentsBuckets(db))
	assertBucketExistence(t, db, false, false)

	require.NoError(t, db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket(alertCommentsBucket)
		if err != nil {
			return err
		}
		return nil
	}))

	assertBucketExistence(t, db, true, false)
	require.NoError(t, deleteProcessAndAlertCommentsBuckets(db))
	assertBucketExistence(t, db, false, false)
}
