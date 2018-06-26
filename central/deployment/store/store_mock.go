package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// GetDeployment is a mock implementation of GetDeployment
func (m *MockStore) GetDeployment(id string) (*v1.Deployment, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Deployment), args.Bool(1), args.Error(2)
}

// GetDeployments is a mock implementation of GetDeployments
func (m *MockStore) GetDeployments() ([]*v1.Deployment, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}

// CountDeployments is a mock implementation of CountDeployments
func (m *MockStore) CountDeployments() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddDeployment is a mock implementation of AddDeployment
func (m *MockStore) AddDeployment(deployment *v1.Deployment) error {
	args := m.Called(deployment)
	return args.Error(0)
}

// UpdateDeployment is a mock implementation of UpdateDeployment
func (m *MockStore) UpdateDeployment(deployment *v1.Deployment) error {
	args := m.Called(deployment)
	return args.Error(0)
}

// RemoveDeployment is a mock implementation of RemoveDeployment
func (m *MockStore) RemoveDeployment(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// GetTombstonedDeployments is a mock implementation of GetTombstonedDeployments
func (m *MockStore) GetTombstonedDeployments() ([]*v1.Deployment, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}
