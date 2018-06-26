package search

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockSearcher is a mock implementation of the Searcher interface.
type MockSearcher struct {
	mock.Mock
}

// SearchDeployments implements a mock version of SearchDeployments
func (m *MockSearcher) SearchDeployments(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawDeployments implements a mock version of SearchRawDeployments
func (m *MockSearcher) SearchRawDeployments(request *v1.ParsedSearchRequest) ([]*v1.Deployment, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Deployment), args.Error(1)
}
