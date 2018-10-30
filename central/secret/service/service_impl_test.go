package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	datastoreMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestSecretService(t *testing.T) {
	suite.Run(t, new(SecretServiceTestSuite))
}

type SecretServiceTestSuite struct {
	suite.Suite

	mockSecretStore     *datastoreMocks.DataStore
	mockDeploymentStore *deploymentMocks.DataStore

	service Service
}

func (suite *SecretServiceTestSuite) SetupTest() {
	suite.mockSecretStore = &datastoreMocks.DataStore{}
	suite.mockDeploymentStore = &deploymentMocks.DataStore{}

	suite.service = New(suite.mockSecretStore, suite.mockDeploymentStore)
}

// Test happy path for getting secrets and relationships
func (suite *SecretServiceTestSuite) TestGetSecret() {
	secretID := "id1"

	expectedSecret := &v1.Secret{
		Id:        secretID,
		Name:      "secretname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}
	suite.mockSecretStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	results := []*v1.SearchResult{
		{
			Id:   "d1",
			Name: "deployment1",
		},
	}

	suite.mockDeploymentStore.On("SearchDeployments", psr).Return(results, nil)

	expectedRelationship := &v1.SecretRelationship{
		DeploymentRelationships: []*v1.SecretDeploymentRelationship{
			{
				Id:   "d1",
				Name: "deployment1",
			},
		},
	}

	actualSecretAndRelationship, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.NoError(err)
	suite.Equal(expectedSecret, actualSecretAndRelationship)
	suite.Equal(expectedRelationship, actualSecretAndRelationship.GetRelationship())

	suite.mockSecretStore.AssertExpectations(suite.T())
}

// Test that when we fail to find a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretNotExists() {
	secretID := "id1"

	suite.mockSecretStore.On("GetSecret", secretID).Return((*v1.Secret)(nil), false, nil)

	_, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(err)

	suite.mockSecretStore.AssertExpectations(suite.T())
}

// Test that when we fail to read the db for a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretFailure() {
	secretID := "id1"

	expectedErr := fmt.Errorf("failure")
	suite.mockSecretStore.On("GetSecret", secretID).Return((*v1.Secret)(nil), true, expectedErr)

	_, actualErr := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(actualErr)

	suite.mockSecretStore.AssertExpectations(suite.T())
}

// Test that when we fail to find a relationship, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithNoRelationship() {
	secretID := "id1"

	expectedSecret := &v1.Secret{
		Id:        secretID,
		Name:      "secretname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}
	suite.mockSecretStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	suite.mockDeploymentStore.On("SearchDeployments", psr).Return([]*v1.SearchResult{}, nil)

	_, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.NoError(err)

	suite.mockSecretStore.AssertExpectations(suite.T())
}

// Test that when we fail to read the db for a relationship, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreRelationshipFailure() {
	secretID := "id1"

	expectedSecret := &v1.Secret{
		Id:        secretID,
		Name:      "secretname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}
	suite.mockSecretStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	expectedErr := fmt.Errorf("failure")
	suite.mockDeploymentStore.On("SearchDeployments", psr).Return(([]*v1.SearchResult)(nil), expectedErr)

	_, actualErr := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(actualErr)

	suite.mockSecretStore.AssertExpectations(suite.T())
}

// Test happy path for searching secrets and relationships
func (suite *SecretServiceTestSuite) TestSearchSecret() {
	expectedReturns := []*v1.ListSecret{
		{Id: "id1"},
	}

	suite.mockSecretStore.On("ListSecrets").Return(expectedReturns, nil)

	_, err := suite.service.ListSecrets((context.Context)(nil), &v1.RawQuery{})
	suite.NoError(err)

	suite.mockSecretStore.AssertExpectations(suite.T())
}

// Test that when searching fails, that error is returned.
func (suite *SecretServiceTestSuite) TestSearchSecretFailure() {
	expectedError := fmt.Errorf("failure")

	suite.mockSecretStore.On("ListSecrets").Return(([]*v1.ListSecret)(nil), expectedError)

	_, actualErr := suite.service.ListSecrets((context.Context)(nil), &v1.RawQuery{})
	suite.True(strings.Contains(actualErr.Error(), expectedError.Error()))

	suite.mockSecretStore.AssertExpectations(suite.T())
}
