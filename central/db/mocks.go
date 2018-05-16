package db

import (
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockNotifierStorage is a mock implementation of the NotifierStorage interface.
type MockNotifierStorage struct {
	mock.Mock
}

// GetNotifier is a mock implementation of GetNotifier
func (m *MockNotifierStorage) GetNotifier(id string) (*v1.Notifier, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Notifier), args.Bool(1), args.Error(2)
}

// GetNotifiers is a mock implementation of GetNotifiers
func (m *MockNotifierStorage) GetNotifiers(request *v1.GetNotifiersRequest) ([]*v1.Notifier, error) {
	args := m.Called(request)
	return args.Get(0).([]*v1.Notifier), args.Error(1)
}

// AddNotifier is a mock implementation of AddNotifier
func (m *MockNotifierStorage) AddNotifier(notifier *v1.Notifier) (string, error) {
	args := m.Called(notifier)
	return args.String(0), args.Error(1)
}

// UpdateNotifier is a mock implementation of UpdateNotifier
func (m *MockNotifierStorage) UpdateNotifier(notifier *v1.Notifier) error {
	args := m.Called(notifier)
	return args.Error(0)
}

// RemoveNotifier is a mock implementation of RemoveNotifier
func (m *MockNotifierStorage) RemoveNotifier(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockClusterStorage is a mock implementation of the ClusterStorage interface.
type MockClusterStorage struct {
	mock.Mock
}

// GetCluster is a mock implementation of GetCluster
func (m *MockClusterStorage) GetCluster(id string) (*v1.Cluster, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Cluster), args.Bool(1), args.Error(2)
}

// GetClusters is a mock implementation of GetClusters
func (m *MockClusterStorage) GetClusters() ([]*v1.Cluster, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Cluster), args.Error(1)
}

// CountClusters is a mock implementation of CountClusters
func (m *MockClusterStorage) CountClusters() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddCluster is a mock implementation of AddCluster
func (m *MockClusterStorage) AddCluster(cluster *v1.Cluster) (string, error) {
	args := m.Called(cluster)
	return args.String(0), args.Error(1)
}

// UpdateCluster is a mock implementation of UpdateCluster
func (m *MockClusterStorage) UpdateCluster(cluster *v1.Cluster) error {
	args := m.Called(cluster)
	return args.Error(0)
}

// RemoveCluster is a mock implementation of RemoveCluster
func (m *MockClusterStorage) RemoveCluster(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// UpdateClusterContactTime is a mock implementation of UpdateClusterContactTime
func (m *MockClusterStorage) UpdateClusterContactTime(id string, t time.Time) error {
	args := m.Called(id, t)
	return args.Error(0)
}
