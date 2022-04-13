package m82tom83

import (
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/common/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

var (
	latestTagPolicyJSON = `{
  "id": "2e90874a-3521-44de-85c6-5720f519a701",
  "name": "Latest tag",
  "description": "Alert on deployments with images using tag 'latest'",
  "rationale": "Using latest tag can result in running heterogeneous versions of code. Many Docker hosts cache the Docker images, which means newer versions of the latest tag will not be picked up. See https://docs.docker.com/develop/dev-best-practices for more best practices.",
  "remediation": "Consider moving to semantic versioning based on code releases (semver.org) or using the first 12 characters of the source control SHA. This will allow you to tie the Docker image to the code.",
  "categories": [
    "DevOps Best Practices"
  ],
  "lifecycleStages": [
    "BUILD",
    "DEPLOY"
  ],
  "exclusions": [
    {
      "name": "Don't alert on kube-system namespace",
      "deployment": {
        "scope": {
          "namespace": "kube-system"
        }
      }
    },
    {
      "name": "Don't alert on istio-system namespace",
      "deployment": {
        "scope": {
          "namespace": "istio-system"
        }
      }
    }
  ],
  "severity": "LOW_SEVERITY",
  "policyVersion": "1.1",
  "policySections": [
    {
      "policyGroups": [
        {
          "fieldName": "Image Tag",
          "values": [
            {
              "value": "latest"
            }
          ]
        }
      ]
    }
  ],
  "criteriaLocked": true,
  "mitreVectorsLocked": true
}`

	customPolicyJSON = `{
  "id": "2e90874a-3521-44de-85c6-5720f519a111",
  "name": "custom policy",
  "categories": [
    "DevOps Best Practices"
  ],
  "lifecycleStages": [
    "BUILD",
    "DEPLOY"
  ],
  "severity": "LOW_SEVERITY",
  "policyVersion": "1.1",
  "policySections": [
    {
      "policyGroups": [
        {
          "fieldName": "Image Tag",
          "values": [
            {
              "value": "latest"
            }
          ]
        }
      ]
    }
  ],
  "criteriaLocked": true,
  "mitreVectorsLocked": true
}`
)

func TestDefaultPolicyIsMigrated(t *testing.T) {
	var policy storage.Policy
	err := jsonpb.Unmarshal(strings.NewReader(latestTagPolicyJSON), &policy)
	require.NoError(t, err)

	runTest(t, &policy, true)
}

func TestCustomPolicyIsNotMigrated(t *testing.T) {
	var policy storage.Policy
	err := jsonpb.Unmarshal(strings.NewReader(customPolicyJSON), &policy)
	require.NoError(t, err)

	runTest(t, &policy, false)
}

func TestEditedDefaultPolicyIsNotMigrated(t *testing.T) {
	var policy storage.Policy
	err := jsonpb.Unmarshal(strings.NewReader(latestTagPolicyJSON), &policy)
	require.NoError(t, err)

	// Edit policy section.
	policy.PolicySections = nil

	runTest(t, &policy, false)
}

func TestDefaultPolicyWithoutLockedCriteriaIsNotMigrated(t *testing.T) {
	var policy storage.Policy
	err := jsonpb.Unmarshal(strings.NewReader(latestTagPolicyJSON), &policy)
	require.NoError(t, err)

	// Edit policy section.
	policy.CriteriaLocked = false

	runTest(t, &policy, false)
}

func runTest(t *testing.T, policy *storage.Policy, mustBeDefault bool) {
	db := test.GetDBWithBucket(t, policyBucket)

	// Add a default policy to DB.
	err := db.Update(func(tx *bolt.Tx) error {
		data, err := policy.Marshal()
		if err != nil {
			return err
		}
		return tx.Bucket(policyBucket).Put([]byte(policy.GetId()), data)
	})
	require.NoError(t, err)

	// Run migration.
	require.NoError(t, updatePoliciesWithDefaultFlag(db))

	// Verify default policy was updated.
	err = db.View(func(tx *bolt.Tx) error {
		val := tx.Bucket(policyBucket).Get([]byte(policy.GetId()))
		assert.NotNil(t, val)

		var storedPolicy storage.Policy
		if err := proto.Unmarshal(val, &storedPolicy); err != nil {
			return errors.Wrapf(err, "unmarshaling policy %s", policy.GetId())
		}
		assert.Equal(t, mustBeDefault, storedPolicy.GetIsDefault())

		return nil
	})
	assert.NoError(t, err)
}
