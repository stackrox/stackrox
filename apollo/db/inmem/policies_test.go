package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPolicies(t *testing.T) {
	suite.Run(t, new(PoliciesTestSuite))
}

type PoliciesTestSuite struct {
	suite.Suite
	*InMemoryStore
}

func (suite *PoliciesTestSuite) SetupSuite() {
	persistent, err := createBoltDB()
	require.Nil(suite.T(), err)
	suite.InMemoryStore = New(persistent)
}

func (suite *PoliciesTestSuite) TeardownSuite() {
	suite.Close()
}

func (suite *PoliciesTestSuite) basicPoliciesTest(updateStore, retrievalStore db.PolicyStorage) {
	expectedPolicies := []*v1.Policy{
		{
			Name:     "policy1",
			Severity: v1.Severity_LOW_SEVERITY,
		},
		{
			Name:     "policy2",
			Severity: v1.Severity_HIGH_SEVERITY,
		},
	}

	// Test Add
	for _, p := range expectedPolicies {
		id, err := updateStore.AddPolicy(p)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	// Verify add is persisted
	policies, err := retrievalStore.GetPolicies(&v1.GetPoliciesRequest{})
	suite.Nil(err)
	suite.ElementsMatch(expectedPolicies, policies)

	// Verify update works
	for _, p := range expectedPolicies {
		p.Severity = v1.Severity_MEDIUM_SEVERITY
		suite.NoError(updateStore.UpdatePolicy(p))
	}
	policies, err = retrievalStore.GetPolicies(&v1.GetPoliciesRequest{})
	suite.NoError(err)
	suite.ElementsMatch(expectedPolicies, policies)

	// Verify deletion is persisted
	for _, p := range expectedPolicies {
		suite.NoError(updateStore.RemovePolicy(p.GetId()))
	}
	policies, err = retrievalStore.GetPolicies(&v1.GetPoliciesRequest{})
	suite.NoError(err)
	suite.Len(policies, 0)
}

func (suite *PoliciesTestSuite) TestPersistence() {
	suite.basicPoliciesTest(suite.InMemoryStore, suite.persistent)
}

func (suite *PoliciesTestSuite) TestPolicies() {
	suite.basicPoliciesTest(suite.InMemoryStore, suite.InMemoryStore)
}

func (suite *PoliciesTestSuite) TestGetPoliciesFilters() {
	policy1 := &v1.Policy{
		Name: "policy1",
	}
	policy2 := &v1.Policy{
		Name: "policy2",
	}
	policies := []*v1.Policy{policy1, policy2}
	for _, p := range policies {
		id, err := suite.AddPolicy(p)
		suite.NoError(err)
		suite.NotEmpty(id)
	}
	// Get all policies
	policies, err := suite.GetPolicies(&v1.GetPoliciesRequest{})
	suite.Nil(err)
	suite.ElementsMatch([]*v1.Policy{policy1, policy2}, policies)

	// Get by Name
	policies, err = suite.GetPolicies(&v1.GetPoliciesRequest{Name: []string{policy1.Name}})
	suite.Nil(err)
	suite.ElementsMatch([]*v1.Policy{policy1}, policies)

	// Cleanup
	err = suite.RemovePolicy(policy1.GetId())
	suite.Nil(err)

	err = suite.RemovePolicy(policy2.GetId())
	suite.Nil(err)
}
