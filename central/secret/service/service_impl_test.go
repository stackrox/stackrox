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
	"github.com/stackrox/rox/pkg/protoassert"
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

	expectedSecret := &storage.Secret{}
	expectedSecret.SetId(secretID)
	expectedSecret.SetName("secretname")
	expectedSecret.SetClusterId("cluster")
	expectedSecret.SetNamespace("namespace")
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	sr := &v1.SearchResult{}
	sr.SetId("d1")
	sr.SetName("deployment1")
	results := []*v1.SearchResult{
		sr,
	}

	suite.mockDeploymentStore.EXPECT().SearchDeployments(gomock.Any(), psr).Return(results, nil)

	sdr := &storage.SecretDeploymentRelationship{}
	sdr.SetId("d1")
	sdr.SetName("deployment1")
	expectedRelationship := &storage.SecretRelationship{}
	expectedRelationship.SetDeploymentRelationships([]*storage.SecretDeploymentRelationship{
		sdr,
	})

	rbid := &v1.ResourceByID{}
	rbid.SetId(secretID)
	actualSecretAndRelationship, err := suite.service.GetSecret((context.Context)(nil), rbid)
	suite.NoError(err)
	protoassert.Equal(suite.T(), expectedSecret, actualSecretAndRelationship)
	protoassert.Equal(suite.T(), expectedRelationship, actualSecretAndRelationship.GetRelationship())
}

// Test that when we fail to find a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretNotExists() {
	secretID := "id1"

	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return((*storage.Secret)(nil), false, nil)

	rbid := &v1.ResourceByID{}
	rbid.SetId(secretID)
	_, err := suite.service.GetSecret((context.Context)(nil), rbid)
	suite.Error(err)
}

// Test that when we fail to read the db for a secret, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreSecretFailure() {
	secretID := "id1"

	expectedErr := errors.New("failure")
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return((*storage.Secret)(nil), true, expectedErr)

	rbid := &v1.ResourceByID{}
	rbid.SetId(secretID)
	_, actualErr := suite.service.GetSecret((context.Context)(nil), rbid)
	suite.Error(actualErr)
}

// Test that when we fail to find a relationship, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithNoRelationship() {
	secretID := "id1"

	expectedSecret := &storage.Secret{}
	expectedSecret.SetId(secretID)
	expectedSecret.SetName("secretname")
	expectedSecret.SetClusterId("cluster")
	expectedSecret.SetNamespace("namespace")
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	suite.mockDeploymentStore.EXPECT().SearchDeployments(gomock.Any(), psr).Return([]*v1.SearchResult{}, nil)

	rbid := &v1.ResourceByID{}
	rbid.SetId(secretID)
	_, err := suite.service.GetSecret((context.Context)(nil), rbid)
	suite.NoError(err)
}

// Test that when we fail to read the db for a relationship, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreRelationshipFailure() {
	secretID := "id1"

	expectedSecret := &storage.Secret{}
	expectedSecret.SetId(secretID)
	expectedSecret.SetName("secretname")
	expectedSecret.SetClusterId("cluster")
	expectedSecret.SetNamespace("namespace")
	suite.mockSecretStore.EXPECT().GetSecret(gomock.Any(), secretID).Return(expectedSecret, true, nil)

	psr := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, "cluster").
		AddExactMatches(search.Namespace, "namespace").
		AddExactMatches(search.SecretName, "secretname").
		ProtoQuery()

	expectedErr := errors.New("failure")
	suite.mockDeploymentStore.EXPECT().SearchDeployments(gomock.Any(), psr).Return(([]*v1.SearchResult)(nil), expectedErr)

	rbid := &v1.ResourceByID{}
	rbid.SetId(secretID)
	_, actualErr := suite.service.GetSecret((context.Context)(nil), rbid)
	suite.Error(actualErr)
}

// Test happy path for searching secrets and relationships
func (suite *SecretServiceTestSuite) TestSearchSecret() {
	ls := &storage.ListSecret{}
	ls.SetId("id1")
	expectedReturns := []*storage.ListSecret{
		ls,
	}

	emptyWithPag := search.EmptyQuery()
	qp := &v1.QueryPagination{}
	qp.SetLimit(maxSecretsReturned)
	emptyWithPag.SetPagination(qp)
	suite.mockSecretStore.EXPECT().SearchListSecrets(gomock.Any(), emptyWithPag).Return(expectedReturns, nil)

	_, err := suite.service.ListSecrets((context.Context)(nil), &v1.RawQuery{})
	suite.NoError(err)
}

// Test that when searching fails, that error is returned.
func (suite *SecretServiceTestSuite) TestSearchSecretFailure() {
	expectedError := errors.New("failure")

	emptyWithPag := search.EmptyQuery()
	qp := &v1.QueryPagination{}
	qp.SetLimit(maxSecretsReturned)
	emptyWithPag.SetPagination(qp)
	suite.mockSecretStore.EXPECT().SearchListSecrets(gomock.Any(), emptyWithPag).Return(([]*storage.ListSecret)(nil), expectedError)

	_, actualErr := suite.service.ListSecrets((context.Context)(nil), &v1.RawQuery{})
	suite.True(strings.Contains(actualErr.Error(), expectedError.Error()))
}
