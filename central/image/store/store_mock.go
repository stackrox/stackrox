package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the ImageStorage interface.
type MockStore struct {
	mock.Mock
}

// ListImage is a mock implementation of ListImage.
func (m *MockStore) ListImage(sha string) (*v1.ListImage, bool, error) {
	args := m.Called()
	return args.Get(0).(*v1.ListImage), args.Bool(1), args.Error(2)
}

// ListImages is a mock implementation of ListImages.
func (m *MockStore) ListImages() ([]*v1.ListImage, error) {
	args := m.Called()
	return args.Get(0).([]*v1.ListImage), args.Error(1)
}

// GetImages is a mock implementation of GetImages.
func (m *MockStore) GetImages() ([]*v1.Image, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// CountImages is a mock implementation of CountImages.
func (m *MockStore) CountImages() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// GetImage is a mock implementation of GetImage.
func (m *MockStore) GetImage(sha string) (*v1.Image, bool, error) {
	args := m.Called(sha)
	return args.Get(0).(*v1.Image), args.Bool(1), args.Error(2)
}

// GetImagesBatch is a mock implementation of GetImagesBatch.
func (m *MockStore) GetImagesBatch(shas []string) ([]*v1.Image, error) {
	args := m.Called(shas)
	return args.Get(0).([]*v1.Image), args.Error(1)
}

// UpsertImage is a mock implementation of UpsertImage.
func (m *MockStore) UpsertImage(image *v1.Image) error {
	args := m.Called(image)
	return args.Error(0)
}

// DeleteImage is a mock implementation of DeleteImage.
func (m *MockStore) DeleteImage(sha string) error {
	args := m.Called(sha)
	return args.Error(0)
}

// GetRegistrySha is a mock implementation of GetRegistrySha.
func (m *MockStore) GetRegistrySha(orchSha string) (string, bool, error) {
	args := m.Called(orchSha)
	return args.String(0), args.Bool(1), args.Error(2)
}

// UpsertRegistrySha is a mock implementation of UpsertRegistrySha.
func (m *MockStore) UpsertRegistrySha(orchSha string, regSha string) error {
	args := m.Called(orchSha, regSha)
	return args.Error(0)
}

// DeleteRegistrySha is a mock implementation of DeleteRegistrySha.
func (m *MockStore) DeleteRegistrySha(orchSha string) error {
	args := m.Called(orchSha)
	return args.Error(0)
}
