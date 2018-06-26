package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of alerts' Store interface.
type MockStore struct {
	mock.Mock
}

// GetAlert is a mock implementation of GetAlert
func (m *MockStore) GetAlert(id string) (*v1.Alert, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Alert), args.Bool(1), args.Error(2)
}

// GetAlerts is a mock implementation of GetAlerts
func (m *MockStore) GetAlerts(request *v1.ListAlertsRequest) ([]*v1.Alert, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Alert), args.Error(1)
}

// CountAlerts is a mock implementation of CountAlerts
func (m *MockStore) CountAlerts() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(2)
}

// AddAlert is a mock implementation of AddAlert
func (m *MockStore) AddAlert(alert *v1.Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

// UpdateAlert is a mock implementation of UpdateAlert
func (m *MockStore) UpdateAlert(alert *v1.Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

// RemoveAlert is a mock implementation of RemoveAlert
func (m *MockStore) RemoveAlert(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
