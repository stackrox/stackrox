package store

import (
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// GetCluster is a mock implementation of GetCluster
func (m *MockStore) GetCluster(id string) (*v1.Cluster, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.Cluster), args.Bool(1), args.Error(2)
}

// GetClusters is a mock implementation of GetClusters
func (m *MockStore) GetClusters() ([]*v1.Cluster, error) {
	args := m.Called()
	return args.Get(0).([]*v1.Cluster), args.Error(1)
}

// CountClusters is a mock implementation of CountClusters
func (m *MockStore) CountClusters() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// AddCluster is a mock implementation of AddCluster
func (m *MockStore) AddCluster(cluster *v1.Cluster) (string, error) {
	args := m.Called(cluster)
	return args.String(0), args.Error(1)
}

// UpdateCluster is a mock implementation of UpdateCluster
func (m *MockStore) UpdateCluster(cluster *v1.Cluster) error {
	args := m.Called(cluster)
	return args.Error(0)
}

// RemoveCluster is a mock implementation of RemoveCluster
func (m *MockStore) RemoveCluster(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// UpdateClusterContactTime is a mock implementation of UpdateClusterContactTime
func (m *MockStore) UpdateClusterContactTime(id string, t time.Time) error {
	args := m.Called(id, t)
	return args.Error(0)
}
