package m74tom75

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func getTestDB(t *testing.T) *bolt.DB {
	db := testutils.DBForT(t)
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(policyBucket)
		return err
	})
	require.NoError(t, err)
	return db
}

func TestMigrationWithNoExistingPolicy(t *testing.T) {
	db := getTestDB(t)
	require.NoError(t, migrateSeverityPolicy(db))

	err := db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(policyBucket).Get([]byte("a919ccaf-6b43-4160-ac5d-a405e1440a41"))

		var policy storage.Policy
		if err := policy.Unmarshal(data); err != nil {
			return err
		}
		assert.True(t, policy.GetDisabled())
		return nil
	})
	assert.NoError(t, err)
}

func TestMigrationWithExistingPolicy(t *testing.T) {
	db := getTestDB(t)
	policy := &storage.Policy{
		Id:       "a919ccaf-6b43-4160-ac5d-a405e1440a41",
		Disabled: false,
	}
	err := db.Update(func(tx *bolt.Tx) error {
		data, err := policy.Marshal()
		if err != nil {
			return err
		}
		return tx.Bucket(policyBucket).Put([]byte(policy.GetId()), data)
	})
	require.NoError(t, err)

	require.NoError(t, migrateSeverityPolicy(db))

	err = db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(policyBucket).Get([]byte("a919ccaf-6b43-4160-ac5d-a405e1440a41"))

		var policy storage.Policy
		if err := policy.Unmarshal(data); err != nil {
			return err
		}
		assert.False(t, policy.GetDisabled())
		return nil
	})
	assert.NoError(t, err)
}
