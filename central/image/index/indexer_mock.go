package index

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/stretchr/testify/mock"
)

// MockIndexer is a mock implementation of the Indexer interface.
type MockIndexer struct {
	mock.Mock
}

// AddImage is a mock implementation of AddImage.
func (m *MockIndexer) AddImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// AddImages is a mock implementation of AddImages.
func (m *MockIndexer) AddImages(imageList []*v1.Image) error {
	args := m.Called(imageList)
	return args.Error(0)
}

// DeleteImage is a mock implementation of DeleteImage.
func (m *MockIndexer) DeleteImage(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// SearchImages is a mock implementation of SearchImages.
func (m *MockIndexer) SearchImages(request *v1.ParsedSearchRequest) ([]search.Result, error) {
	args := m.Called(request)
	return args.Get(0).([]search.Result), args.Error(1)
}
