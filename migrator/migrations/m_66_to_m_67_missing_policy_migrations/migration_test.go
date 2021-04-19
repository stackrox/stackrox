package m66tom67

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

var (
	testPolicyDir = "testdata"
)

func TestUpdatePoliciesMigration(t *testing.T) {
	suite.Run(t, new(policyUpdatesTestSuite))
}

type policyUpdatesTestSuite struct {
	suite.Suite
	db              *bolt.DB
	defaultPolicies map[string]*storage.Policy
}

func (suite *policyUpdatesTestSuite) SetupSuite() {
	suite.defaultPolicies = make(map[string]*storage.Policy)

	for policyID := range policiesToMigrate {
		policyPath := filepath.Join(testPolicyDir, fmt.Sprintf("%s.json", policyID))
		contents, err := os.ReadFile(policyPath)
		suite.NoError(err)

		r := new(storage.Policy)
		err = jsonpb.Unmarshal(bytes.NewReader(contents), r)
		suite.NoError(err)

		suite.defaultPolicies[r.Id] = r
	}
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
}

func (suite *policyUpdatesTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func insertPolicy(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	return bucket.Update(func(b *bolt.Bucket) error {
		policyBytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), policyBytes)
	})
}

func checkPolicyMatches(suite *policyUpdatesTestSuite, bucket bolthelpers.BucketRef, policy *storage.Policy) {
	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))

	// Sort the exclusions so that the order doesn't matter in comparison
	sort.SliceStable(policy.Exclusions, func(i, j int) bool {
		return policy.Exclusions[i].Name < policy.Exclusions[j].Name
	})
	sort.SliceStable(newPolicy.Exclusions, func(i, j int) bool {
		return newPolicy.Exclusions[i].Name < newPolicy.Exclusions[j].Name
	})

	suite.EqualValues(policy, &newPolicy)
}

func checkAllPoliciesMatch(policiesToTest []*storage.Policy, suite *policyUpdatesTestSuite, bucket bolthelpers.BucketRef) {
	for _, policy := range policiesToTest {
		checkPolicyMatches(suite, bucket, policy)
	}
}

// Test that unrelated policies aren't migrated
func (suite *policyUpdatesTestSuite) TestUnrelatedPolicyIsNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)
	policy := &storage.Policy{
		Id:          "this-is-a-random-id-that-should-not-exist",
		Remediation: "remediation",
		Description: "description",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{FieldName: "Process Name", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "iptables"}}},
				},
			},
		},
	}

	suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	suite.NoError(updatePolicies(suite.db))

	checkPolicyMatches(suite, bucket, policy)
}

// Test that all unmodified policies are migrated
func (suite *policyUpdatesTestSuite) TestAllUnmodifiedPoliciesGetMigrated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	var policiesToTest []*storage.Policy
	comparisonPolicies, err := getComparisonPoliciesFromFiles()
	suite.NoError(err)

	for policyID := range policiesToMigrate {
		policy, ok := comparisonPolicies[policyID]
		suite.True(ok)

		suite.NoError(insertPolicy(bucket, policy.GetId(), policy))

		policiesToTest = append(policiesToTest, suite.defaultPolicies[policyID])
	}

	suite.NoError(updatePolicies(suite.db))
	checkAllPoliciesMatch(policiesToTest, suite, bucket)
}

func (suite *policyUpdatesTestSuite) TestModifiedPoliciesAreNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	// Test that a policy that matches id, but has multiple policy sections is not updated
	var policiesToTest []*storage.Policy
	comparisonPolicies, err := getComparisonPoliciesFromFiles()
	suite.NoError(err)

	for policyID := range policiesToMigrate {
		policy, ok := comparisonPolicies[policyID]
		suite.True(ok)

		// Modify the policy to have multiple policy sections
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

		suite.NoError(insertPolicy(bucket, modifiedPolicy.GetId(), modifiedPolicy))

		policiesToTest = append(policiesToTest, modifiedPolicy)
	}

	suite.NoError(updatePolicies(suite.db))
	checkAllPoliciesMatch(policiesToTest, suite, bucket)

	// Test that a policy that matches id, but has additional policy groups is not updated
	policiesToTest = nil
	for policyID := range policiesToMigrate {
		policy, ok := comparisonPolicies[policyID]
		suite.True(ok)

		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections[0].PolicyGroups = append(policy.PolicySections[0].PolicyGroups, &storage.PolicyGroup{FieldName: "someField", BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "someValue"}}})

		suite.NoError(insertPolicy(bucket, modifiedPolicy.GetId(), modifiedPolicy))

		policiesToTest = append(policiesToTest, modifiedPolicy)
	}

	suite.NoError(updatePolicies(suite.db))
	checkAllPoliciesMatch(policiesToTest, suite, bucket)

	// Test that a policy that matches id, field name _but not_ criteria is not updated
	policiesToTest = nil
	for policyID := range policiesToMigrate {
		policy, ok := comparisonPolicies[policyID]
		suite.True(ok)

		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections = []*storage.PolicySection{
			{
				SectionName: "",
				PolicyGroups: []*storage.PolicyGroup{
					{FieldName: policy.PolicySections[0].PolicyGroups[0].FieldName, BooleanOperator: storage.BooleanOperator_OR, Negate: false, Values: []*storage.PolicyValue{{Value: "fake"}}},
				},
			},
		}

		suite.NoError(insertPolicy(bucket, policy.GetId(), modifiedPolicy))

		policiesToTest = append(policiesToTest, modifiedPolicy)
	}

	suite.NoError(updatePolicies(suite.db))
	checkAllPoliciesMatch(policiesToTest, suite, bucket)
}
