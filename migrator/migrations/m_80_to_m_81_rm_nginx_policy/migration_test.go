package m80tom81

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
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

func TestMigrationWithoutEditedPolicy(t *testing.T) {
	db := getTestDB(t)

	policy := &storage.Policy{}
	err := jsonpb.Unmarshal(strings.NewReader(nginxPolicyJSON), policy)
	require.NoError(t, err)

	err = db.Update(func(tx *bolt.Tx) error {
		data, err := policy.Marshal()
		if err != nil {
			return err
		}
		return tx.Bucket(policyBucket).Put([]byte(nginxPolicyID), data)
	})
	require.NoError(t, err)

	// Verify default policies exists.
	err = db.View(func(tx *bolt.Tx) error {
		require.NotNil(t, tx.Bucket(policyBucket).Get([]byte(nginxPolicyID)))
		return nil
	})
	require.NoError(t, err)

	// Run migration
	assert.NoError(t, rmNginxPolicy(db))

	// Verify default policies is removed.
	err = db.View(func(tx *bolt.Tx) error {
		assert.Nil(t, tx.Bucket(policyBucket).Get([]byte(nginxPolicyID)))
		return nil
	})
	assert.NoError(t, err)
}

func TestMigrationWithEditedPolicy(t *testing.T) {
	db := getTestDB(t)

	err := db.Update(func(tx *bolt.Tx) error {
		policy := &storage.Policy{
			Id:   nginxPolicyID,
			Name: "DockerHub NGINX 1.10",
		}
		data, err := policy.Marshal()
		if err != nil {
			return err
		}
		return tx.Bucket(policyBucket).Put([]byte(nginxPolicyID), data)
	})
	require.NoError(t, err)

	// Verify default policies exists.
	err = db.View(func(tx *bolt.Tx) error {
		require.NotNil(t, tx.Bucket(policyBucket).Get([]byte(nginxPolicyID)))
		return nil
	})
	require.NoError(t, err)

	// Run migration
	assert.NoError(t, rmNginxPolicy(db))

	// Verify default policies is not removed.
	err = db.View(func(tx *bolt.Tx) error {
		assert.NotNil(t, tx.Bucket(policyBucket).Get([]byte(nginxPolicyID)))
		return nil
	})
	assert.NoError(t, err)
}
