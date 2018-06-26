package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// GetDeploymentEvent is a mock implementation of GetDeploymentEvent
func (m *MockStore) GetDeploymentEvent(id uint64) (*v1.DeploymentEvent, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.DeploymentEvent), args.Bool(1), args.Error(2)
}

// GetDeploymentEventIds is a mock implementation of GetDeploymentEvents
func (m *MockStore) GetDeploymentEventIds(clusterID string) ([]uint64, map[string]uint64, error) {
	args := m.Called(clusterID)
	return args.Get(0).([]uint64), args.Get(1).(map[string]uint64), args.Error(2)
}

// AddDeploymentEvent is a mock implementation of AddDeploymentEvent
func (m *MockStore) AddDeploymentEvent(deployment *v1.DeploymentEvent) (uint64, error) {
	args := m.Called(deployment)
	return args.Get(0).(uint64), args.Error(1)
}

// UpdateDeploymentEvent is a mock implementation of UpdateDeploymentEvent
func (m *MockStore) UpdateDeploymentEvent(id uint64, deployment *v1.DeploymentEvent) error {
	args := m.Called(id, deployment)
	return args.Error(0)
}

// RemoveDeploymentEvent is a mock implementation of RemoveDeploymentEvent
func (m *MockStore) RemoveDeploymentEvent(id uint64) error {
	args := m.Called(id)
	return args.Error(0)
}
