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

// CountScanConfigurations mocks base method.
func (m *MockDataStore) CountScanConfigurations(ctx context.Context, q *v1.Query) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CountScanConfigurations", ctx, q)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CountScanConfigurations indicates an expected call of CountScanConfigurations.
func (mr *MockDataStoreMockRecorder) CountScanConfigurations(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CountScanConfigurations", reflect.TypeOf((*MockDataStore)(nil).CountScanConfigurations), ctx, q)
}

// DeleteScanConfiguration mocks base method.
func (m *MockDataStore) DeleteScanConfiguration(ctx context.Context, id string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteScanConfiguration", ctx, id)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteScanConfiguration indicates an expected call of DeleteScanConfiguration.
func (mr *MockDataStoreMockRecorder) DeleteScanConfiguration(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteScanConfiguration", reflect.TypeOf((*MockDataStore)(nil).DeleteScanConfiguration), ctx, id)
}

// DistinctProfiles mocks base method.
func (m *MockDataStore) DistinctProfiles(ctx context.Context, q *v1.Query) (map[string]int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DistinctProfiles", ctx, q)
	ret0, _ := ret[0].(map[string]int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DistinctProfiles indicates an expected call of DistinctProfiles.
func (mr *MockDataStoreMockRecorder) DistinctProfiles(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DistinctProfiles", reflect.TypeOf((*MockDataStore)(nil).DistinctProfiles), ctx, q)
}

// GetProfilesNames mocks base method.
func (m *MockDataStore) GetProfilesNames(ctx context.Context, q *v1.Query) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProfilesNames", ctx, q)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProfilesNames indicates an expected call of GetProfilesNames.
func (mr *MockDataStoreMockRecorder) GetProfilesNames(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProfilesNames", reflect.TypeOf((*MockDataStore)(nil).GetProfilesNames), ctx, q)
}

// GetScanConfigClusterStatus mocks base method.
func (m *MockDataStore) GetScanConfigClusterStatus(ctx context.Context, scanConfigID string) ([]*storage.ComplianceOperatorClusterScanConfigStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScanConfigClusterStatus", ctx, scanConfigID)
	ret0, _ := ret[0].([]*storage.ComplianceOperatorClusterScanConfigStatus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetScanConfigClusterStatus indicates an expected call of GetScanConfigClusterStatus.
func (mr *MockDataStoreMockRecorder) GetScanConfigClusterStatus(ctx, scanConfigID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScanConfigClusterStatus", reflect.TypeOf((*MockDataStore)(nil).GetScanConfigClusterStatus), ctx, scanConfigID)
}

// GetScanConfiguration mocks base method.
func (m *MockDataStore) GetScanConfiguration(ctx context.Context, id string) (*storage.ComplianceOperatorScanConfigurationV2, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScanConfiguration", ctx, id)
	ret0, _ := ret[0].(*storage.ComplianceOperatorScanConfigurationV2)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetScanConfiguration indicates an expected call of GetScanConfiguration.
func (mr *MockDataStoreMockRecorder) GetScanConfiguration(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScanConfiguration", reflect.TypeOf((*MockDataStore)(nil).GetScanConfiguration), ctx, id)
}

// GetScanConfigurationByName mocks base method.
func (m *MockDataStore) GetScanConfigurationByName(ctx context.Context, scanName string) (*storage.ComplianceOperatorScanConfigurationV2, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScanConfigurationByName", ctx, scanName)
	ret0, _ := ret[0].(*storage.ComplianceOperatorScanConfigurationV2)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetScanConfigurationByName indicates an expected call of GetScanConfigurationByName.
func (mr *MockDataStoreMockRecorder) GetScanConfigurationByName(ctx, scanName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScanConfigurationByName", reflect.TypeOf((*MockDataStore)(nil).GetScanConfigurationByName), ctx, scanName)
}

// GetScanConfigurations mocks base method.
func (m *MockDataStore) GetScanConfigurations(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorScanConfigurationV2, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScanConfigurations", ctx, query)
	ret0, _ := ret[0].([]*storage.ComplianceOperatorScanConfigurationV2)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetScanConfigurations indicates an expected call of GetScanConfigurations.
func (mr *MockDataStoreMockRecorder) GetScanConfigurations(ctx, query any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScanConfigurations", reflect.TypeOf((*MockDataStore)(nil).GetScanConfigurations), ctx, query)
}

// RemoveClusterFromScanConfig mocks base method.
func (m *MockDataStore) RemoveClusterFromScanConfig(ctx context.Context, clusterID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveClusterFromScanConfig", ctx, clusterID)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveClusterFromScanConfig indicates an expected call of RemoveClusterFromScanConfig.
func (mr *MockDataStoreMockRecorder) RemoveClusterFromScanConfig(ctx, clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveClusterFromScanConfig", reflect.TypeOf((*MockDataStore)(nil).RemoveClusterFromScanConfig), ctx, clusterID)
}

// RemoveClusterStatus mocks base method.
func (m *MockDataStore) RemoveClusterStatus(ctx context.Context, scanConfigID, clusterID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveClusterStatus", ctx, scanConfigID, clusterID)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveClusterStatus indicates an expected call of RemoveClusterStatus.
func (mr *MockDataStoreMockRecorder) RemoveClusterStatus(ctx, scanConfigID, clusterID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveClusterStatus", reflect.TypeOf((*MockDataStore)(nil).RemoveClusterStatus), ctx, scanConfigID, clusterID)
}

// ScanConfigurationProfileExists mocks base method.
func (m *MockDataStore) ScanConfigurationProfileExists(ctx context.Context, id string, profiles, clusters []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ScanConfigurationProfileExists", ctx, id, profiles, clusters)
	ret0, _ := ret[0].(error)
	return ret0
}

// ScanConfigurationProfileExists indicates an expected call of ScanConfigurationProfileExists.
func (mr *MockDataStoreMockRecorder) ScanConfigurationProfileExists(ctx, id, profiles, clusters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ScanConfigurationProfileExists", reflect.TypeOf((*MockDataStore)(nil).ScanConfigurationProfileExists), ctx, id, profiles, clusters)
}

// UpdateClusterStatus mocks base method.
func (m *MockDataStore) UpdateClusterStatus(ctx context.Context, scanConfigID, clusterID, clusterStatus, clusterName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateClusterStatus", ctx, scanConfigID, clusterID, clusterStatus, clusterName)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateClusterStatus indicates an expected call of UpdateClusterStatus.
func (mr *MockDataStoreMockRecorder) UpdateClusterStatus(ctx, scanConfigID, clusterID, clusterStatus, clusterName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateClusterStatus", reflect.TypeOf((*MockDataStore)(nil).UpdateClusterStatus), ctx, scanConfigID, clusterID, clusterStatus, clusterName)
}

// UpsertScanConfiguration mocks base method.
func (m *MockDataStore) UpsertScanConfiguration(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpsertScanConfiguration", ctx, scanConfig)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpsertScanConfiguration indicates an expected call of UpsertScanConfiguration.
func (mr *MockDataStoreMockRecorder) UpsertScanConfiguration(ctx, scanConfig any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpsertScanConfiguration", reflect.TypeOf((*MockDataStore)(nil).UpsertScanConfiguration), ctx, scanConfig)
}
