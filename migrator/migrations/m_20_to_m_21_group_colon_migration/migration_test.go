package m20to21

import (
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	db, err := bolthelpers.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)

	require.NoError(t, rewrite(db))

	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(groupsBucket)
		require.NoError(t, err)

		require.NoError(t, bucket.Put([]byte("a:b:c"), []byte("1")))
		require.NoError(t, bucket.Put([]byte("d:e:f"), []byte("2")))
		return nil
	})
	require.NoError(t, err)

	require.NoError(t, rewrite(db))

	err = db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(groupsBucket)
		assert.Equal(t, []byte("1"), bucket.Get([]byte("a\x00b\x00c")))
		assert.Equal(t, []byte("2"), bucket.Get([]byte("d\x00e\x00f")))
		return nil
	})
	require.NoError(t, err)
}
