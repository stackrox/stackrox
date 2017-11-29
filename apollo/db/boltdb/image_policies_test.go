package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltImagePolicies(t *testing.T) {
	suite.Run(t, new(BoltImagePoliciesTestSuite))
}

type BoltImagePoliciesTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltImagePoliciesTestSuite) SetupSuite() {
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

func (suite *BoltImagePoliciesTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltImagePoliciesTestSuite) TestImagePolicies() {
	policy1 := &v1.ImagePolicy{
		Name:     "policy1",
		Severity: v1.Severity_LOW_SEVERITY,
	}
	err := suite.AddImagePolicy(policy1)
	suite.Nil(err)

	policy2 := &v1.ImagePolicy{
		Name:     "policy2",
		Severity: v1.Severity_HIGH_SEVERITY,
	}
	err = suite.AddImagePolicy(policy2)
	suite.Nil(err)
	// Get all alerts
	imagePolicies, err := suite.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImagePolicy{policy1, policy2}, imagePolicies)

	policy1.Severity = v1.Severity_HIGH_SEVERITY
	err = suite.UpdateImagePolicy(policy1)
	suite.Nil(err)
	imagePolicies, err = suite.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImagePolicy{policy1, policy2}, imagePolicies)

	err = suite.RemoveImagePolicy(policy1.Name)
	suite.Nil(err)
	imagePolicies, err = suite.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImagePolicy{policy2}, imagePolicies)
}
