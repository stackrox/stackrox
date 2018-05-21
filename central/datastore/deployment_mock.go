package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// MockDeploymentDataStore is a mock implementation of the DeploymentDataStore interface.
type MockDeploymentDataStore struct {
	db.MockDeploymentStorage
}

// SearchDeployments implements a mock version of SearchDeployments
func (m *MockDeploymentDataStore) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawDeployments implements a mock version of SearchRawDeployments
func (m *MockDeploymentDataStore) SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}
