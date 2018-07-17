package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// GetSensorEvent is a mock implementation of GetSensorEvent
func (m *MockStore) GetSensorEvent(id uint64) (*v1.SensorEvent, bool, error) {
	args := m.Called(id)
	return args.Get(0).(*v1.SensorEvent), args.Bool(1), args.Error(2)
}

// GetSensorEventIds is a mock implementation of GetSensorEventIds
func (m *MockStore) GetSensorEventIds(clusterID string) ([]uint64, map[string]uint64, error) {
	args := m.Called(clusterID)
	return args.Get(0).([]uint64), args.Get(1).(map[string]uint64), args.Error(2)
}

// AddSensorEvent is a mock implementation of AddSensorEvent
func (m *MockStore) AddSensorEvent(event *v1.SensorEvent) (uint64, error) {
	args := m.Called(event)
	return args.Get(0).(uint64), args.Error(1)
}

// UpdateSensorEvent is a mock implementation of UpdateSensorEvent
func (m *MockStore) UpdateSensorEvent(id uint64, event *v1.SensorEvent) error {
	args := m.Called(id, event)
	return args.Error(0)
}

// RemoveSensorEvent is a mock implementation of RemoveSensorEvent
func (m *MockStore) RemoveSensorEvent(id uint64) error {
	args := m.Called(id)
	return args.Error(0)
}
