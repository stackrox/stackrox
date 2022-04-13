package m65tom66

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/bolthelpers"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestUpdatePoliciesMigration(t *testing.T) {
	suite.Run(t, new(policyUpdatesTestSuite))
}

type policyUpdatesTestSuite struct {
	suite.Suite
	db             *bolt.DB
	policiesToTest map[string]*storage.Policy
}

func (suite *policyUpdatesTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(policyBucketName)
		return err
	}))
	suite.db = db

	suite.policiesToTest = map[string]*storage.Policy{
		k8sDashPolicyID: {
			Id:             k8sDashPolicyID,
			PolicySections: []*storage.PolicySection{{SectionName: "", PolicyGroups: k8sDashExistingPolicyGroups}},
		},
		curlPolicyID: {
			Id:          curlPolicyID,
			Remediation: curlExistingRemediation,
			Exclusions: []*storage.Exclusion{
				{Name: "Don't alert on StackRox collector", Deployment: &storage.Exclusion_Deployment{Name: "collector", Scope: &storage.Scope{Namespace: "stackrox"}}},
				{Name: "Don't alert on StackRox central", Deployment: &storage.Exclusion_Deployment{Name: "central", Scope: &storage.Scope{Namespace: "stackrox"}}},
				{Name: "Don't alert on StackRox sensor", Deployment: &storage.Exclusion_Deployment{Name: "sensor", Scope: &storage.Scope{Namespace: "stackrox"}}},
				{Name: "Don't alert on StackRox admission controller", Deployment: &storage.Exclusion_Deployment{Name: "collector", Scope: &storage.Scope{Namespace: "admission-control"}}},
			},
			PolicySections: []*storage.PolicySection{{SectionName: "", PolicyGroups: curlExistingPolicyGroups}},
		},
		iptablesPolicyID: {
			Id: iptablesPolicyID,
			Exclusions: []*storage.Exclusion{
				{Name: "Don't alert on Kube System Namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "kube-system"}}},
				{Name: "Don't alert on istio-system namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "istio-system"}}},
				{Name: "Don't alert on stackrox namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "stackrox"}}},
				{Name: "Don't alert on openshift-sdn namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-sdn"}}},
			},
			PolicySections: []*storage.PolicySection{{SectionName: "", PolicyGroups: iptablesExistingPolicyGroups}},
		},
	}
}

func (suite *policyUpdatesTestSuite) TearDownTest() {
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

func checkPolicyMatches(suite *policyUpdatesTestSuite, bucket bolthelpers.BucketRef, policy *storage.Policy) {
	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.EqualValues(policy, &newPolicy)
}

func checkAllPoliciesMatch(suite *policyUpdatesTestSuite, bucket bolthelpers.BucketRef, policies []*storage.Policy) {
	for _, policy := range policies {
		checkPolicyMatches(suite, bucket, policy)
	}
}

// Test that unrelated policies aren't migrated
func (suite *policyUpdatesTestSuite) TestUnrelatedPolicyIsNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id: "this-is-a-random-id-that-should-not-exist",
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePolicies(suite.db))
	checkPolicyMatches(suite, bucket, policy)
}

// Test that all unmodified policies are migrated
func (suite *policyUpdatesTestSuite) TestAllUnmodifiedPoliciesGetMigrated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	for policyID, policy := range suite.policiesToTest {
		suite.NoError(insertPolicy(bucket, policyID, policy))
	}
	suite.NoError(updatePolicies(suite.db))

	// Check k8s dashboard policy was updated
	expectedPolicy := suite.policiesToTest[k8sDashPolicyID]
	// Manually updating the whole thing otherwise it would modify the actual data used by policiesToTest and fail other tests
	expectedPolicy.PolicySections[0].PolicyGroups = []*storage.PolicyGroup{
		{FieldName: "Image Remote", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: k8sDashNewCriteria}}},
	}
	checkPolicyMatches(suite, bucket, expectedPolicy)

	// Check curl in image policy was updated
	expectedPolicy = suite.policiesToTest[curlPolicyID]
	expectedPolicy.Remediation = curlNewRemediation
	checkPolicyMatches(suite, bucket, expectedPolicy)

	// Check privileged iptables policy was updated
	expectedPolicy = suite.policiesToTest[iptablesPolicyID]
	expectedPolicy.Exclusions = []*storage.Exclusion{
		{Name: "Don't alert on Kube System Namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "kube-system"}}},
		{Name: "Don't alert on istio-system namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "istio-system"}}},
		{Name: "Don't alert on openshift-sdn namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-sdn"}}},
	}
	checkPolicyMatches(suite, bucket, expectedPolicy)
}

func (suite *policyUpdatesTestSuite) TestK8sDashPolicyWithModifiedCriteriaIsNotMigrated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id:             k8sDashPolicyID,
		PolicySections: []*storage.PolicySection{{SectionName: "", PolicyGroups: []*storage.PolicyGroup{{FieldName: "Image Remote", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "something else*"}}}}}},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePolicies(suite.db))
	checkPolicyMatches(suite, bucket, policy)
}

func (suite *policyUpdatesTestSuite) TestCurlPolicyWithModifiedRemediationIsNotMigrated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id:             curlPolicyID,
		Remediation:    "something else",
		PolicySections: []*storage.PolicySection{{SectionName: "", PolicyGroups: curlExistingPolicyGroups}},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePolicies(suite.db))
	checkPolicyMatches(suite, bucket, policy)
}

func (suite *policyUpdatesTestSuite) TestIptablesPolicyWithoutStackroxExclusionIsNotMigrated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id: iptablesPolicyID,
		Exclusions: []*storage.Exclusion{
			{Name: "Don't alert on Kube System Namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "kube-system"}}},
			{Name: "Don't alert on istio-system namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "istio-system"}}},
			{Name: "Don't alert on stackrox namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "stackrrrroxx"}}},
			{Name: "Don't alert on openshift-sdn namespace", Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "openshift-sdn"}}},
		},
		PolicySections: []*storage.PolicySection{{SectionName: "", PolicyGroups: iptablesExistingPolicyGroups}},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePolicies(suite.db))
	checkPolicyMatches(suite, bucket, policy)
}

func (suite *policyUpdatesTestSuite) TestModifiedPoliciesAreNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	// Test that a policy that matches id, but has multiple policy sections is not updated
	var modifiedPolicies []*storage.Policy
	for policyID, policy := range suite.policiesToTest {
		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections = []*storage.PolicySection{
			{
				SectionName:  "",
				PolicyGroups: policy.PolicySections[0].PolicyGroups,
			},
			{
				SectionName:  "section 2",
				PolicyGroups: []*storage.PolicyGroup{{FieldName: "Image OS", Values: []*storage.PolicyValue{{Value: "ubuntu:19.04"}}}},
			},
		}
		suite.NoError(insertPolicy(bucket, policyID, modifiedPolicy))
		modifiedPolicies = append(modifiedPolicies, modifiedPolicy)
	}

	suite.NoError(updatePolicies(suite.db))
	checkAllPoliciesMatch(suite, bucket, modifiedPolicies)

	// Test that a policy that matches id, but has additional policy groups is not updated
	modifiedPolicies = nil
	for policyID, policy := range suite.policiesToTest {
		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections[0].PolicyGroups = append(modifiedPolicy.PolicySections[0].PolicyGroups, &storage.PolicyGroup{FieldName: "someField", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "someValue"}}})
		suite.NoError(insertPolicy(bucket, policyID, modifiedPolicy))
		modifiedPolicies = append(modifiedPolicies, modifiedPolicy)
	}

	suite.NoError(updatePolicies(suite.db))
	checkAllPoliciesMatch(suite, bucket, modifiedPolicies)

	// Test that a policy that matches id, field name _but not_ criteria is not updated
	modifiedPolicies = nil
	for policyID, policy := range suite.policiesToTest {
		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections = []*storage.PolicySection{
			{
				SectionName: "",
				PolicyGroups: []*storage.PolicyGroup{
					{FieldName: policy.PolicySections[0].PolicyGroups[0].FieldName, BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "fake"}}},
				},
			},
		}
		suite.NoError(insertPolicy(bucket, policyID, modifiedPolicy))
		modifiedPolicies = append(modifiedPolicies, modifiedPolicy)
	}

	suite.NoError(updatePolicies(suite.db))
	checkAllPoliciesMatch(suite, bucket, modifiedPolicies)
}
