// Code generated by MockGen. DO NOT EDIT.
// Source: datastore.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/datastore.go -source datastore.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	gomock "go.uber.org/mock/gomock"
)

// MockDataStore is a mock of DataStore interface.
type MockDataStore struct {
	ctrl     *gomock.Controller
	recorder *MockDataStoreMockRecorder
	isgomock struct{}
}

// MockDataStoreMockRecorder is the mock recorder for MockDataStore.
type MockDataStoreMockRecorder struct {
	mock *MockDataStore
}

// NewMockDataStore creates a new mock instance.
func NewMockDataStore(ctrl *gomock.Controller) *MockDataStore {
	mock := &MockDataStore{ctrl: ctrl}
	mock.recorder = &MockDataStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDataStore) EXPECT() *MockDataStoreMockRecorder {
	return m.recorder
}

// DeleteScanSettingBinding mocks base method.
func (m *MockDataStore) DeleteScanSettingBinding(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteScanSettingBinding", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteScanSettingBinding indicates an expected call of DeleteScanSettingBinding.
func (mr *MockDataStoreMockRecorder) DeleteScanSettingBinding(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteScanSettingBinding", reflect.TypeOf((*MockDataStore)(nil).DeleteScanSettingBinding), ctx, id)
}

// DeleteScanSettingByCluster mocks base method.
func (m *MockDataStore) DeleteScanSettingByCluster(ctx context.Context, clusterID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteScanSettingByCluster", ctx, clusterID)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteScanSettingByCluster indicates an expected call of DeleteScanSettingByCluster.
func (mr *MockDataStoreMockRecorder) DeleteScanSettingByCluster(ctx, clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteScanSettingByCluster", reflect.TypeOf((*MockDataStore)(nil).DeleteScanSettingByCluster), ctx, clusterID)
}

// GetScanSettingBinding mocks base method.
func (m *MockDataStore) GetScanSettingBinding(ctx context.Context, id string) (*storage.ComplianceOperatorScanSettingBindingV2, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScanSettingBinding", ctx, id)
	ret0, _ := ret[0].(*storage.ComplianceOperatorScanSettingBindingV2)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetScanSettingBinding indicates an expected call of GetScanSettingBinding.
func (mr *MockDataStoreMockRecorder) GetScanSettingBinding(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScanSettingBinding", reflect.TypeOf((*MockDataStore)(nil).GetScanSettingBinding), ctx, id)
}

// GetScanSettingBindings mocks base method.
func (m *MockDataStore) GetScanSettingBindings(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanSettingBindingV2, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScanSettingBindings", ctx, query)
	ret0, _ := ret[0].([]*storage.ComplianceOperatorScanSettingBindingV2)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetScanSettingBindings indicates an expected call of GetScanSettingBindings.
func (mr *MockDataStoreMockRecorder) GetScanSettingBindings(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScanSettingBindings", reflect.TypeOf((*MockDataStore)(nil).GetScanSettingBindings), ctx, query)
}

// GetScanSettingBindingsByCluster mocks base method.
func (m *MockDataStore) GetScanSettingBindingsByCluster(ctx context.Context, clusterID string) ([]*storage.ComplianceOperatorScanSettingBindingV2, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScanSettingBindingsByCluster", ctx, clusterID)
	ret0, _ := ret[0].([]*storage.ComplianceOperatorScanSettingBindingV2)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetScanSettingBindingsByCluster indicates an expected call of GetScanSettingBindingsByCluster.
func (mr *MockDataStoreMockRecorder) GetScanSettingBindingsByCluster(ctx, clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScanSettingBindingsByCluster", reflect.TypeOf((*MockDataStore)(nil).GetScanSettingBindingsByCluster), ctx, clusterID)
}

// UpsertScanSettingBinding mocks base method.
func (m *MockDataStore) UpsertScanSettingBinding(ctx context.Context, result *storage.ComplianceOperatorScanSettingBindingV2) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpsertScanSettingBinding", ctx, result)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpsertScanSettingBinding indicates an expected call of UpsertScanSettingBinding.
func (mr *MockDataStoreMockRecorder) UpsertScanSettingBinding(ctx, result any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpsertScanSettingBinding", reflect.TypeOf((*MockDataStore)(nil).UpsertScanSettingBinding), ctx, result)
}
