package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// GetNotifier is a mock implementation of GetNotifier
func (m *MockStore) GetNotifier(id string) (*v1.Notifier, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Notifier), args.Bool(1), args.Error(2)
}

// GetNotifiers is a mock implementation of GetNotifiers
func (m *MockStore) GetNotifiers(request *v1.GetNotifiersRequest) ([]*v1.Notifier, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Notifier), args.Error(1)
}

// AddNotifier is a mock implementation of AddNotifier
func (m *MockStore) AddNotifier(notifier *v1.Notifier) (string, error) {
	args := m.Called(notifier)
	return args.String(0), args.Error(1)
}

// UpdateNotifier is a mock implementation of UpdateNotifier
func (m *MockStore) UpdateNotifier(notifier *v1.Notifier) error {
	args := m.Called(notifier)
	return args.Error(0)
}

// RemoveNotifier is a mock implementation of RemoveNotifier
func (m *MockStore) RemoveNotifier(id string) error {
	args := m.Called(id)
	return args.Error(0)
}
