package store

import (
	"os"
	"strings"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
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

func (suite *PolicyStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *PolicyStoreTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *PolicyStoreTestSuite) TestPolicies() {
	policy1 := &v1.Policy{
		Name:     "policy1",
		Severity: v1.Severity_LOW_SEVERITY,
	}
	policy2 := &v1.Policy{
		Name:     "policy2",
		Severity: v1.Severity_HIGH_SEVERITY,
	}
	policies := []*v1.Policy{policy1, policy2}
	for _, p := range policies {
		id, err := suite.store.AddPolicy(p)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	// Get all policies
	retrievedPolicies, err := suite.store.GetPolicies()
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	// Update policies with new severity and name.
	for _, p := range policies {
		p.Severity = v1.Severity_MEDIUM_SEVERITY
		p.Name = p.Name + " "
		suite.NoError(suite.store.UpdatePolicy(p))
	}
	retrievedPolicies, err = suite.store.GetPolicies()
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	// Revert policy name changes.
	for _, p := range policies {
		p.Name = strings.TrimSpace(p.Name)
		suite.NoError(suite.store.UpdatePolicy(p))
	}
	retrievedPolicies, err = suite.store.GetPolicies()
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	for _, p := range policies {
		suite.NoError(suite.store.RemovePolicy(p.GetId()))
	}

	retrievedPolicies, err = suite.store.GetPolicies()
	suite.NoError(err)
	suite.Empty(retrievedPolicies)
}
