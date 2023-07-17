package service

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	datastoreMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestSecretService(t *testing.T) {
	suite.Run(t, new(SecretServiceTestSuite))
}

type SecretServiceTestSuite struct {
	suite.Suite

	mockSecretStore     *datastoreMocks.MockDataStore
	mockDeploymentStore *deploymentMocks.MockDataStore

	service Service

	mockCtrl *gomock.Controller
}

func (suite *SecretServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockSecretStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockDeploymentStore = deploymentMocks.NewMockDataStore(suite.mockCtrl)

	suite.service = New(suite.mockSecretStore, suite.mockDeploymentStore)
}

// Test happy path for getting secrets and relationships
func (suite *SecretServiceTestSuite) TestGetSecret() {
	secretID := "id1"

	expectedSecret := &storage.Secret{
		Id:        secretID,
		Name:      "secretname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return(expectedSecret, true, nil)

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

	suite.mockDeploymentStore.EXPECT().SearchDeployments(gomock.Any(), psr).Return(results, nil)

	expectedRelationship := &storage.SecretRelationship{
		DeploymentRelationships: []*storage.SecretDeploymentRelationship{
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
}

// Test that when we fail to find a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretNotExists() {
	secretID := "id1"

	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return((*storage.Secret)(nil), false, nil)

	_, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(err)
}

// Test that when we fail to read the db for a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretFailure() {
	secretID := "id1"

	expectedErr := errors.New("failure")
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return((*storage.Secret)(nil), true, expectedErr)

	_, actualErr := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(actualErr)
}

// Test that when we fail to find a relationship, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithNoRelationship() {
	secretID := "id1"

	expectedSecret := &storage.Secret{
		Id:        secretID,
		Name:      "secretname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	suite.mockDeploymentStore.EXPECT().SearchDeployments(gomock.Any(), psr).Return([]*v1.SearchResult{}, nil)

	_, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.NoError(err)
}

// Test that when we fail to read the db for a relationship, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreRelationshipFailure() {
	secretID := "id1"

	expectedSecret := &storage.Secret{
		Id:        secretID,
		Name:      "secretname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	expectedErr := errors.New("failure")
	suite.mockDeploymentStore.EXPECT().SearchDeployments(gomock.Any(), psr).Return(([]*v1.SearchResult)(nil), expectedErr)

	_, actualErr := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(actualErr)
}

// Test happy path for searching secrets and relationships
func (suite *SecretServiceTestSuite) TestSearchSecret() {
	expectedReturns := []*storage.ListSecret{
		{Id: "id1"},
	}

	emptyWithPag := search.EmptyQuery()
	emptyWithPag.Pagination = &v1.QueryPagination{
		Limit: maxSecretsReturned,
	}
	suite.mockSecretStore.EXPECT().SearchListSecrets(gomock.Any(), emptyWithPag).Return(expectedReturns, nil)

	_, err := suite.service.ListSecrets((context.Context)(nil), &v1.RawQuery{})
	suite.NoError(err)
}

// Test that when searching fails, that error is returned.
func (suite *SecretServiceTestSuite) TestSearchSecretFailure() {
	expectedError := errors.New("failure")

	emptyWithPag := search.EmptyQuery()
	emptyWithPag.Pagination = &v1.QueryPagination{
		Limit: maxSecretsReturned,
	}
	suite.mockSecretStore.EXPECT().SearchListSecrets(gomock.Any(), emptyWithPag).Return(([]*storage.ListSecret)(nil), expectedError)

	_, actualErr := suite.service.ListSecrets((context.Context)(nil), &v1.RawQuery{})
	suite.True(strings.Contains(actualErr.Error(), expectedError.Error()))
}
