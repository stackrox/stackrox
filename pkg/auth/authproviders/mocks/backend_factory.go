// Code generated by MockGen. DO NOT EDIT.
// Source: backend_factory.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/backend_factory.go -source backend_factory.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	http "net/http"
	reflect "reflect"

	authproviders "github.com/stackrox/rox/pkg/auth/authproviders"
	gomock "go.uber.org/mock/gomock"
)

// MockBackendFactory is a mock of BackendFactory interface.
type MockBackendFactory struct {
	ctrl     *gomock.Controller
	recorder *MockBackendFactoryMockRecorder
}

// MockBackendFactoryMockRecorder is the mock recorder for MockBackendFactory.
type MockBackendFactoryMockRecorder struct {
	mock *MockBackendFactory
}

// NewMockBackendFactory creates a new mock instance.
func NewMockBackendFactory(ctrl *gomock.Controller) *MockBackendFactory {
	mock := &MockBackendFactory{ctrl: ctrl}
	mock.recorder = &MockBackendFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackendFactory) EXPECT() *MockBackendFactoryMockRecorder {
	return m.recorder
}

// CreateBackend mocks base method.
func (m *MockBackendFactory) CreateBackend(ctx context.Context, id string, uiEndpoints []string, config, mappings map[string]string) (authproviders.Backend, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateBackend", ctx, id, uiEndpoints, config, mappings)
	ret0, _ := ret[0].(authproviders.Backend)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateBackend indicates an expected call of CreateBackend.
func (mr *MockBackendFactoryMockRecorder) CreateBackend(ctx, id, uiEndpoints, config, mappings any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateBackend", reflect.TypeOf((*MockBackendFactory)(nil).CreateBackend), ctx, id, uiEndpoints, config, mappings)
}

// GetSuggestedAttributes mocks base method.
func (m *MockBackendFactory) GetSuggestedAttributes() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSuggestedAttributes")
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetSuggestedAttributes indicates an expected call of GetSuggestedAttributes.
func (mr *MockBackendFactoryMockRecorder) GetSuggestedAttributes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSuggestedAttributes", reflect.TypeOf((*MockBackendFactory)(nil).GetSuggestedAttributes))
}

// MergeConfig mocks base method.
func (m *MockBackendFactory) MergeConfig(newCfg, oldCfg map[string]string) map[string]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MergeConfig", newCfg, oldCfg)
	ret0, _ := ret[0].(map[string]string)
	return ret0
}

// MergeConfig indicates an expected call of MergeConfig.
func (mr *MockBackendFactoryMockRecorder) MergeConfig(newCfg, oldCfg any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MergeConfig", reflect.TypeOf((*MockBackendFactory)(nil).MergeConfig), newCfg, oldCfg)
}

// ProcessHTTPRequest mocks base method.
func (m *MockBackendFactory) ProcessHTTPRequest(w http.ResponseWriter, r *http.Request) (string, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProcessHTTPRequest", w, r)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ProcessHTTPRequest indicates an expected call of ProcessHTTPRequest.
func (mr *MockBackendFactoryMockRecorder) ProcessHTTPRequest(w, r any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessHTTPRequest", reflect.TypeOf((*MockBackendFactory)(nil).ProcessHTTPRequest), w, r)
}

// RedactConfig mocks base method.
func (m *MockBackendFactory) RedactConfig(config map[string]string) map[string]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RedactConfig", config)
	ret0, _ := ret[0].(map[string]string)
	return ret0
}

// RedactConfig indicates an expected call of RedactConfig.
func (mr *MockBackendFactoryMockRecorder) RedactConfig(config any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RedactConfig", reflect.TypeOf((*MockBackendFactory)(nil).RedactConfig), config)
}

// ResolveProviderAndClientState mocks base method.
func (m *MockBackendFactory) ResolveProviderAndClientState(state string) (string, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveProviderAndClientState", state)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ResolveProviderAndClientState indicates an expected call of ResolveProviderAndClientState.
func (mr *MockBackendFactoryMockRecorder) ResolveProviderAndClientState(state any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveProviderAndClientState", reflect.TypeOf((*MockBackendFactory)(nil).ResolveProviderAndClientState), state)
}