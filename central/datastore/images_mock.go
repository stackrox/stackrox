package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// MockImageDataStore is a mock implementation of the ImageDataStore interface.
type MockImageDataStore struct {
	db.MockImageStorage
}

// SearchImages implements a mock version of SearchImages
func (m *MockImageDataStore) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawImages implements a mock version of SearchRawImages
func (m *MockImageDataStore) SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Image), args.Error(1)
}
