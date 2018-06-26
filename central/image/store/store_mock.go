package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the ImageStorage interface.
type MockStore struct {
	mock.Mock
}

// GetImage is a mock implementation of GetImage
func (m *MockStore) GetImage(sha string) (*v1.Image, bool, error) {
	args := m.Called(sha)
	return args.Get(0).(*v1.Image), args.Bool(1), args.Error(2)
}

// GetImages is a mock implementation of GetImages
func (m *MockStore) GetImages() ([]*v1.Image, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// CountImages is a mock implementation of CountImages
func (m *MockStore) CountImages() (count int, err error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddImage is a mock implementation of AddImage
func (m *MockStore) AddImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// UpdateImage is a mock implementation of UpdateImage
func (m *MockStore) UpdateImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// RemoveImage is a mock implementation of RemoveImage
func (m *MockStore) RemoveImage(sha string) error {
	args := m.Called(sha)
	return args.Error(0)
}
