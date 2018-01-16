package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltPolicies(t *testing.T) {
	suite.Run(t, new(BoltPoliciesTestSuite))
}

type BoltPoliciesTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltPoliciesTestSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := New(tmpDir)
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltPoliciesTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltPoliciesTestSuite) TestPolicies() {
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
		id, err := suite.AddPolicy(p)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	// Get all alerts
	retrievedPolicies, err := suite.GetPolicies(&v1.GetPoliciesRequest{})
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	for _, p := range policies {
		p.Severity = v1.Severity_MEDIUM_SEVERITY
		suite.NoError(suite.UpdatePolicy(p))
	}
	retrievedPolicies, err = suite.GetPolicies(&v1.GetPoliciesRequest{})
	suite.Nil(err)
	suite.ElementsMatch(policies, retrievedPolicies)

	for _, p := range policies {
		suite.NoError(suite.RemovePolicy(p.GetId()))
	}

	retrievedPolicies, err = suite.GetPolicies(&v1.GetPoliciesRequest{})
	suite.NoError(err)
	suite.Empty(retrievedPolicies)
}
