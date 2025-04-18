// Code generated by MockGen. DO NOT EDIT.
// Source: client.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/client.go -source client.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	securityhub "github.com/aws/aws-sdk-go-v2/service/securityhub"
	gomock "go.uber.org/mock/gomock"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
	isgomock struct{}
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// BatchImportFindings mocks base method.
func (m *MockClient) BatchImportFindings(arg0 context.Context, arg1 *securityhub.BatchImportFindingsInput, arg2 ...func(*securityhub.Options)) (*securityhub.BatchImportFindingsOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "BatchImportFindings", varargs...)
	ret0, _ := ret[0].(*securityhub.BatchImportFindingsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BatchImportFindings indicates an expected call of BatchImportFindings.
func (mr *MockClientMockRecorder) BatchImportFindings(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchImportFindings", reflect.TypeOf((*MockClient)(nil).BatchImportFindings), varargs...)
}

// GetFindings mocks base method.
func (m *MockClient) GetFindings(arg0 context.Context, arg1 *securityhub.GetFindingsInput, arg2 ...func(*securityhub.Options)) (*securityhub.GetFindingsOutput, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetFindings", varargs...)
	ret0, _ := ret[0].(*securityhub.GetFindingsOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFindings indicates an expected call of GetFindings.
func (mr *MockClientMockRecorder) GetFindings(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFindings", reflect.TypeOf((*MockClient)(nil).GetFindings), varargs...)
}
