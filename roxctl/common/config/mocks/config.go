// Code generated by MockGen. DO NOT EDIT.
// Source: config.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/config.go -source config.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	config "github.com/stackrox/rox/roxctl/common/config"
	gomock "go.uber.org/mock/gomock"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
	isgomock struct{}
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockStore) Read() (*config.RoxctlConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read")
	ret0, _ := ret[0].(*config.RoxctlConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockStoreMockRecorder) Read() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockStore)(nil).Read))
}

// Write mocks base method.
func (m *MockStore) Write(cfg *config.RoxctlConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Write", cfg)
	ret0, _ := ret[0].(error)
	return ret0
}

// Write indicates an expected call of Write.
func (mr *MockStoreMockRecorder) Write(cfg any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockStore)(nil).Write), cfg)
}
