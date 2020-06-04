package m37tom38

import (
	"fmt"
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	unmigratedPolicies = []*storage.Policy{
		{
			Id:   "0",
			Name: "policy0",
			Fields: &storage.PolicyFields{
				ImageName: &storage.ImageNamePolicy{
					Registry: "123",
					Remote:   "456",
					Tag:      "789",
				},
				RequiredImageLabel: &storage.KeyValuePolicy{
					Key:   "abc",
					Value: "def",
				},
			},
		},
		{
			Id:   "1",
			Name: "policy1",
			Fields: &storage.PolicyFields{
				FixedBy: "hjkdf",
				Cvss: &storage.NumericalPolicy{
					Op:    storage.Comparator_EQUALS,
					Value: 0,
				},
			},
		},
		{
			Id:            "unmigratablePolicy",
			Name:          "unmigratablePolicy",
			PolicyVersion: "abcd",
		},
	}

	unmigratedPoliciesAfterMigration = []*storage.Policy{
		{
			Id:            "0",
			Name:          "policy0",
			PolicyVersion: version,
			Fields: &storage.PolicyFields{
				ImageName: &storage.ImageNamePolicy{
					Registry: "123",
					Remote:   "456",
					Tag:      "789",
				},
				RequiredImageLabel: &storage.KeyValuePolicy{
					Key:   "abc",
					Value: "def",
				},
			},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: imageRegistry,
							Values: []*storage.PolicyValue{
								{
									Value: "123",
								},
							},
						},
						{
							FieldName: imageRemote,
							Values: []*storage.PolicyValue{
								{
									Value: "r/.*456.*",
								},
							},
						},
						{
							FieldName: imageTag,
							Values: []*storage.PolicyValue{
								{
									Value: "789",
								},
							},
						},
						{
							FieldName: requiredImageLabel,
							Values: []*storage.PolicyValue{
								{
									Value: "abc=def",
								},
							},
						},
					},
				},
			},
		},
		{
			Id:            "1",
			Name:          "policy1",
			PolicyVersion: version,
			Fields: &storage.PolicyFields{
				FixedBy: "hjkdf",
				Cvss: &storage.NumericalPolicy{
					Op:    storage.Comparator_EQUALS,
					Value: 0,
				},
			},
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: fixedBy,
							Values: []*storage.PolicyValue{
								{
									Value: "hjkdf",
								},
							},
						},
						{
							FieldName: cvss,
							Values: []*storage.PolicyValue{
								{
									Value: "0.000000",
								},
							},
						},
					},
				},
			},
		},
	}

	alreadyMigratedPolicies = []*storage.Policy{
		{
			Id:            "2",
			Name:          "policy2",
			PolicyVersion: version,
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: containerCPULimit,
							Values: []*storage.PolicyValue{
								{
									Value: ">= 22",
								},
							},
						},
					},
				},
			},
		},
		{
			Id:            "3",
			Name:          "policy3",
			PolicyVersion: version,
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName: processName,
							Values: []*storage.PolicyValue{
								{
									Value: "poiuytrewq",
								},
							},
						},
					},
				},
			},
		},
	}
)

func TestPolicyMigration(t *testing.T) {
	db := testutils.DBForT(t)

	var policiesToUpsert []*storage.Policy
	policiesToUpsert = append(policiesToUpsert, unmigratedPolicies...)
	policiesToUpsert = append(policiesToUpsert, alreadyMigratedPolicies...)

	require.NoError(t, db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket(policyBucket)
		if err != nil {
			return err
		}
		uBucket, err := tx.CreateBucket(uniqueBucket)
		if err != nil {
			return err
		}
		mBucket, err := tx.CreateBucket(mapperBucket)
		if err != nil {
			return err
		}

		for _, policy := range policiesToUpsert {
			bytes, err := proto.Marshal(policy)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(policy.GetId()), bytes); err != nil {
				return err
			}
			if err := uBucket.Put([]byte(policy.GetName()), []byte("")); err != nil {
				return err
			}
			if err := mBucket.Put([]byte(policy.GetId()), []byte(policy.GetName())); err != nil {
				return err
			}
		}
		return nil
	}))

	require.NoError(t, migrateLegacyPoliciesToBPL(db))

	var allPoliciesAfterMigration []*storage.Policy
	var namesAfterMigration []string
	var idsAfterMigration []string
	var unmigratablePoliciesAfterMigration []*storage.Policy
	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(policyBucket)
		if bucket == nil {
			return fmt.Errorf("bucket %s does not exist", string(policyBucket))
		}
		err := bucket.ForEach(func(k, v []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(v, policy); err != nil {
				return err
			}
			allPoliciesAfterMigration = append(allPoliciesAfterMigration, policy)
			return nil
		})
		if err != nil {
			return err
		}

		uBucket := tx.Bucket(uniqueBucket)
		if uBucket == nil {
			return fmt.Errorf("bucket %s does not exist", string(uniqueBucket))
		}
		err = uBucket.ForEach(func(k, v []byte) error {
			namesAfterMigration = append(namesAfterMigration, string(k))
			return nil
		})
		if err != nil {
			return err
		}

		mBucket := tx.Bucket(mapperBucket)
		if mBucket == nil {
			return fmt.Errorf("bucket %s does not exist", string(policyBucket))
		}
		err = mBucket.ForEach(func(k, v []byte) error {
			idsAfterMigration = append(idsAfterMigration, string(k))
			return nil
		})
		if err != nil {
			return err
		}

		unmigratableBucket := tx.Bucket(unmigratableBucketName)
		if unmigratableBucket == nil {
			return fmt.Errorf("bucket %s does not exist", string(unmigratableBucketName))
		}
		return unmigratableBucket.ForEach(func(k, v []byte) error {
			policy := &storage.Policy{}
			if err := proto.Unmarshal(v, policy); err != nil {
				return err
			}
			unmigratablePoliciesAfterMigration = append(unmigratablePoliciesAfterMigration, policy)
			return nil
		})
	}))

	// All valid policies were migrated
	var expectedPoliciesAfterMigration []*storage.Policy
	expectedPoliciesAfterMigration = append(expectedPoliciesAfterMigration, unmigratedPoliciesAfterMigration...)
	expectedPoliciesAfterMigration = append(expectedPoliciesAfterMigration, alreadyMigratedPolicies...)
	assert.ElementsMatch(t, expectedPoliciesAfterMigration, allPoliciesAfterMigration)

	// Unmigratable policies were put in a separate bucket
	expectedUnmigratable := unmigratedPolicies[2:]
	assert.ElementsMatch(t, expectedUnmigratable, unmigratablePoliciesAfterMigration)

	// Name/ID of the unmigratable policy was removed from the name/ID cross reference and no other cross references
	// were removed.
	expectedNamesAfterMigration := make([]string, 0, len(expectedPoliciesAfterMigration))
	expectedIDsAfterMigration := make([]string, 0, len(expectedPoliciesAfterMigration))
	for _, policy := range expectedPoliciesAfterMigration {
		expectedNamesAfterMigration = append(expectedNamesAfterMigration, policy.GetName())
		expectedIDsAfterMigration = append(expectedIDsAfterMigration, policy.GetId())
	}
	assert.ElementsMatch(t, expectedNamesAfterMigration, namesAfterMigration)
	assert.ElementsMatch(t, expectedIDsAfterMigration, idsAfterMigration)
}
