package search

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockSearcher is a mock implementation of the Searcher interface.
type MockSearcher struct {
	mock.Mock
}

// SearchAlerts implements a mock version of SearchAlerts
func (m *MockSearcher) SearchAlerts(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawAlerts implements a mock version of SearchRawAlerts
func (m *MockSearcher) SearchRawAlerts(request *v1.ParsedSearchRequest) ([]*v1.Alert, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Alert), args.Error(1)
}
