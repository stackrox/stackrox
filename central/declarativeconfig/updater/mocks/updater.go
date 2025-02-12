// Code generated by MockGen. DO NOT EDIT.
// Source: updater.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/updater.go -source updater.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	protocompat "github.com/stackrox/rox/pkg/protocompat"
	gomock "go.uber.org/mock/gomock"
)

// MockResourceUpdater is a mock of ResourceUpdater interface.
type MockResourceUpdater struct {
	ctrl     *gomock.Controller
	recorder *MockResourceUpdaterMockRecorder
	isgomock struct{}
}

// MockResourceUpdaterMockRecorder is the mock recorder for MockResourceUpdater.
type MockResourceUpdaterMockRecorder struct {
	mock *MockResourceUpdater
}

// NewMockResourceUpdater creates a new mock instance.
func NewMockResourceUpdater(ctrl *gomock.Controller) *MockResourceUpdater {
	mock := &MockResourceUpdater{ctrl: ctrl}
	mock.recorder = &MockResourceUpdaterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceUpdater) EXPECT() *MockResourceUpdaterMockRecorder {
	return m.recorder
}

// DeleteResources mocks base method.
func (m *MockResourceUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx}
	for _, a := range resourceIDsToSkip {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeleteResources", varargs...)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteResources indicates an expected call of DeleteResources.
func (mr *MockResourceUpdaterMockRecorder) DeleteResources(ctx any, resourceIDsToSkip ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx}, resourceIDsToSkip...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteResources", reflect.TypeOf((*MockResourceUpdater)(nil).DeleteResources), varargs...)
}

// Upsert mocks base method.
func (m_2 *MockResourceUpdater) Upsert(ctx context.Context, m protocompat.Message) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "Upsert", ctx, m)
	ret0, _ := ret[0].(error)
	return ret0
}

// Upsert indicates an expected call of Upsert.
func (mr *MockResourceUpdaterMockRecorder) Upsert(ctx, m any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upsert", reflect.TypeOf((*MockResourceUpdater)(nil).Upsert), ctx, m)
}
