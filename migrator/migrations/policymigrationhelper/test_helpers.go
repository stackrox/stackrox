package policymigrationhelper

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func insertPolicy(t *testing.T, bucket bolthelpers.BucketRef, policy *storage.Policy) {
	require.NoError(t, bucket.Update(func(b *bolt.Bucket) error {
		policyBytes, err := proto.Marshal(policy)
		if err != nil {
			return err
		}
		return b.Put([]byte(policy.GetId()), policyBytes)
	}))
}

func getAndNormalizePolicies(t *testing.T, bucket bolthelpers.BucketRef, policy *storage.Policy) (normalizedExpected, normalizedFromDB *storage.Policy) {
	normalizedFromDB = &storage.Policy{}
	assert.NoError(t, bucket.View(func(b *bolt.Bucket) error {
		v := b.Get([]byte(policy.GetId()))
		return proto.Unmarshal(v, normalizedFromDB)
	}))

	normalizedExpected = policy.Clone()
	sort.Slice(normalizedExpected.GetExclusions(), func(i, j int) bool {
		return normalizedExpected.Exclusions[i].Name < normalizedExpected.Exclusions[j].Name
	})
	sort.Slice(normalizedFromDB.GetExclusions(), func(i, j int) bool {
		return normalizedFromDB.Exclusions[i].Name < normalizedFromDB.Exclusions[j].Name
	})
	return normalizedExpected, normalizedFromDB
}

func checkPolicyMatches(t *testing.T, bucket bolthelpers.BucketRef, policy *storage.Policy) {
	normalizedExpected, normalizedFromDB := getAndNormalizePolicies(t, bucket, policy)
	assert.EqualValues(t, normalizedExpected, normalizedFromDB)
}

func checkPolicyNotMatches(t *testing.T, bucket bolthelpers.BucketRef, policy *storage.Policy) {
	normalizedExpected, normalizedFromDB := getAndNormalizePolicies(t, bucket, policy)
	assert.NotEqualValues(t, normalizedExpected, normalizedFromDB)
}

// DiffTestSuite is a helper suite that can be embedded for tests that use the PolicyDiff functionality to migrate policies.
type DiffTestSuite struct {
	suite.Suite
	PolicyDiffFS embed.FS

	beforePolicies, afterPolicies map[string]*storage.Policy

	db *bolt.DB
}

// SetupSuite implements the suite contract.
func (suite *DiffTestSuite) SetupSuite() {
	beforePolicyFiles, err := fs.ReadDir(suite.PolicyDiffFS, beforeDirName)
	suite.Require().NoError(err)
	afterPolicyFiles, err := fs.ReadDir(suite.PolicyDiffFS, afterDirName)
	suite.Require().NoError(err)

	// Ensure we have the same file names.
	suite.ElementsMatch(sliceutils.Map(beforePolicyFiles, func(d fs.DirEntry) string {
		return d.Name()
	}), sliceutils.Map(afterPolicyFiles, func(d fs.DirEntry) string {
		return d.Name()
	}))

	suite.beforePolicies = make(map[string]*storage.Policy)
	suite.afterPolicies = make(map[string]*storage.Policy)
	for _, f := range beforePolicyFiles {
		policy, err := ReadPolicyFromFile(suite.PolicyDiffFS, filepath.Join(beforeDirName, f.Name()))
		suite.Require().NoError(err)
		suite.Require().NotEmpty(policy.GetId())
		suite.beforePolicies[policy.GetId()] = policy
	}
	for _, f := range afterPolicyFiles {
		policy, err := ReadPolicyFromFile(suite.PolicyDiffFS, filepath.Join(afterDirName, f.Name()))
		suite.Require().NoError(err)
		suite.Require().NotEmpty(policy.GetId())
		suite.afterPolicies[policy.GetId()] = policy
	}
}

// SetupTest implements the Suite contract.
func (suite *DiffTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	suite.Require().NoError(err)
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(policyBucketName)
		return err
	}))
	suite.db = db
}

// RunTests runs the common tests we would expect to run for policy updates.
func (suite *DiffTestSuite) RunTests(migrationFunc func(db *bolt.DB) error, modifyPolicySectionAndTest bool) {
	suite.Run("TestUnmodifiedPoliciesAreMigrated", func() {
		suite.testUnmodifiedPolicies(migrationFunc)
	})
	if modifyPolicySectionAndTest {
		suite.Run("TestModifiedPoliciesAreNotMigrated", func() {
			suite.testModifiedPolicies(migrationFunc)
		})
	}
}

func (suite *DiffTestSuite) testUnmodifiedPolicies(migrationFunc func(db *bolt.DB) error) {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	for _, policy := range suite.beforePolicies {
		insertPolicy(suite.T(), bucket, policy)
	}

	suite.NoError(migrationFunc(suite.db))

	for policyID := range suite.beforePolicies {
		expectedPolicy, ok := suite.afterPolicies[policyID]
		suite.True(ok)
		checkPolicyMatches(suite.T(), bucket, expectedPolicy)
	}
}

// pollutePolicyContents pollutes policy contents with gibberish values
// fields to pollute - policy section values & remediation - were chosen arbitrary
// running policy migration on such a polluted policy allows to test a scenario where the "before" state does not match the current state
func pollutePolicyContents(policy *storage.Policy) {
	for i := range policy.PolicySections {
		for j := range policy.PolicySections[i].PolicyGroups {
			for k := range policy.PolicySections[i].PolicyGroups[j].Values {
				policy.PolicySections[i].PolicyGroups[j].Values[k].Value = "gibberish"
			}
		}
	}
	policy.Remediation = "gibberish"
}

func (suite *DiffTestSuite) testModifiedPolicies(migrationFunc func(db *bolt.DB) error) {
	bucket := bolthelpers.TopLevelRef(suite.db, policyBucketName)

	modifiedPolicies := make(map[string]*storage.Policy)

	for _, policy := range suite.beforePolicies {
		modifiedPolicy := policy.Clone()
		pollutePolicyContents(modifiedPolicy)
		modifiedPolicies[modifiedPolicy.GetId()] = modifiedPolicy
		insertPolicy(suite.T(), bucket, modifiedPolicy)
	}

	suite.NoError(migrationFunc(suite.db))

	for policyID := range suite.beforePolicies {
		expectedPolicy, ok := suite.afterPolicies[policyID]
		suite.True(ok)

		modifiedPolicy, ok := modifiedPolicies[policyID]
		suite.True(ok)

		checkPolicyMatches(suite.T(), bucket, modifiedPolicy)
		checkPolicyNotMatches(suite.T(), bucket, expectedPolicy)
	}
}

// TestSuite is a helper suite that can be embedded for tests involving migrations of policies.
// Deprecated: New migrations should strive to use DiffTestSuite (and the PolicyDiff) approach instead.
type TestSuite struct {
	suite.Suite

	PoliciesToMigrate   map[string]PolicyChanges
	PreMigPoliciesFS    embed.FS
	PreMigPoliciesDir   string
	ExpectedPoliciesDir string

	DB               *bolt.DB
	ExpectedPolicies map[string]*storage.Policy
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
		err = jsonutil.JSONBytesToProto(contents, &policy)
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
		err = jsonutil.JSONBytesToProto(policyBytes, &policy)
		suite.Require().NoError(err)
		insertPolicy(suite.T(), bucket, &policy)
	}

	suite.NoError(migrationFunc(suite.DB))

	for policyID := range suite.PoliciesToMigrate {
		expectedPolicy, ok := suite.ExpectedPolicies[policyID]
		suite.True(ok)

		checkPolicyMatches(suite.T(), bucket, expectedPolicy)
	}

}

func (suite *TestSuite) testModifiedPolicies(migrationFunc func(db *bolt.DB) error) {
	bucket := bolthelpers.TopLevelRef(suite.DB, policyBucketName)

	modifiedPolicies := make(map[string]*storage.Policy)

	for policyID := range suite.PoliciesToMigrate {
		policyBytes, err := suite.PreMigPoliciesFS.ReadFile(fmt.Sprintf("%s/%s.json", suite.PreMigPoliciesDir, policyID))
		suite.Require().NoError(err)
		var policy storage.Policy
		err = jsonutil.JSONBytesToProto(policyBytes, &policy)
		suite.Require().NoError(err)
		modifiedPolicy := policy.Clone()
		modifiedPolicy.PolicySections[0].PolicyGroups[0].Values[0].Value = "assfasdf"
		modifiedPolicies[modifiedPolicy.GetId()] = modifiedPolicy
		insertPolicy(suite.T(), bucket, modifiedPolicy)
	}

	suite.NoError(migrationFunc(suite.DB))

	for policyID := range suite.PoliciesToMigrate {
		expectedPolicy, ok := suite.ExpectedPolicies[policyID]
		suite.True(ok)

		modifiedPolicy, ok := modifiedPolicies[policyID]
		suite.True(ok)

		checkPolicyMatches(suite.T(), bucket, modifiedPolicy)
		checkPolicyNotMatches(suite.T(), bucket, expectedPolicy)
	}
}
