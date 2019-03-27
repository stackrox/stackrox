package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	datastoreMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestServiceAccountService(t *testing.T) {
	suite.Run(t, new(ServiceAccountServiceTestSuite))
}

type ServiceAccountServiceTestSuite struct {
	suite.Suite

	mockServiceAccountStore *datastoreMocks.MockDataStore

	service Service

	mockCtrl *gomock.Controller
}

func (suite *ServiceAccountServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockServiceAccountStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)

	suite.service = New(suite.mockServiceAccountStore)
}

// Test happy path for getting service accounts
func (suite *ServiceAccountServiceTestSuite) TestGetServiceAccount() {
	saID := "id1"

	expectedSA := &storage.ServiceAccount{
		Id:        saID,
		Name:      "serviceaccountname",
		ClusterId: "cluster",
		Namespace: "namespace",
	}
	suite.mockServiceAccountStore.EXPECT().GetServiceAccount(saID).Return(expectedSA, true, nil)

	sa, err := suite.service.GetServiceAccount((context.Context)(nil), &v1.ResourceByID{Id: saID})
	suite.NoError(err)
	suite.Equal(expectedSA, sa.ServiceAccount)
}

// Test that when we fail to find a service account, an error is returned.
func (suite *ServiceAccountServiceTestSuite) TestGetSAWithStoreSANotExists() {
	saID := "id1"

	suite.mockServiceAccountStore.EXPECT().GetServiceAccount(saID).Return((*storage.ServiceAccount)(nil), false, nil)

	_, err := suite.service.GetServiceAccount((context.Context)(nil), &v1.ResourceByID{Id: saID})
	suite.Error(err)
}

// Test that when we fail to read the db for a secret, an error is returned.
func (suite *ServiceAccountServiceTestSuite) TestGetSAWithStoreSAFailure() {
	saID := "id1"

	expectedErr := fmt.Errorf("failure")
	suite.mockServiceAccountStore.EXPECT().GetServiceAccount(saID).Return((*storage.ServiceAccount)(nil), true, expectedErr)

	_, actualErr := suite.service.GetServiceAccount((context.Context)(nil), &v1.ResourceByID{Id: saID})
	suite.Error(actualErr)
}

// Test happy path for searching secrets and relationships
func (suite *ServiceAccountServiceTestSuite) TestSearchServiceAccount() {
	expectedReturns := []*storage.ServiceAccount{
		{Id: "id1"},
	}

	suite.mockServiceAccountStore.EXPECT().ListServiceAccounts().Return(expectedReturns, nil)

	_, err := suite.service.ListServiceAccounts((context.Context)(nil), &v1.RawQuery{})
	suite.NoError(err)
}

// Test that when searching fails, that error is returned.
func (suite *ServiceAccountServiceTestSuite) TestSearchServiceAccountFailure() {
	expectedError := fmt.Errorf("failure")

	suite.mockServiceAccountStore.EXPECT().ListServiceAccounts().Return(nil, expectedError)

	_, actualErr := suite.service.ListServiceAccounts((context.Context)(nil), &v1.RawQuery{})
	suite.True(strings.Contains(actualErr.Error(), expectedError.Error()))
}
