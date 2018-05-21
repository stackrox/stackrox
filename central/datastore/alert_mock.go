package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// MockAlertDataStore is a mock implementation of the AlertDataStore interface.
type MockAlertDataStore struct {
	db.MockAlertStorage
}

// SearchAlerts implements a mock version of SearchAlerts
func (m *MockAlertDataStore) SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawAlerts implements a mock version of SearchRawAlerts
func (m *MockAlertDataStore) SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Alert), args.Error(1)
}
