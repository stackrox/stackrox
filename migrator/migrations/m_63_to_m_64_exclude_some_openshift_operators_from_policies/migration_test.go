package m63tom64

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/bolthelpers"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestUpdatePoliciesWithOSExclusionsMigration(t *testing.T) {
	suite.Run(t, new(excludeOpenShiftNamespacesFromPoliciesTestSuite))
}

type excludeOpenShiftNamespacesFromPoliciesTestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *excludeOpenShiftNamespacesFromPoliciesTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(policyBucketName)
		return err
	}))
	suite.db = db
}

func (suite *excludeOpenShiftNamespacesFromPoliciesTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertPolicy(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func addFakeExcludes(numOfExcludes int, policy *storage.Policy) {
	for i := 0; i < numOfExcludes; i++ {
		policy.Exclusions = append(policy.Exclusions, &storage.Exclusion{
			Name:       fmt.Sprintf("Existing exclusion %d", i),
			Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: fmt.Sprintf("test-namespace-%d", i)}},
		})
	}
}

func checkModifiedCriteriaIsNotUpdated(suite *excludeOpenShiftNamespacesFromPoliciesTestSuite, bucket bolthelpers.BucketRef, testPolicyID string, modifiedPolicySection []*storage.PolicySection) {
	policy := &storage.Policy{
		Id:             testPolicyID,
		PolicySections: modifiedPolicySection,
	}
	addFakeExcludes(policiesToMigrate[testPolicyID].existingNumExclusions, policy)

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePoliciesWithOSExclusions(suite.db))

	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)
}

// Test that unrelated policies aren't migrated
func (suite *excludeOpenShiftNamespacesFromPoliciesTestSuite) TestUnrelatedPolicyAreNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id: "this-is-a-random-id-that-should-not-exist",
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePoliciesWithOSExclusions(suite.db))

	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)
}

// Test that all unmodified policies are migrated
func (suite *excludeOpenShiftNamespacesFromPoliciesTestSuite) TestUnmodifiedPoliciesGetMigrated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	var policiesToTest []*storage.Policy
	for policyID, updateDetails := range policiesToMigrate {
		policy := &storage.Policy{
			Id: policyID,
			PolicySections: []*storage.PolicySection{
				{
					SectionName:  "",
					PolicyGroups: updateDetails.existingPolicyGroups,
				},
			},
		}
		addFakeExcludes(updateDetails.existingNumExclusions, policy) // Add some fake exclusions that match the number of existing ones it had

		suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
		policiesToTest = append(policiesToTest, policy)
	}

	suite.NoError(updatePoliciesWithOSExclusions(suite.db))

	for _, policy := range policiesToTest {
		var newPolicy storage.Policy
		suite.NoError(bucket.View(func(b *bolt.Bucket) error {
			v := b.Get([]byte(policy.GetId()))
			return proto.Unmarshal(v, &newPolicy)
		}))
		policy.Exclusions = append(policy.Exclusions, policiesToMigrate[policy.GetId()].newExclusions...)
		suite.EqualValues(policy, &newPolicy)
	}
}

// Test a mix of policies, some which are unmodified, some modified and some missing. Only existing and unmodified should be updated
func (suite *excludeOpenShiftNamespacesFromPoliciesTestSuite) TestMixOfMissingAndUpdatedPolicies() {
	updatedPolicyID := "880fd131-46f0-43d2-82c9-547f5aa7e043" // this policy will be modified and so shouldn't have exclusions
	missingPolicyID := "32d770b9-c6ba-4398-b48a-0c3e807644ed" // This policy won't be added to DB

	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	var policiesToTest []*storage.Policy
	for policyID, updateDetails := range policiesToMigrate {
		if policyID == missingPolicyID {
			continue
		}

		policy := &storage.Policy{
			Id: policyID,
			PolicySections: []*storage.PolicySection{
				{
					SectionName:  "",
					PolicyGroups: updateDetails.existingPolicyGroups,
				},
			},
		}
		addFakeExcludes(updateDetails.existingNumExclusions, policy) // Add some fake exclusions that match the number of existing ones it had

		if policyID == updatedPolicyID {
			policy.PolicySections = []*storage.PolicySection{
				{
					SectionName:  "blah",
					PolicyGroups: []*storage.PolicyGroup{{FieldName: "something", Values: []*storage.PolicyValue{{Value: "unreal"}}}},
				},
			}
		}

		suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
		policiesToTest = append(policiesToTest, policy)
	}

	suite.NoError(updatePoliciesWithOSExclusions(suite.db))

	for _, policy := range policiesToTest {
		var newPolicy storage.Policy
		suite.NoError(bucket.View(func(b *bolt.Bucket) error {
			v := b.Get([]byte(policy.GetId()))
			return proto.Unmarshal(v, &newPolicy)
		}))
		if policy.GetId() != updatedPolicyID {
			// The modified policy shouldn't have any of the new exclusions
			policy.Exclusions = append(policy.Exclusions, policiesToMigrate[policy.GetId()].newExclusions...)
		}
		suite.EqualValues(policy, &newPolicy)
	}
}

// Test that policies with extra exclusions don't get migrated
func (suite *excludeOpenShiftNamespacesFromPoliciesTestSuite) TestPolicyWithExtraExclusionsAreNotUpdated() {
	// The following tests are all run on just one policy to make it simple
	testPolicyID := "880fd131-46f0-43d2-82c9-547f5aa7e043"

	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id: testPolicyID,
		PolicySections: []*storage.PolicySection{
			{
				SectionName:  "",
				PolicyGroups: policiesToMigrate[testPolicyID].existingPolicyGroups,
			},
		},
	}
	addFakeExcludes(policiesToMigrate[testPolicyID].existingNumExclusions+1, policy)

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePoliciesWithOSExclusions(suite.db))

	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)
}

func (suite *excludeOpenShiftNamespacesFromPoliciesTestSuite) TestModifiedPoliciesAreNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	// The following tests are all run on just one policy to make it simple
	testPolicyID := "880fd131-46f0-43d2-82c9-547f5aa7e043"

	//// Test that a policy that matches id, field name _but not_ criteria is not updated
	checkModifiedCriteriaIsNotUpdated(suite, bucket, testPolicyID, []*storage.PolicySection{
		{
			SectionName: "",
			PolicyGroups: []*storage.PolicyGroup{
				{FieldName: policiesToMigrate[testPolicyID].existingPolicyGroups[0].FieldName, BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "fake"}}},
			},
		},
	})

	// Test that a policy that matches id, but has additional policy groups is not updated
	fakePolicyGroups := policiesToMigrate[testPolicyID].existingPolicyGroups
	fakePolicyGroups = append(fakePolicyGroups, &storage.PolicyGroup{FieldName: "someField", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "someValue"}}})
	checkModifiedCriteriaIsNotUpdated(suite, bucket, testPolicyID, []*storage.PolicySection{
		{
			SectionName:  "",
			PolicyGroups: fakePolicyGroups,
		},
	})

	// Test that a policy that matches id, but has multiple policy sections is not updated
	fakePolicySections := []*storage.PolicySection{
		{
			SectionName:  "",
			PolicyGroups: fakePolicyGroups,
		},
		{
			SectionName:  "section 2",
			PolicyGroups: []*storage.PolicyGroup{{FieldName: "Image OS", Values: []*storage.PolicyValue{{Value: "ubuntu:19.04"}}}},
		},
	}
	checkModifiedCriteriaIsNotUpdated(suite, bucket, testPolicyID, fakePolicySections)
}
