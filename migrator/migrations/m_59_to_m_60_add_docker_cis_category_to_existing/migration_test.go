package m59tom60

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

var (
	someOtherCategory = "some other category"
)

func TestPolicyMigration(t *testing.T) {
	db := testutils.DBForT(t)

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucket(policyBucket)
		if err != nil {
			return err
		}

		for i, id := range policiesToUpdate {
			testPolicy := &storage.Policy{
				Id:         id,
				Categories: []string{someOtherCategory},
			}
			if i == 0 {
				testPolicy.Categories = append(testPolicy.Categories, dockerCIS)
			}
			bytes, err := proto.Marshal(testPolicy)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(testPolicy.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	})

	require.NoError(t, err, "Prepare test policy bucket")

	err = migrateNewPolicyCategories(db)
	require.NoError(t, err, "Run migration")

	var migratedPolicies []*storage.Policy
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucket)
		}
		return bucket.ForEach(func(_, obj []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(obj, policy); err != nil {
				return err
			}
			migratedPolicies = append(migratedPolicies, policy)
			return nil
		})
	})
	require.NoError(t, err, "Read migrated policies from the bucket")

	require.Len(t, migratedPolicies, len(policiesToUpdate))
	for _, policy := range migratedPolicies {
		require.Len(t, policy.GetCategories(), 2)
		require.Contains(t, policy.GetCategories(), dockerCIS)
		require.Contains(t, policy.GetCategories(), someOtherCategory)
	}
}
