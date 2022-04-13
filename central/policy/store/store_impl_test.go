package store

import (
	"errors"
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestPolicyStore(t *testing.T) {
	suite.Run(t, new(PolicyStoreTestSuite))
}

type PolicyStoreTestSuite struct {
	suite.Suite

	db              *bolt.DB
	removedPolicyDB *bolt.DB
	store           Store
}

// Do setup before each test so we have a clean DB
func (suite *PolicyStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	removedPolicyDB, err := bolthelper.NewTemp(suite.T().Name() + "-removed-policies.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.db = db
	suite.removedPolicyDB = removedPolicyDB
	suite.store = newWithoutDefaults(db)
}

// Do teardown after each test because we're doing setup before each test
func (suite *PolicyStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
	testutils.TearDownDB(suite.removedPolicyDB)
}

func (suite *PolicyStoreTestSuite) verifyAddPolicySucceeds(policy *storage.Policy) {
	dbID, err := suite.store.AddPolicy(policy, true)
	suite.NoError(err)
	suite.Equal(policy.GetId(), dbID)
}

func (suite *PolicyStoreTestSuite) verifyPolicyExists(policy *storage.Policy) {
	dbPolicy, exists, err := suite.store.GetPolicy(policy.GetId())
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(policy, dbPolicy)
}

func (suite *PolicyStoreTestSuite) verifyPolicyDoesNotExist(id string) {
	_, exists, err := suite.store.GetPolicy(id)
	suite.NoError(err)
	suite.False(exists)
}

func (suite *PolicyStoreTestSuite) verifyPolicyStoreErrorList(policy *storage.Policy, errorTypes []error) {
	_, err := suite.store.AddPolicy(policy, true)
	suite.Error(err)
	policyStoreErrorList := new(PolicyStoreErrorList)
	suite.Require().IsType(policyStoreErrorList, err)
	if errors.As(err, &policyStoreErrorList) {
		suite.Require().Len(policyStoreErrorList.Errors, len(errorTypes))
		for i, errType := range errorTypes {
			suite.IsType(errType, policyStoreErrorList.Errors[i])
		}
	}
}

func (suite *PolicyStoreTestSuite) TestPolicies() {
	policy1 := &storage.Policy{
		Name:     "policy1",
		Severity: storage.Severity_LOW_SEVERITY,
	}
	policy2 := &storage.Policy{
		Name:     "policy2",
		Severity: storage.Severity_HIGH_SEVERITY,
	}
	policies := []*storage.Policy{policy1, policy2}
	for _, p := range policies {
		id, err := suite.store.AddPolicy(p, true)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	// Get all policies
	retrievedPolicies, err := suite.store.GetAllPolicies()
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	// Update policies with new severity and name.
	for _, p := range policies {
		p.Severity = storage.Severity_MEDIUM_SEVERITY
		p.Name = p.Name + " "
		suite.NoError(suite.store.UpdatePolicy(p))
	}
	retrievedPolicies, err = suite.store.GetAllPolicies()
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	// Revert policy name changes.
	for _, p := range policies {
		p.Name = strings.TrimSpace(p.Name)
		suite.NoError(suite.store.UpdatePolicy(p))
	}
	retrievedPolicies, err = suite.store.GetAllPolicies()
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	for _, p := range policies {
		suite.NoError(suite.store.RemovePolicy(p.GetId()))
	}

	retrievedPolicies, err = suite.store.GetAllPolicies()
	suite.NoError(err)
	suite.Empty(retrievedPolicies)
}

func (suite *PolicyStoreTestSuite) TestAddPolicyIDConflict() {
	id := "SomeID"
	policy1 := &storage.Policy{
		Name: "policy1",
		Id:   id,
	}
	policy2 := &storage.Policy{
		Name: "policy2",
		Id:   id,
	}

	suite.verifyAddPolicySucceeds(policy1)

	suite.verifyPolicyStoreErrorList(policy2, []error{new(IDConflictError)})

	suite.verifyPolicyExists(policy1)
}

func (suite *PolicyStoreTestSuite) TestAddPolicyNameConflict() {
	name := "SomeName"
	policy1 := &storage.Policy{
		Name: name,
		Id:   "abcd",
	}
	policy2 := &storage.Policy{
		Name: name,
		Id:   "zyxw",
	}
	suite.verifyAddPolicySucceeds(policy1)

	suite.verifyPolicyStoreErrorList(policy2, []error{new(NameConflictError)})

	suite.verifyPolicyExists(policy1)

	suite.verifyPolicyDoesNotExist(policy2.GetId())
}

func (suite *PolicyStoreTestSuite) TestAddPolicyNameAndIDConflict() {
	name := "SomeName"
	id := "abcd"
	policy1 := &storage.Policy{
		Name: name,
		Id:   id,
	}
	policy2 := &storage.Policy{
		Name:        name,
		Id:          id,
		Description: "This is a non equal policy",
	}

	suite.verifyAddPolicySucceeds(policy1)

	suite.verifyPolicyStoreErrorList(policy2, []error{new(IDConflictError), new(NameConflictError)})

	suite.verifyPolicyExists(policy1)
}

func (suite *PolicyStoreTestSuite) TestAddSamePolicySucceeds() {
	policy1 := &storage.Policy{
		Name: "Joseph",
		Id:   "Rules",
	}

	suite.verifyAddPolicySucceeds(policy1)

	suite.verifyAddPolicySucceeds(policy1)
}

func (suite *PolicyStoreTestSuite) TestPolicyLockFieldUpdates() {
	policy1 := &storage.Policy{
		Id:                 "policy1",
		Name:               "policy1",
		MitreVectorsLocked: true,
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "t1",
				Techniques: []string{"tt1", "tt2"},
			},
		},
	}
	policy2 := &storage.Policy{
		Id:                 "policy2",
		Name:               "policy2",
		MitreVectorsLocked: false,
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "t1",
				Techniques: []string{"tt1", "tt2"},
			},
		},
	}

	policies := []*storage.Policy{policy1, policy2}
	for _, p := range policies {
		id, err := suite.store.AddPolicy(p, true)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	suite.Error(suite.store.UpdatePolicy(&storage.Policy{
		Id:                 "policy1",
		Name:               "policy1",
		MitreVectorsLocked: true,
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "t2",
				Techniques: []string{"tt1", "tt2"},
			},
		},
	}))

	suite.NoError(suite.store.UpdatePolicy(&storage.Policy{
		Id:                 "policy1",
		Name:               "policy1",
		MitreVectorsLocked: false,
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "t1",
				Techniques: []string{"tt1", "tt2"},
			},
		},
	}))

	suite.NoError(suite.store.UpdatePolicy(&storage.Policy{
		Id:                 "policy2",
		Name:               "policy2",
		MitreVectorsLocked: false,
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "t2",
				Techniques: []string{"tt1", "tt2"},
			},
		},
	}))

	suite.NoError(suite.store.UpdatePolicy(&storage.Policy{
		Id:                 "policy2",
		Name:               "policy2",
		MitreVectorsLocked: true,
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "t2",
				Techniques: []string{"tt1", "tt2"},
			},
		},
	}))

	for _, p := range policies {
		suite.NoError(suite.store.RemovePolicy(p.GetId()))
	}

	policies, err := suite.store.GetAllPolicies()
	suite.NoError(err)
	suite.Empty(policies)
}

func (suite *PolicyStoreTestSuite) TestUpdatePolicyAlreadyExists() {
	policy1 := &storage.Policy{
		Name: "Boo",
		Id:   "boo-1",
	}

	suite.verifyAddPolicySucceeds(policy1)

	suite.NoError(suite.store.UpdatePolicy(&storage.Policy{Id: "boo-1",
		Name: "Foo",
	}))
}

func TestDefaultPolicyRemoval(t *testing.T) {
	db, err := bolthelper.NewTemp(t.Name() + ".db")
	if err != nil {
		assert.FailNow(t, "Failed to make BoltDB", err.Error())
	}
	defer testutils.TearDownDB(db)

	store := New(db)

	policy := &storage.Policy{
		Id:   "da4e0776-159b-42a3-90a9-18cdd9b485ba",
		Name: "OpenShift: Advanced Cluster Security Central Admin Secret Accessed",
	}

	// Test remove.
	err = store.RemovePolicy(policy.GetId())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Default system policies cannot be removed")

	policy = &storage.Policy{
		Id:   "da4e0776-159b-42a3-90a9-18cdd9b48111",
		Name: "OpenShift: Advanced Cluster Security Central Admin Secret Accessed (CUSTOM)",
	}

	_, err = store.AddPolicy(policy, true)
	require.NoError(t, err)

	// Test remove.
	err = store.RemovePolicy(policy.GetId())
	assert.NoError(t, err)
}
