package m54tom55

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

var (
	// Sections are not essential for the test
	// but are required for a policy to be valid.
	sections = []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: "CVSS",
					Values: []*storage.PolicyValue{
						{
							Value: ">= 7.000000",
						},
					},
				},
			},
		},
	}

	exclusions = []*storage.Exclusion{
		{
			Name: "42",
		},
	}

	originalPolicies = []*storage.Policy{
		{
			Id:             "0",
			Name:           "policy 0 with no whitelists",
			PolicyVersion:  oldVersion,
			PolicySections: sections,
		},
		{
			Id:             "1",
			Name:           "policy 1 with a whitelist",
			PolicyVersion:  oldVersion,
			PolicySections: sections,
			Whitelists:     exclusions,
		},
		{
			Id:             "2",
			Name:           "policy 2 with both a whitelist and an exclusion",
			PolicyVersion:  oldVersion,
			PolicySections: sections,
			Whitelists:     exclusions,
			Exclusions:     exclusions,
		},
		{
			Id:             "3",
			Name:           "policy 3 with an exclusion but the old version",
			PolicyVersion:  oldVersion,
			PolicySections: sections,
			Exclusions:     exclusions,
		},
		{
			Id:             "4",
			Name:           "policy 4 with an exclusion and the new version",
			PolicyVersion:  newVersion,
			PolicySections: sections,
			Exclusions:     exclusions,
		},
		{
			Id:             "5",
			Name:           "policy 5 with no exclusion and and the new version",
			PolicyVersion:  newVersion,
			PolicySections: sections,
		},
		{
			Id:             "6",
			Name:           "policy 6 with a whitelist and the new version",
			PolicyVersion:  newVersion,
			PolicySections: sections,
			Whitelists:     exclusions,
		},
	}

	expectedPolicies = []*storage.Policy{
		{
			Id:             "0",
			Name:           "policy 0 with no whitelists",
			PolicyVersion:  newVersion,
			PolicySections: sections,
		},
		{
			Id:             "1",
			Name:           "policy 1 with a whitelist",
			PolicyVersion:  newVersion,
			PolicySections: sections,
			Exclusions:     exclusions,
		},
		{
			Id:             "2",
			Name:           "policy 2 with both a whitelist and an exclusion",
			PolicyVersion:  newVersion,
			PolicySections: sections,
			Exclusions:     append(exclusions, exclusions...),
		},
		{
			Id:             "3",
			Name:           "policy 3 with an exclusion but the old version",
			PolicyVersion:  newVersion,
			PolicySections: sections,
			Exclusions:     exclusions,
		},
		{
			Id:             "4",
			Name:           "policy 4 with an exclusion and the new version",
			PolicyVersion:  newVersion,
			PolicySections: sections,
			Exclusions:     exclusions,
		},
		{
			Id:             "5",
			Name:           "policy 5 with no exclusion and and the new version",
			PolicyVersion:  newVersion,
			PolicySections: sections,
		},
		{
			Id:             "6",
			Name:           "policy 6 with a whitelist and the new version",
			PolicyVersion:  newVersion,
			PolicySections: sections,
			Exclusions:     exclusions,
		},
	}
)

func TestPolicyMigration(t *testing.T) {
	db := testutils.DBForT(t)

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucket(policyBucket)
		if err != nil {
			return err
		}

		for _, policy := range originalPolicies {
			bytes, err := proto.Marshal(policy)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(policy.GetId()), bytes); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err, "Prepare test policy bucket")

	err = migrateWhitelistsToExclusions(db)
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

	assert.ElementsMatch(t, expectedPolicies, migratedPolicies)
}
