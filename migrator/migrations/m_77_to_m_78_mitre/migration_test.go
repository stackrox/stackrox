package m77tom78

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	db := test.GetDBWithBucket(t, policyBucket)

	// Add default policies to DB with no mitre att&ck.
	policies, err := defaultPolicies()
	require.NoError(t, err)
	for _, slimPolicy := range policies {
		err := db.Update(func(tx *bolt.Tx) error {
			policy := &storage.Policy{
				Id:   slimPolicy.ID,
				Name: slimPolicy.Name,
			}
			data, err := policy.Marshal()
			if err != nil {
				return err
			}
			return tx.Bucket(policyBucket).Put([]byte(slimPolicy.ID), data)
		})
		require.NoError(t, err)
	}

	// Verify default policies do not have mitre att&ck.
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		require.NotNil(t, bucket)
		return bucket.ForEach(func(k, v []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(v, policy); err != nil {
				return errors.Wrapf(err, "unmarshaling policy %s", k)
			}
			assert.Nil(t, policy.GetMitreAttackVectors())
			assert.False(t, policy.GetMitreVectorsLocked())
			return nil
		})
	})
	require.NoError(t, err)

	// Run migration
	require.NoError(t, updatePoliciesWithMitre(db))

	// Verify default policies have mitre att&ck.
	err = db.View(func(tx *bolt.Tx) error {
		for _, policy := range policies {
			val := tx.Bucket(policyBucket).Get([]byte(policy.ID))
			require.NotNil(t, val)

			var storedPolicy storage.Policy
			if err := proto.Unmarshal(val, &storedPolicy); err != nil {
				return errors.Wrapf(err, "unmarshaling policy %s", policy.ID)
			}
			assert.ElementsMatch(t, policy.MitreAttackVectors, storedPolicy.GetMitreAttackVectors())
			assert.True(t, storedPolicy.GetMitreVectorsLocked())
		}
		return nil
	})
	assert.NoError(t, err)
}
