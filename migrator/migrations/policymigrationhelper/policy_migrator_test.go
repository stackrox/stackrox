package policymigrationhelper

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestPolicyMigrator(t *testing.T) {
	suite.Run(t, new(policyMigratorTestSuite))
}

type policyMigratorTestSuite struct {
	suite.Suite
	db *bolt.DB
}

var (
	policyID = "0000-0000-0000-0000"
)

func (suite *policyMigratorTestSuite) SetupTest() {
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

func (suite *policyMigratorTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertPolicyIntoBucket(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func comparePolicyWithDB(suite *policyMigratorTestSuite, bucket bolthelpers.BucketRef, policy *storage.Policy) {
	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.Id))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)
}

func strPtr(s string) *string {
	return &s
}

func testPolicy(id string) storage.Policy {
	return storage.Policy{
		Id:          id,
		Remediation: "remediation",
		Rationale:   "rationale",
		Description: "description",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{FieldName: "Process Name", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "iptables"}}},
				},
			},
		},
		Exclusions: []*storage.Exclusion{
			{Name: "exclusion name", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace name"}}},
		},
	}
}

// Test that unrelated policies aren't updated
func (suite *policyMigratorTestSuite) TestUnrelatedPolicyIsNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policyID := "this-is-a-random-id-that-should-not-exist"
	policy := testPolicy(policyID)

	policiesToMigrate := map[string]PolicyChanges{
		"0000-0000-0000-0000": {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
	}

	comparisonPolicies := map[string]*storage.Policy{
		policyID: &policy,
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that an unmodified policy that matches comparison policy is updated
func (suite *policyMigratorTestSuite) TestUnmodifiedAndMatchingPolicyIsUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
	}

	comparisonPolicies := map[string]*storage.Policy{
		policyID: &policy,
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy should've had description changed, but nothing else
	policy.Description = *policiesToMigrate[policyID].ToChange.Description
	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that all unmodified policies are updated
func (suite *policyMigratorTestSuite) TestAllUnmodifiedPoliciesGetUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	policiesToTest := make([]*storage.Policy, 10)
	comparisonPolicies := make(map[string]*storage.Policy)
	policiesToMigrate := make(map[string]PolicyChanges)

	// Create and insert a set of unmodified fake policies
	for i := 0; i < 10; i++ {
		policy := testPolicy(fmt.Sprintf("policy%d", i))
		policiesToTest[i] = &policy
		suite.NoError(insertPolicyIntoBucket(bucket, policy.Id, &policy))
		policy.Description = "sfasdf"

		comparisonPolicy := testPolicy(policy.Id)
		comparisonPolicies[policy.Id] = &comparisonPolicy
		policiesToMigrate[policy.Id] = PolicyChanges{
			FieldsToCompare: []FieldComparator{PolicySectionComparator, ExclusionComparator, RemediationComparator, RationaleComparator},
			ToChange:        PolicyUpdates{Description: strPtr(fmt.Sprintf("%s new description", policy.Id))}, // give them all a new description
		}
	}

	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	for _, policy := range policiesToTest {
		// All of the policies should've changed
		policy.Description = fmt.Sprintf("%s new description", policy.Id)
		comparePolicyWithDB(suite, bucket, policy)
	}
}

// Test that any policies that are not in db are not updated and won't cause an error
func (suite *policyMigratorTestSuite) TestMissingPoliciesDontReturnError() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)
	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
		"missing-policy-id": {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
	}

	comparisonPolicy := testPolicy(policyID)
	comparisonPolicies := map[string]*storage.Policy{
		policyID: &comparisonPolicy,
	}

	// Ensure that running the migration with one of the policiesToMigrate missing won't cause an error
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))
	// And the policy that did exist gets updated
	policy.Description = "this is a new description"
	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that an unmodified policy that doesn't match comparison policy is not updated
func (suite *policyMigratorTestSuite) TestUnmodifiedPolicyThatDoesntMatchIsNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
	}

	comparisonPolicy := testPolicy(policyID)
	comparisonPolicy.Description = "something else"
	comparisonPolicies := map[string]*storage.Policy{
		policyID: &comparisonPolicy,
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that exclusions can get added and removed appropriately
func (suite *policyMigratorTestSuite) TestExclusionAreAddedAndRemovedAsNecessary() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	// Add a bunch of exclusions into the DB
	policy.Exclusions = []*storage.Exclusion{
		{Name: "exclusion0", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-0"}}},
		{Name: "exclusion1", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace 1"}}},
		{Name: "exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-2"}}},
		{Name: "exclusion3", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-3"}}},
		{Name: "exclusion4", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-4"}}},
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policy.Id, &policy))

	comparisonPolicies := map[string]*storage.Policy{
		policyID: &policy,
	}

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{ExclusionComparator},
			ToChange: PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{
					{Name: "exclusion1-changed", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-1"}}},
					{Name: "NEW exclusion", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW"}}},
					{Name: "NEW exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW2"}}},
				},
				ExclusionsToRemove: []*storage.Exclusion{
					{Name: "exclusion1", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace 1"}}},
					{Name: "exclusion4", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-4"}}},
					{Name: "exclusion-NaN", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NaN"}}}, // this exclusion doesn't exist so it shouldn't get removed
				},
			},
		},
	}

	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy exclusions should be updated
	policy.Exclusions = []*storage.Exclusion{
		{Name: "exclusion0", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-0"}}},
		{Name: "exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-2"}}},
		{Name: "exclusion3", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-3"}}},
		{Name: "exclusion1-changed", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-1"}}},
		{Name: "NEW exclusion", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW"}}},
		{Name: "NEW exclusion2", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace-NEW2"}}},
	}

	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that exclusions are added if the policy never had any before
func (suite *policyMigratorTestSuite) TestExclusionAreAddedEvenIfPolicyHadNoneBefore() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	// Remove all exclusions to start with
	policy.Exclusions = nil

	suite.NoError(insertPolicyIntoBucket(bucket, policy.Id, &policy))

	comparisonPolicies := map[string]*storage.Policy{
		policyID: &policy,
	}

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{ExclusionComparator},
			ToChange: PolicyUpdates{
				ExclusionsToAdd: []*storage.Exclusion{
					{Name: "exclusion1-added", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace"}}},
				},
			},
		},
	}

	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy exclusions should be updated
	policy.Exclusions = []*storage.Exclusion{
		{Name: "exclusion1-added", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "namespace"}}},
	}

	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that policies whose exclusions don't match are not updated
func (suite *policyMigratorTestSuite) TestPolicyWithModifiedExclusionIsNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	// Change the exclusions into something else
	policy.Exclusions = []*storage.Exclusion{
		{Name: "alt excl", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "alt ns"}}},
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))

	comparisonPolicy := testPolicy(policyID)
	comparisonPolicies := map[string]*storage.Policy{
		policyID: &comparisonPolicy,
	}

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{ExclusionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("alt description")},
		},
	}

	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy should not have changed
	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that only policies whose policy sections match are updated if that's selected as a comparison
func (suite *policyMigratorTestSuite) TestPolicySectionComparison() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	policiesToTest := make([]*storage.Policy, 3)
	for i := 0; i < 3; i++ {
		policy := testPolicy(fmt.Sprintf("policy%d", i))
		policiesToTest[i] = &policy
		suite.NoError(insertPolicyIntoBucket(bucket, policy.Id, &policy))
	}

	comparisonPolicies := map[string]*storage.Policy{
		"policy0": policiesToTest[0], // keep the first one unmodified
	}

	// .. but modify the second and third ones so that they don't match
	for i := 1; i < 3; i++ {
		comparisonPolicy := testPolicy(fmt.Sprintf("policy%d", i))
		comparisonPolicy.PolicySections[0].PolicyGroups[0].FieldName = "blah"
		comparisonPolicies[comparisonPolicy.Id] = &comparisonPolicy
	}

	policiesToMigrate := map[string]PolicyChanges{
		"policy0": {
			FieldsToCompare: []FieldComparator{PolicySectionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("policy0 description")},
		},
		"policy1": {
			FieldsToCompare: []FieldComparator{PolicySectionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("policy1 description")},
		},
		"policy2": {
			FieldsToCompare: []FieldComparator{DescriptionComparator}, // not comparing policy section so it should get updated regardless of if policy section is different
			ToChange:        PolicyUpdates{Description: strPtr("policy2 description")},
		},
	}

	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Only the first and last policies should have changed
	policiesToTest[0].Description = "policy0 description"
	policiesToTest[2].Description = "policy2 description"

	for _, policy := range policiesToTest {
		comparePolicyWithDB(suite, bucket, policy)
	}
}

// Test that the string fields are updated as necessary
func (suite *policyMigratorTestSuite) TestStringFieldsAreUpdatedIfNecessary() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange: PolicyUpdates{
				Description: strPtr("new description"),
				Rationale:   strPtr("new rationale"),
				Remediation: strPtr("new remediation"),
			},
		},
	}

	comparisonPolicies := map[string]*storage.Policy{
		policyID: &policy,
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy should've had description, rationale and remediation changed, but nothing else
	policy.Description = *policiesToMigrate[policyID].ToChange.Description
	policy.Rationale = *policiesToMigrate[policyID].ToChange.Rationale
	policy.Remediation = *policiesToMigrate[policyID].ToChange.Remediation

	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that policy section property is updated if asked
func (suite *policyMigratorTestSuite) TestPolicySectionIsUpdatedIfNecessary() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange: PolicyUpdates{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: "My field",
								Values:    []*storage.PolicyValue{{Value: "abcdef"}},
							},
						},
					},
				},
			},
		},
	}

	comparisonPolicies := map[string]*storage.Policy{
		policyID: &policy,
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy section should have been updated
	policy.PolicySections = policiesToMigrate[policyID].ToChange.PolicySections

	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that comparisons only compare the specified string field even if the other ones don't match
func (suite *policyMigratorTestSuite) TestPolicyIsUpdatedOnlyIfStringFieldComparisonsPass() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	policiesToTest := make([]*storage.Policy, 3)
	comparisonPolicies := make(map[string]*storage.Policy)

	for i := 0; i < 3; i++ {
		policy := testPolicy(fmt.Sprintf("policy%d", i))
		policiesToTest[i] = &policy
		suite.NoError(insertPolicyIntoBucket(bucket, policy.Id, &policy))
	}

	comparisonPolicy0 := testPolicy("policy0")
	// Everything but description should _not_ match
	comparisonPolicy0.Remediation = "alt remediation"
	comparisonPolicy0.Rationale = "alt rationale"
	comparisonPolicies["policy0"] = &comparisonPolicy0

	comparisonPolicy1 := testPolicy("policy1")
	// Everything but remediation should _not_ match
	comparisonPolicy1.Description = "alt desc"
	comparisonPolicy1.Rationale = "alt rationale"
	comparisonPolicies["policy1"] = &comparisonPolicy1

	comparisonPolicy2 := testPolicy("policy2")
	// Everything but rationale should _not_ match
	comparisonPolicy2.Description = "alt desc"
	comparisonPolicy2.Remediation = "alt remediation"
	comparisonPolicies["policy2"] = &comparisonPolicy2

	policiesToMigrate := map[string]PolicyChanges{
		"policy0": {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("policy0 new description")},
		},
		"policy1": {
			FieldsToCompare: []FieldComparator{RemediationComparator},
			ToChange:        PolicyUpdates{Remediation: strPtr("policy1 new remediation")},
		},
		"policy2": {
			FieldsToCompare: []FieldComparator{RationaleComparator},
			ToChange:        PolicyUpdates{Rationale: strPtr("policy2 new rationale")},
		},
	}

	policiesToTest[0].Description = *policiesToMigrate["policy0"].ToChange.Description
	policiesToTest[1].Remediation = *policiesToMigrate["policy1"].ToChange.Remediation
	policiesToTest[2].Rationale = *policiesToMigrate["policy2"].ToChange.Rationale

	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	for _, policy := range policiesToTest {
		comparePolicyWithDB(suite, bucket, policy)
	}
}

func (suite *policyMigratorTestSuite) TestPolicyIsNotUpdatedIfStringFieldComparisonsFail() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	policiesToTest := make([]*storage.Policy, 3)
	comparisonPolicies := make(map[string]*storage.Policy)
	for i := 0; i < 3; i++ {
		policy := testPolicy(fmt.Sprintf("policy%d", i))

		// Modify the policy so that they don't match the default ones
		policy.Description = "alt desc"
		policy.Remediation = "alt remediation"
		policy.Rationale = "alt rationale"

		policiesToTest[i] = &policy
		suite.NoError(insertPolicyIntoBucket(bucket, policy.Id, &policy))

		// Use the default policy for comparison
		comparisonPolicy := testPolicy(policy.Id)
		comparisonPolicies[policy.Id] = &comparisonPolicy
	}

	policiesToMigrate := map[string]PolicyChanges{
		"policy0": {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("policy0 new description")},
		},
		"policy1": {
			FieldsToCompare: []FieldComparator{RemediationComparator},
			ToChange:        PolicyUpdates{Remediation: strPtr("policy1 new remediation")},
		},
		"policy2": {
			FieldsToCompare: []FieldComparator{RationaleComparator},
			ToChange:        PolicyUpdates{Rationale: strPtr("policy2 new rationale")},
		},
	}

	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// None of the policies should've changed
	for _, policy := range policiesToTest {
		comparePolicyWithDB(suite, bucket, policy)
	}
}

// Test that if multiple fields are to be compared one is modified, then the policy is not modified
func (suite *policyMigratorTestSuite) TestPolicyIsNotUpdatedIfEvenOneFieldIsModified() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)

	// Modify just one of the fields
	policy.Exclusions = append(policy.Exclusions, &storage.Exclusion{Name: "another excl", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "another ns"}}})

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator, ExclusionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("this is a new description")},
		},
	}

	comparisonPolicy := testPolicy(policyID)
	comparisonPolicies := map[string]*storage.Policy{
		policyID: &comparisonPolicy,
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy should be unaltered
	comparePolicyWithDB(suite, bucket, &policy)
}

// If no fields are to be compared, then the policy should be updated even if it's mismatched
func (suite *policyMigratorTestSuite) TestPolicyWithMismatchIsUpdatedIfNoFieldsToCompare() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)
	policy.Description = "alt desc"
	policy.PolicySections[0].PolicyGroups[0].FieldName = "blah" // make sure the policy is different

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: nil, // don't match anything
			ToChange: PolicyUpdates{
				Description: strPtr("new description"),
			},
		},
	}

	comparisonPolicy := testPolicy(policyID)
	comparisonPolicies := map[string]*storage.Policy{
		policyID: &comparisonPolicy,
	}

	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))
	suite.NoError(MigratePolicies(suite.db, policiesToMigrate, comparisonPolicies))

	// Policy should've been updated
	policy.Description = *policiesToMigrate[policyID].ToChange.Description
	comparePolicyWithDB(suite, bucket, &policy)
}

// Test that it will throw an error if a policy is missing from comparison policies
func (suite *policyMigratorTestSuite) TestMissingComparisonPolicyResultsInError() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := testPolicy(policyID)
	suite.NoError(insertPolicyIntoBucket(bucket, policyID, &policy))

	policiesToMigrate := map[string]PolicyChanges{
		policyID: {
			FieldsToCompare: []FieldComparator{DescriptionComparator},
			ToChange:        PolicyUpdates{Description: strPtr("new description")},
		},
	}
	err := MigratePolicies(suite.db, policiesToMigrate, map[string]*storage.Policy{})
	suite.Error(err, "expected an error when comparison policy is missing")
}
