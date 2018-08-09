package service

import (
	"context"
	"fmt"
	"testing"

	searchMocks "github.com/stackrox/rox/central/secret/search/mocks"
	storeMocks "github.com/stackrox/rox/central/secret/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestSecretService(t *testing.T) {
	suite.Run(t, new(SecretServiceTestSuite))
}

type SecretServiceTestSuite struct {
	suite.Suite

	mockStore    *storeMocks.Store
	mockSearcher *searchMocks.Searcher

	service Service
}

func (suite *SecretServiceTestSuite) SetupTest() {
	suite.mockStore = &storeMocks.Store{}
	suite.mockSearcher = &searchMocks.Searcher{}

	suite.service = New(suite.mockStore, suite.mockSearcher)
}

// Test happy path for getting secrets and relationships
func (suite *SecretServiceTestSuite) TestGetSecret() {
	secretID := "id1"

	expectedSecret := &v1.Secret{Id: secretID}
	suite.mockStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	expectedRelationship := &v1.SecretRelationship{Id: secretID}
	suite.mockStore.On("GetRelationship", secretID).Return(expectedRelationship, true, nil)

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
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreRelationshipNotExists() {
	secretID := "id1"

	expectedSecret := &v1.Secret{Id: "id1"}
	suite.mockStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	suite.mockStore.On("GetRelationship", secretID).Return((*v1.SecretRelationship)(nil), false, nil)

	_, err := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(err)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}

// Test that when we fail to read the db for a relationship, an error is returned.
func (suite *SecretServiceTestSuite) TestGetSecretsWithStoreRelationshipFailure() {
	secretID := "id1"

	expectedSecret := &v1.Secret{Id: secretID}
	suite.mockStore.On("GetSecret", secretID).Return(expectedSecret, true, nil)

	expectedErr := fmt.Errorf("failure")
	suite.mockStore.On("GetRelationship", secretID).Return((*v1.SecretRelationship)(nil), true, expectedErr)

	_, actualErr := suite.service.GetSecret((context.Context)(nil), &v1.ResourceByID{Id: secretID})
	suite.Error(actualErr)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}

// Test happy path for searching secrets and relationships
func (suite *SecretServiceTestSuite) TestSearchSecret() {
	query := &v1.RawQuery{Query: "derp"}

	expectedReturns := []*v1.SecretAndRelationship{
		{
			Secret: &v1.Secret{Id: "id1"},
		},
	}
	suite.mockSearcher.On("SearchRawSecrets", query).Return(expectedReturns, nil)

	actualReturns, err := suite.service.GetSecrets((context.Context)(nil), query)
	suite.NoError(err)
	suite.Equal(1, len(actualReturns.GetSecretAndRelationships()))
	suite.Equal(expectedReturns, actualReturns.GetSecretAndRelationships())

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}

// Test that when searching fails, that error is returned.
func (suite *SecretServiceTestSuite) TestSearchSecretFailure() {
	query := &v1.RawQuery{Query: "derp"}

	expectedError := fmt.Errorf("failure")
	suite.mockSearcher.On("SearchRawSecrets", query).Return(([]*v1.SecretAndRelationship)(nil), expectedError)

	_, actualErr := suite.service.GetSecrets((context.Context)(nil), query)
	suite.Equal(expectedError, actualErr)

	suite.mockStore.AssertExpectations(suite.T())
	suite.mockSearcher.AssertExpectations(suite.T())
}
