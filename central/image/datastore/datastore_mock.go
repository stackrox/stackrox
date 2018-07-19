package datastore

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockDataStore is a mock implementation of the DataStore interface.
type MockDataStore struct {
	mock.Mock
}

// SearchListImages implements a mock version of SearchListImages
func (m *MockDataStore) SearchListImages(request *v1.ParsedSearchRequest) ([]*v1.ListImage, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.ListImage), args.Error(1)
}

// ListImage implements a mock version of ListImage
func (m *MockDataStore) ListImage(id string) (*v1.ListImage, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.ListImage), args.Bool(1), args.Error(2)
}

// ListImages implements a mock version of ListImages
func (m *MockDataStore) ListImages() ([]*v1.ListImage, error) {
	args := m.Called()
	return args.Get(0).([]*v1.ListImage), args.Error(1)
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

// GetImages is a mock implementation of GetImages
func (m *MockDataStore) GetImages() ([]*v1.Image, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// CountImages is a mock implementation of CountImages
func (m *MockDataStore) CountImages() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// GetImage is a mock implementation of GetImage
func (m *MockDataStore) GetImage(sha string) (*v1.Image, bool, error) {
	args := m.Called(sha)
	return args.Get(0).(*v1.Image), args.Bool(1), args.Error(2)
}

// GetImagesBatch is a mock implementation of GetImagesBatch
func (m *MockDataStore) GetImagesBatch(shas []string) ([]*v1.Image, error) {
	args := m.Called(shas)
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// UpsertDedupeImage is a mock implementation of UpsertDedupeImage
func (m *MockDataStore) UpsertDedupeImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}
