package test

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

// GetDBWithBucket returns a bolt DB with specified bucket for testing purposes.
func GetDBWithBucket(t *testing.T, bucket []byte) *bolt.DB {
	db := testutils.DBForT(t)
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	require.NoError(t, err)
	return db
}
