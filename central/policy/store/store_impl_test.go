package store

import (
	"errors"
	"strings"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestPolicyStore(t *testing.T) {
	suite.Run(t, new(PolicyStoreTestSuite))
}

type PolicyStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

// Do setup before each test so we have a clean DB
func (suite *PolicyStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = newWithoutDefaults(db)
}

// Do teardown after each test because we're doing setup before each test
func (suite *PolicyStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *PolicyStoreTestSuite) verifyAddPolicySucceeds(policy *storage.Policy) {
	dbID, err := suite.store.AddPolicy(policy)
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
	_, err := suite.store.AddPolicy(policy)
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
		id, err := suite.store.AddPolicy(p)
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
