package store

import (
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the Store interface.
type MockStore struct {
	mock.Mock
}

// GetLogs is a mock implementation of GetLogs
func (m *MockStore) GetLogs() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// CountLogs is a mock implementation of CountLogs
func (m *MockStore) CountLogs() (count int, err error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// GetLogsRange is a mock implementation of GetLogsRange
func (m *MockStore) GetLogsRange() (start int64, end int64, err error) {
	args := m.Called()
	return args.Get(0).(int64), args.Get(1).(int64), args.Error(2)
}

// AddLog is a mock implementation of AddLog
func (m *MockStore) AddLog(log string) error {
	args := m.Called(log)
	return args.Error(0)
}

// RemoveLogs is a mock implementation of RemoveLogs
func (m *MockStore) RemoveLogs(from, to int64) error {
	args := m.Called(from, to)
	return args.Error(0)
}
