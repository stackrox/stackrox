package datastore

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockDataStore is a mock implementation of the DataStore interface.
type MockDataStore struct {
	mock.Mock
}

// SearchAlerts implements a mock version of SearchAlerts
func (m *MockDataStore) SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawAlerts implements a mock version of SearchRawAlerts
func (m *MockDataStore) SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Alert), args.Error(1)
}

// GetAlert is a mock implementation of GetAlert
func (m *MockDataStore) GetAlert(id string) (*v1.Alert, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Alert), args.Bool(1), args.Error(2)
}

// GetAlerts is a mock implementation of GetAlerts
func (m *MockDataStore) GetAlerts(request *v1.ListAlertsRequest) ([]*v1.Alert, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Alert), args.Error(1)
}

// CountAlerts is a mock implementation of CountAlerts
func (m *MockDataStore) CountAlerts() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(2)
}

// AddAlert is a mock implementation of AddAlert
func (m *MockDataStore) AddAlert(alert *v1.Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

// UpdateAlert is a mock implementation of UpdateAlert
func (m *MockDataStore) UpdateAlert(alert *v1.Alert) error {
	args := m.Called(alert)
	return args.Error(0)
}

// RemoveAlert is a mock implementation of RemoveAlert
func (m *MockDataStore) RemoveAlert(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
