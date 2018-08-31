package service

import (
	"context"
	"fmt"
	"testing"

	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	searchMocks "github.com/stackrox/rox/central/secret/search/mocks"
	storeMocks "github.com/stackrox/rox/central/secret/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestSecretService(t *testing.T) {
	suite.Run(t, new(SecretServiceTestSuite))
}

type SecretServiceTestSuite struct {
	suite.Suite

	mockStore     *storeMocks.Store
	mockSearcher  *searchMocks.Searcher
	mockDatastore *deploymentMocks.DataStore

	service Service
}

func (suite *SecretServiceTestSuite) SetupTest() {
	suite.mockStore = &storeMocks.Store{}
	suite.mockSearcher = &searchMocks.Searcher{}
	suite.mockDatastore = &deploymentMocks.DataStore{}

	suite.service = New(suite.mockStore, suite.mockSearcher, suite.mockDatastore)
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
	suite.mockStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddStrings(search.ClusterID, "cluster").
		AddStrings(search.Namespace, "namespace").
		AddStrings(search.SecretName, "secretname").
		ProtoQuery()

	results := []*v1.SearchResult{
		{
			Id:   "d1",
			Name: "deployment1",
		},
	}

	suite.mockDatastore.On("SearchDeployments", psr).Return(results, nil)

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
	suite.Equal(expectedSecret, actualSecretAndRelationship.GetSecret())
	suite.Equal(expectedRelationship, actualSecretAndRelationship.GetRelationship())

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}

// Test that when we fail to find a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretNotExists() {
	secretID := "id1"

	suite.mockStore.On("GetSecret", secretID).Return((*v1.Secret)(nil), false, nil)

	_, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(err)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}

// Test that when we fail to read the db for a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretFailure() {
	secretID := "id1"

	expectedErr := fmt.Errorf("failure")
	suite.mockStore.On("GetSecret", secretID).Return((*v1.Secret)(nil), true, expectedErr)

	_, actualErr := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(actualErr)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
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
	suite.mockStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddStrings(search.ClusterID, "cluster").
		AddStrings(search.Namespace, "namespace").
		AddStrings(search.SecretName, "secretname").
		ProtoQuery()

	suite.mockDatastore.On("SearchDeployments", psr).Return([]*v1.SearchResult{}, nil)

	_, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.NoError(err)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
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
	suite.mockStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddStrings(search.ClusterID, "cluster").
		AddStrings(search.Namespace, "namespace").
		AddStrings(search.SecretName, "secretname").
		ProtoQuery()

	expectedErr := fmt.Errorf("failure")
	suite.mockDatastore.On("SearchDeployments", psr).Return(([]*v1.SearchResult)(nil), expectedErr)

	_, actualErr := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(actualErr)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}

// Test happy path for searching secrets and relationships
func (suite *SecretServiceTestSuite) TestSearchSecret() {
	query := &v1.RawQuery{Query: "derp"}

	expectedReturns := []*v1.Secret{
		{Id: "id1"},
	}
	suite.mockSearcher.On("SearchRawSecrets", query).Return(expectedReturns, nil)

	_, err := suite.service.GetSecrets((context.Context)(nil), query)
	suite.NoError(err)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}

// Test that when searching fails, that error is returned.
func (suite *SecretServiceTestSuite) TestSearchSecretFailure() {
	query := &v1.RawQuery{Query: "derp"}

	expectedError := fmt.Errorf("failure")
	suite.mockSearcher.On("SearchRawSecrets", query).Return(([]*v1.Secret)(nil), expectedError)

	_, actualErr := suite.service.GetSecrets((context.Context)(nil), query)
	suite.Equal(expectedError, actualErr)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}
