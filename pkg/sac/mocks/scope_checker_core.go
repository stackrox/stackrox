// Code generated by MockGen. DO NOT EDIT.
// Source: scope_checker_core.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	permissions "github.com/stackrox/rox/pkg/auth/permissions"
	sac "github.com/stackrox/rox/pkg/sac"
	effectiveaccessscope "github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
)

// MockScopeCheckerCore is a mock of ScopeCheckerCore interface.
type MockScopeCheckerCore struct {
	ctrl     *gomock.Controller
	recorder *MockScopeCheckerCoreMockRecorder
}

// MockScopeCheckerCoreMockRecorder is the mock recorder for MockScopeCheckerCore.
type MockScopeCheckerCoreMockRecorder struct {
	mock *MockScopeCheckerCore
}

// NewMockScopeCheckerCore creates a new mock instance.
func NewMockScopeCheckerCore(ctrl *gomock.Controller) *MockScopeCheckerCore {
	mock := &MockScopeCheckerCore{ctrl: ctrl}
	mock.recorder = &MockScopeCheckerCoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockScopeCheckerCore) EXPECT() *MockScopeCheckerCoreMockRecorder {
	return m.recorder
}

// EffectiveAccessScope mocks base method.
func (m *MockScopeCheckerCore) EffectiveAccessScope(resource permissions.ResourceWithAccess) (*effectiveaccessscope.ScopeTree, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EffectiveAccessScope", resource)
	ret0, _ := ret[0].(*effectiveaccessscope.ScopeTree)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EffectiveAccessScope indicates an expected call of EffectiveAccessScope.
func (mr *MockScopeCheckerCoreMockRecorder) EffectiveAccessScope(resource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EffectiveAccessScope", reflect.TypeOf((*MockScopeCheckerCore)(nil).EffectiveAccessScope), resource)
}

// PerformChecks mocks base method.
func (m *MockScopeCheckerCore) PerformChecks(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PerformChecks", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// PerformChecks indicates an expected call of PerformChecks.
func (mr *MockScopeCheckerCoreMockRecorder) PerformChecks(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PerformChecks", reflect.TypeOf((*MockScopeCheckerCore)(nil).PerformChecks), ctx)
}

// SubScopeChecker mocks base method.
func (m *MockScopeCheckerCore) SubScopeChecker(scopeKey sac.ScopeKey) sac.ScopeCheckerCore {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubScopeChecker", scopeKey)
	ret0, _ := ret[0].(sac.ScopeCheckerCore)
	return ret0
}

// SubScopeChecker indicates an expected call of SubScopeChecker.
func (mr *MockScopeCheckerCoreMockRecorder) SubScopeChecker(scopeKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubScopeChecker", reflect.TypeOf((*MockScopeCheckerCore)(nil).SubScopeChecker), scopeKey)
}

// TryAllowed mocks base method.
func (m *MockScopeCheckerCore) TryAllowed() sac.TryAllowedResult {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TryAllowed")
	ret0, _ := ret[0].(sac.TryAllowedResult)
	return ret0
}

// TryAllowed indicates an expected call of TryAllowed.
func (mr *MockScopeCheckerCoreMockRecorder) TryAllowed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TryAllowed", reflect.TypeOf((*MockScopeCheckerCore)(nil).TryAllowed))
}
