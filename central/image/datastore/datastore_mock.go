package datastore

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockDataStore is a mock implementation of the DataStore interface.
type MockDataStore struct {
	mock.Mock
}

// SearchImages implements a mock version of SearchImages
func (m *MockDataStore) SearchImages(request *v1.ParsedSearchRequest) ([]*v1.SearchResult, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.SearchResult), args.Error(1)
}

// SearchRawImages implements a mock version of SearchRawImages
func (m *MockDataStore) SearchRawImages(request *v1.ParsedSearchRequest) ([]*v1.Image, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// GetImage is a mock implementation of GetImage
func (m *MockDataStore) GetImage(sha string) (*v1.Image, bool, error) {
	args := m.Called(sha)
	return args.Get(0).(*v1.Image), args.Bool(1), args.Error(2)
}

// GetImages is a mock implementation of GetImages
func (m *MockDataStore) GetImages() ([]*v1.Image, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// CountImages is a mock implementation of CountImages
func (m *MockDataStore) CountImages() (count int, err error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddImage is a mock implementation of AddImage
func (m *MockDataStore) AddImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// UpdateImage is a mock implementation of UpdateImage
func (m *MockDataStore) UpdateImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// RemoveImage is a mock implementation of RemoveImage
func (m *MockDataStore) RemoveImage(sha string) error {
	args := m.Called(sha)
	return args.Error(0)
}
