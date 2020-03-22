package m15to16

import (
	"testing"

	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	db, err := bolthelpers.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)
	assert.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(alertBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(alertListBucket); err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(transactionsBucket)
		return err
	}))

	assert.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		alerts := tx.Bucket(alertBucket)
		_ = alerts.Put([]byte("1"), []byte("1"))
		_ = alerts.Put([]byte("2"), []byte("2"))

		listAlerts := tx.Bucket(alertListBucket)
		_ = listAlerts.Put([]byte("1"), []byte("1"))
		_ = listAlerts.Put([]byte("2"), []byte("2"))
		_ = listAlerts.Put([]byte("3"), []byte("3"))
		return nil
	}))

	require.NoError(t, updateListAlerts(db, nil))

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		listAlerts := tx.Bucket(alertListBucket)

		assert.Equal(t, []byte("1"), listAlerts.Get([]byte("1")))
		assert.Equal(t, []byte("2"), listAlerts.Get([]byte("2")))
		assert.Nil(t, listAlerts.Get([]byte("3")))
		return nil
	}))
}
