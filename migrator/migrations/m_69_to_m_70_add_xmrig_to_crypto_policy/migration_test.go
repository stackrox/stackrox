package m69tom70

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/bolthelpers"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

var (
	policyBucketName  = []byte("policies")
	expectedPolicyDir = "testdata"
)

func TestPolicyMigration(t *testing.T) {
	suite.Run(t, new(policyUpdatesTestSuite))
}

type policyUpdatesTestSuite struct {
	suite.Suite
	db               *bolt.DB
	expectedPolicies map[string]*storage.Policy
}

func (suite *policyUpdatesTestSuite) SetupSuite() {
	suite.expectedPolicies = make(map[string]*storage.Policy)

	for policyID := range policiesToMigrate {
		policyPath := filepath.Join(expectedPolicyDir, fmt.Sprintf("%s.json", policyID))
		contents, err := os.ReadFile(policyPath)
		suite.NoError(err)

		policy := &storage.Policy{}
		err = jsonpb.Unmarshal(bytes.NewReader(contents), policy)
		suite.NoError(err)

		suite.expectedPolicies[policy.Id] = policy
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

	suite.EqualValues(policy, &newPolicy)
}

func checkPolicyNotMatches(suite *policyUpdatesTestSuite, bucket bolthelpers.BucketRef, policy *storage.Policy) {
	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))

	suite.NotEqualValues(policy, &newPolicy)
}

// Test that all unmodified policies are migrated
func (suite *policyUpdatesTestSuite) TestAllUnmodifiedPoliciesGetMigrated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	beforeMigrationPolicies, err := getComparisonPoliciesFromFiles()
	suite.NoError(err)

	for policyID := range policiesToMigrate {
		policy, ok := beforeMigrationPolicies[policyID]
		suite.True(ok)

		suite.NoError(insertPolicy(bucket, policy.GetId(), policy))
	}

	suite.NoError(updatePolicies(suite.db))

	for policyID := range policiesToMigrate {
		expectedPolicy, ok := suite.expectedPolicies[policyID]
		suite.True(ok)

		checkPolicyMatches(suite, bucket, expectedPolicy)
	}
}

func (suite *policyUpdatesTestSuite) TestModifiedPoliciesAreNotUpdated() {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	modifiedPolicies := make(map[string]*storage.Policy)
	comparisonPolicies, err := getComparisonPoliciesFromFiles()
	suite.NoError(err)

	for policyID := range policiesToMigrate {
		policy, ok := comparisonPolicies[policyID]
		suite.True(ok)

		// Modify the policy slightly
		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections[0].PolicyGroups[0].Values[0].Value = "assfasdf"

		suite.NoError(insertPolicy(bucket, modifiedPolicy.GetId(), modifiedPolicy))

		modifiedPolicies[policyID] = modifiedPolicy
	}

	suite.NoError(updatePolicies(suite.db))

	for policyID := range policiesToMigrate {
		expectedPolicy, ok := suite.expectedPolicies[policyID]
		suite.True(ok)

		modifiedPolicy, ok := modifiedPolicies[policyID]
		suite.True(ok)

		checkPolicyMatches(suite, bucket, modifiedPolicy)
		checkPolicyNotMatches(suite, bucket, expectedPolicy)
	}
}
