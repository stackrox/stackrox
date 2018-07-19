package search

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockSearcher is a mock implementation of the Searcher interface.
type MockSearcher struct {
	mock.Mock
}

// SearchImages implements a mock version of SearchImages
func (m *MockSearcher) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawImages implements a mock version of SearchRawImages
func (m *MockSearcher) SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// SearchListImages implements a mock version of SearchListImages
func (m *MockSearcher) SearchListImages(request *v1.ParsedSearchRequest) ([]*v1.ListImage, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.ListImage), args.Error(1)
}
