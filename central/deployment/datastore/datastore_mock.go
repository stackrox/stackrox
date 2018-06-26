package datastore

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockDataStore is a mock implementation of the DataStore interface.
type MockDataStore struct {
	mock.Mock
}

// SearchDeployments implements a mock version of SearchDeployments
func (m *MockDataStore) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawDeployments implements a mock version of SearchRawDeployments
func (m *MockDataStore) SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}

// GetDeployment is a mock implementation of GetDeployment
func (m *MockDataStore) GetDeployment(id string) (*v1.Deployment, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Deployment), args.Bool(1), args.Error(2)
}

// GetDeployments is a mock implementation of GetDeployments
func (m *MockDataStore) GetDeployments() ([]*v1.Deployment, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}

// CountDeployments is a mock implementation of CountDeployments
func (m *MockDataStore) CountDeployments() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddDeployment is a mock implementation of AddDeployment
func (m *MockDataStore) AddDeployment(deployment *v1.Deployment) error {
	args := m.Called(deployment)
	return args.Error(0)
}

// UpdateDeployment is a mock implementation of UpdateDeployment
func (m *MockDataStore) UpdateDeployment(deployment *v1.Deployment) error {
	args := m.Called(deployment)
	return args.Error(0)
}

// RemoveDeployment is a mock implementation of RemoveDeployment
func (m *MockDataStore) RemoveDeployment(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// GetTombstonedDeployments is a mock implementation of GetTombstonedDeployments
func (m *MockDataStore) GetTombstonedDeployments() ([]*v1.Deployment, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}
