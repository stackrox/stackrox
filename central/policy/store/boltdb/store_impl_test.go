package boltdb

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
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
	ctx             context.Context
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

	suite.ctx = policyCtx
}

// Do teardown after each test because we're doing setup before each test
func (suite *PolicyStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
	testutils.TearDownDB(suite.removedPolicyDB)
}

func (suite *PolicyStoreTestSuite) verifyAddPolicySucceeds(policy *storage.Policy) {
	err := suite.store.Upsert(suite.ctx, policy)
	suite.NoError(err)
}

func (suite *PolicyStoreTestSuite) verifyPolicyExists(policy *storage.Policy) {
	dbPolicy, exists, err := suite.store.Get(suite.ctx, policy.GetId())
	suite.NoError(err)
	suite.True(exists)
	suite.Equal(policy, dbPolicy)
}

func (suite *PolicyStoreTestSuite) verifyPolicyDoesNotExist(id string) {
	_, exists, err := suite.store.Get(suite.ctx, id)
	suite.NoError(err)
	suite.False(exists)
}

func (suite *PolicyStoreTestSuite) verifyPolicyStoreErrorList(policy *storage.Policy, errorTypes []error) {
	err := suite.store.Upsert(suite.ctx, policy)
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
		Id:       "policy1",
		Name:     "policy1",
		Severity: storage.Severity_LOW_SEVERITY,
	}
	policy2 := &storage.Policy{
		Id:       "policy2",
		Name:     "policy2",
		Severity: storage.Severity_HIGH_SEVERITY,
	}
	policies := []*storage.Policy{policy1, policy2}
	for _, p := range policies {
		suite.NoError(suite.store.Upsert(suite.ctx, p))
	}

	// Get all policies
	retrievedPolicies, err := suite.store.GetAll(suite.ctx)
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	// Update policies with new severity and name.
	for _, p := range policies {
		p.Severity = storage.Severity_MEDIUM_SEVERITY
		p.Name = p.Name + " "
		suite.NoError(suite.store.Upsert(suite.ctx, p))
	}
	retrievedPolicies, err = suite.store.GetAll(suite.ctx)
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	// Revert policy name changes.
	for _, p := range policies {
		p.Name = strings.TrimSpace(p.Name)
		suite.NoError(suite.store.Upsert(suite.ctx, p))
	}
	retrievedPolicies, err = suite.store.GetAll(suite.ctx)
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	for _, p := range policies {
		suite.NoError(suite.store.Delete(suite.ctx, p.GetId()))
	}

	retrievedPolicies, err = suite.store.GetAll(suite.ctx)
	suite.NoError(err)
	suite.Empty(retrievedPolicies)
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
		suite.NoError(suite.store.Upsert(suite.ctx, p))
	}

	suite.Error(suite.store.Upsert(suite.ctx, &storage.Policy{
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

	suite.NoError(suite.store.Upsert(suite.ctx, &storage.Policy{
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

	suite.NoError(suite.store.Upsert(suite.ctx, &storage.Policy{
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

	suite.NoError(suite.store.Upsert(suite.ctx, &storage.Policy{
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
		suite.NoError(suite.store.Delete(suite.ctx, p.GetId()))
	}

	policies, err := suite.store.GetAll(suite.ctx)
	suite.NoError(err)
	suite.Empty(policies)
}

func (suite *PolicyStoreTestSuite) TestUpdatePolicyAlreadyExists() {
	policy1 := &storage.Policy{
		Name: "Boo",
		Id:   "boo-1",
	}

	suite.verifyAddPolicySucceeds(policy1)

	suite.NoError(suite.store.Upsert(suite.ctx, &storage.Policy{Id: "boo-1",
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
	err = store.Delete(policyCtx, policy.GetId())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Default system policies cannot be removed")

	policy = &storage.Policy{
		Id:   "da4e0776-159b-42a3-90a9-18cdd9b48111",
		Name: "OpenShift: Advanced Cluster Security Central Admin Secret Accessed (CUSTOM)",
	}

	err = store.Upsert(policyCtx, policy)
	require.NoError(t, err)

	// Test remove.
	err = store.Delete(policyCtx, policy.GetId())
	assert.NoError(t, err)
}
