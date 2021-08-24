package policymigrationhelper

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

// TestSuite is a helper suite that can be embedded for tests involving migrations of policies.
type TestSuite struct {
	suite.Suite

	PoliciesToMigrate   map[string]PolicyChanges
	PreMigPoliciesFS    embed.FS
	PreMigPoliciesDir   string
	ExpectedPoliciesDir string

	DB               *bolt.DB
	ExpectedPolicies map[string]*storage.Policy
}

func (suite *TestSuite) insertPolicy(bucket bolthelpers.BucketRef, policy *storage.Policy) {
	suite.Require().NoError(bucket.Update(func(b *bolt.Bucket) error {
		policyBytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return b.Put([]byte(policy.GetId()), policyBytes)
	}))
}

func (suite *TestSuite) checkPolicyMatches(bucket bolthelpers.BucketRef, policy *storage.Policy) {
	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))

	suite.EqualValues(policy, &newPolicy)
}

func (suite *TestSuite) checkPolicyNotMatches(bucket bolthelpers.BucketRef, policy *storage.Policy) {
	var newPolicy storage.Policy
	suite.NoError(bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, &newPolicy)
	}))
	suite.NotEqualValues(policy, &newPolicy)
}

// SetupSuite implements the Suite contract.
func (suite *TestSuite) SetupSuite() {
	suite.Require().NotEmpty(suite.PoliciesToMigrate)
	suite.ExpectedPolicies = make(map[string]*storage.Policy)

	for policyID := range suite.PoliciesToMigrate {
		policyPath := filepath.Join(suite.ExpectedPoliciesDir, fmt.Sprintf("%s.json", policyID))
		contents, err := os.ReadFile(policyPath)
		suite.NoError(err)

		var policy storage.Policy
		err = jsonpb.Unmarshal(bytes.NewReader(contents), &policy)
		suite.NoError(err)

		suite.ExpectedPolicies[policy.Id] = &policy
	}
}

// SetupTest implements the Suite contract.
func (suite *TestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(policyBucketName)
		return err
	}))
	suite.DB = db
}

// TearDownTest implements the Suite contract.
func (suite *TestSuite) TearDownTest() {
	testutils.TearDownDB(suite.DB)
}

// RunTests runs the common tests we would expect to run for policy updates.
func (suite *TestSuite) RunTests(migrationFunc func(db *bolt.DB) error) {
	suite.Run("TestUnmodifiedPoliciesAreMigrated", func() {
		suite.testUnmodifiedPolicies(migrationFunc)
	})
	suite.Run("TestModifiedPoliciesAreNotMigrated", func() {
		suite.testModifiedPolicies(migrationFunc)
	})
}

func (suite *TestSuite) testUnmodifiedPolicies(migrationFunc func(db *bolt.DB) error) {
	bucket := bolthelpers.TopLevelRef(suite.DB, policyBucketName)

	for policyID := range suite.PoliciesToMigrate {
		policyBytes, err := suite.PreMigPoliciesFS.ReadFile(fmt.Sprintf("%s/%s.json", suite.PreMigPoliciesDir, policyID))
		suite.Require().NoError(err)
		var policy storage.Policy
		err = jsonpb.Unmarshal(bytes.NewReader(policyBytes), &policy)
		suite.Require().NoError(err)
		suite.insertPolicy(bucket, &policy)
	}

	suite.NoError(migrationFunc(suite.DB))

	for policyID := range suite.PoliciesToMigrate {
		expectedPolicy, ok := suite.ExpectedPolicies[policyID]
		suite.True(ok)

		suite.checkPolicyMatches(bucket, expectedPolicy)
	}

}

func (suite *TestSuite) testModifiedPolicies(migrationFunc func(db *bolt.DB) error) {
	bucket := bolthelpers.TopLevelRef(suite.DB, policyBucketName)

	modifiedPolicies := make(map[string]*storage.Policy)

	for policyID := range suite.PoliciesToMigrate {
		policyBytes, err := suite.PreMigPoliciesFS.ReadFile(fmt.Sprintf("%s/%s.json", suite.PreMigPoliciesDir, policyID))
		suite.Require().NoError(err)
		var policy storage.Policy
		err = jsonpb.Unmarshal(bytes.NewReader(policyBytes), &policy)
		suite.Require().NoError(err)
		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections[0].PolicyGroups[0].Values[0].Value = "assfasdf"
		modifiedPolicies[modifiedPolicy.GetId()] = modifiedPolicy
		suite.insertPolicy(bucket, modifiedPolicy)
	}

	suite.NoError(migrationFunc(suite.DB))

	for policyID := range suite.PoliciesToMigrate {
		expectedPolicy, ok := suite.ExpectedPolicies[policyID]
		suite.True(ok)

		modifiedPolicy, ok := modifiedPolicies[policyID]
		suite.True(ok)

		suite.checkPolicyMatches(bucket, modifiedPolicy)
		suite.checkPolicyNotMatches(bucket, expectedPolicy)
	}
}
