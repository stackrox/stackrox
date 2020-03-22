package m29to30

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	oldContents := map[string]string{
		"provider1\x00group\x00something": "foo",
		"provider1":                       "bar",
		"provider2\x00group\x00with\x00null\x00bytes\x00somethingelse": "baz",
	}
	expectedNewContents := map[string]string{
		"\x09provider1\x05group\x09something":                              "foo",
		"\x09provider1\x00\x00":                                            "bar",
		"\x09provider2\x15group\x00with\x00null\x00bytes\x0dsomethingelse": "baz",
	}

	db, err := bolthelpers.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)

	require.NoError(t, db.Update(func(tx *bolt.Tx) error {
		oldBucket, err := tx.CreateBucket(legacyGroupsBucketName)
		if err != nil {
			return err
		}

		for k, v := range oldContents {
			if err := oldBucket.Put([]byte(k), []byte(v)); err != nil {
				return err
			}
		}
		return nil
	}))

	assert.NoError(t, migration.Run(db, nil))

	newContents := make(map[string]string)
	oldBucketExists := false
	require.NoError(t, db.View(func(tx *bolt.Tx) error {
		oldBucketExists = tx.Bucket(legacyGroupsBucketName) != nil
		newBucket := tx.Bucket(newGroupsBucketName)
		if newBucket == nil {
			return errors.New("new groups bucket not found")
		}
		return newBucket.ForEach(func(k, v []byte) error {
			newContents[string(k)] = string(v)
			return nil
		})
	}))

	assert.False(t, oldBucketExists)
	assert.Equal(t, expectedNewContents, newContents)
}
