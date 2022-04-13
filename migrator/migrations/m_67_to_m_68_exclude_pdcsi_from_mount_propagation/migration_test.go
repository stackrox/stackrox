package m67tom68

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

var (
	policyBucketName  = []byte("policies")
	expectedPolicyDir = "test_expected_policies"
)

func TestPolicyMigration(t *testing.T) {
	db := testutils.DBForT(t)

	beforeMigrationPolicies, err := getComparisonPoliciesFromFiles()
	require.NoError(t, err, "Loading original policies")

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucket(policyBucketName)
		if err != nil {
			return err
		}

		for policyID := range policiesToMigrate {
			policy, ok := beforeMigrationPolicies[policyID]
			require.True(t, ok)

			policyBytes, err := proto.Marshal(policy)
			if err != nil {
				return err
			}
			if err = bucket.Put([]byte(policyID), policyBytes); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err, "Prepare test policy bucket")

	err = updatePolicies(db)
	require.NoError(t, err, "Run migration")

	migratedPolicies := make(map[string]*storage.Policy)
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucketName)
		if bucket == nil {
			return errors.Errorf("bucket %q not found", policyBucketName)
		}
		return bucket.ForEach(func(_, obj []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(obj, policy); err != nil {
				return err
			}
			migratedPolicies[policy.GetId()] = policy
			return nil
		})
	})
	require.NoError(t, err, "Read migrated policies from the bucket")

	expectedPolicies := make(map[string]*storage.Policy)
	for policyID := range policiesToMigrate {
		policyPath := filepath.Join(expectedPolicyDir, fmt.Sprintf("%s.json", policyID))
		contents, err := os.ReadFile(policyPath)
		require.NoError(t, err)

		policy := &storage.Policy{}
		err = jsonpb.Unmarshal(bytes.NewReader(contents), policy)
		require.NoError(t, err)

		expectedPolicies[policy.Id] = policy
	}

	for policyID := range policiesToMigrate {
		expectedPolicy, ok := expectedPolicies[policyID]
		require.True(t, ok)

		migratedPolicy, ok := migratedPolicies[policyID]
		require.True(t, ok)

		require.EqualValues(t, expectedPolicy, migratedPolicy, "Policy migrated")
	}
}
