package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestImagePolicies(t *testing.T) {
	suite.Run(t, new(ImagePoliciesTestSuite))
}

type ImagePoliciesTestSuite struct {
	suite.Suite
	*InMemoryStore
}

func (suite *ImagePoliciesTestSuite) SetupSuite() {
	persistent, err := createBoltDB()
	require.Nil(suite.T(), err)
	suite.InMemoryStore = New(persistent)
}

func (suite *ImagePoliciesTestSuite) TeardownSuite() {
	suite.Close()
}

func (suite *ImagePoliciesTestSuite) basicImagePoliciesTest(updateStore, retrievalStore db.ImagePolicyStorage) {
	expectedPolicies := []*v1.ImagePolicy{
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
		suite.NoError(updateStore.AddImagePolicy(p))
	}

	// Verify insertion multiple times does not deadlock and causes an error
	for _, p := range expectedPolicies {
		suite.Error(updateStore.AddImagePolicy(p))
	}

	// Verify add is persisted
	imagePolicies, err := retrievalStore.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	suite.Nil(err)
	suite.Equal(expectedPolicies, imagePolicies)

	// Verify update works
	for _, p := range expectedPolicies {
		p.Severity = v1.Severity_MEDIUM_SEVERITY
		suite.NoError(updateStore.UpdateImagePolicy(p))
	}
	imagePolicies, err = retrievalStore.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	suite.NoError(err)
	suite.Equal(expectedPolicies, imagePolicies)

	// Verify deletion is persisted
	for _, p := range expectedPolicies {
		suite.NoError(updateStore.RemoveImagePolicy(p.Name))
	}
	imagePolicies, err = retrievalStore.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	suite.NoError(err)
	suite.Len(imagePolicies, 0)
}

func (suite *ImagePoliciesTestSuite) TestPersistence() {
	suite.basicImagePoliciesTest(suite.InMemoryStore, suite.persistent)
}

func (suite *ImagePoliciesTestSuite) TestImagePolicies() {
	suite.basicImagePoliciesTest(suite.InMemoryStore, suite.InMemoryStore)
}

func (suite *ImagePoliciesTestSuite) TestGetImagePoliciesFilters() {
	policy1 := &v1.ImagePolicy{
		Name: "policy1",
	}
	err := suite.AddImagePolicy(policy1)
	suite.Nil(err)
	policy2 := &v1.ImagePolicy{
		Name: "policy2",
	}
	err = suite.AddImagePolicy(policy2)
	suite.Nil(err)
	// Get all image imagePolicies
	imagePolicies, err := suite.GetImagePolicies(&v1.GetImagePoliciesRequest{})
	suite.Nil(err)
	suite.Equal([]*v1.ImagePolicy{policy1, policy2}, imagePolicies)

	// Get by ID
	imagePolicies, err = suite.GetImagePolicies(&v1.GetImagePoliciesRequest{Name: policy1.Name})
	suite.Nil(err)
	suite.Equal([]*v1.ImagePolicy{policy1}, imagePolicies)

	// Cleanup
	err = suite.RemoveImagePolicy(policy1.Name)
	suite.Nil(err)

	err = suite.RemoveImagePolicy(policy2.Name)
	suite.Nil(err)
}
