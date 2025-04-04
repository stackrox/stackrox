// Code generated by MockGen. DO NOT EDIT.
// Source: exchanger.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/exchanger.go -source exchanger.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	storage "github.com/stackrox/rox/generated/storage"
	authproviders "github.com/stackrox/rox/pkg/auth/authproviders"
	gomock "go.uber.org/mock/gomock"
)

// MockTokenExchanger is a mock of TokenExchanger interface.
type MockTokenExchanger struct {
	ctrl     *gomock.Controller
	recorder *MockTokenExchangerMockRecorder
	isgomock struct{}
}

// MockTokenExchangerMockRecorder is the mock recorder for MockTokenExchanger.
type MockTokenExchangerMockRecorder struct {
	mock *MockTokenExchanger
}

// NewMockTokenExchanger creates a new mock instance.
func NewMockTokenExchanger(ctrl *gomock.Controller) *MockTokenExchanger {
	mock := &MockTokenExchanger{ctrl: ctrl}
	mock.recorder = &MockTokenExchangerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTokenExchanger) EXPECT() *MockTokenExchangerMockRecorder {
	return m.recorder
}

// Config mocks base method.
func (m *MockTokenExchanger) Config() *storage.AuthMachineToMachineConfig {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Config")
	ret0, _ := ret[0].(*storage.AuthMachineToMachineConfig)
	return ret0
}

// Config indicates an expected call of Config.
func (mr *MockTokenExchangerMockRecorder) Config() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Config", reflect.TypeOf((*MockTokenExchanger)(nil).Config))
}

// ExchangeToken mocks base method.
func (m *MockTokenExchanger) ExchangeToken(ctx context.Context, rawIDToken string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExchangeToken", ctx, rawIDToken)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExchangeToken indicates an expected call of ExchangeToken.
func (mr *MockTokenExchangerMockRecorder) ExchangeToken(ctx, rawIDToken any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExchangeToken", reflect.TypeOf((*MockTokenExchanger)(nil).ExchangeToken), ctx, rawIDToken)
}

// Provider mocks base method.
func (m *MockTokenExchanger) Provider() authproviders.Provider {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Provider")
	ret0, _ := ret[0].(authproviders.Provider)
	return ret0
}

// Provider indicates an expected call of Provider.
func (mr *MockTokenExchangerMockRecorder) Provider() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Provider", reflect.TypeOf((*MockTokenExchanger)(nil).Provider))
}
