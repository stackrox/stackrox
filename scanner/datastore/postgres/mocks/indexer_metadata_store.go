// Code generated by MockGen. DO NOT EDIT.
// Source: indexer_metadata_store.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/indexer_metadata_store.go -source indexer_metadata_store.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	postgres "github.com/stackrox/rox/scanner/datastore/postgres"
	gomock "go.uber.org/mock/gomock"
)

// MockIndexerMetadataStore is a mock of IndexerMetadataStore interface.
type MockIndexerMetadataStore struct {
	ctrl     *gomock.Controller
	recorder *MockIndexerMetadataStoreMockRecorder
}

// MockIndexerMetadataStoreMockRecorder is the mock recorder for MockIndexerMetadataStore.
type MockIndexerMetadataStoreMockRecorder struct {
	mock *MockIndexerMetadataStore
}

// NewMockIndexerMetadataStore creates a new mock instance.
func NewMockIndexerMetadataStore(ctrl *gomock.Controller) *MockIndexerMetadataStore {
	mock := &MockIndexerMetadataStore{ctrl: ctrl}
	mock.recorder = &MockIndexerMetadataStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIndexerMetadataStore) EXPECT() *MockIndexerMetadataStoreMockRecorder {
	return m.recorder
}

// GCManifests mocks base method.
func (m *MockIndexerMetadataStore) GCManifests(ctx context.Context, expiration time.Time, opts ...postgres.GCManifestsOption) ([]string, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, expiration}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GCManifests", varargs...)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GCManifests indicates an expected call of GCManifests.
func (mr *MockIndexerMetadataStoreMockRecorder) GCManifests(ctx, expiration any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, expiration}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GCManifests", reflect.TypeOf((*MockIndexerMetadataStore)(nil).GCManifests), varargs...)
}

// ManifestExists mocks base method.
func (m *MockIndexerMetadataStore) ManifestExists(ctx context.Context, manifestID string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ManifestExists", ctx, manifestID)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ManifestExists indicates an expected call of ManifestExists.
func (mr *MockIndexerMetadataStoreMockRecorder) ManifestExists(ctx, manifestID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ManifestExists", reflect.TypeOf((*MockIndexerMetadataStore)(nil).ManifestExists), ctx, manifestID)
}

// MigrateManifests mocks base method.
func (m *MockIndexerMetadataStore) MigrateManifests(ctx context.Context, expiration time.Time) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MigrateManifests", ctx, expiration)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MigrateManifests indicates an expected call of MigrateManifests.
func (mr *MockIndexerMetadataStoreMockRecorder) MigrateManifests(ctx, expiration any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MigrateManifests", reflect.TypeOf((*MockIndexerMetadataStore)(nil).MigrateManifests), ctx, expiration)
}

// StoreManifest mocks base method.
func (m *MockIndexerMetadataStore) StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreManifest", ctx, manifestID, expiration)
	ret0, _ := ret[0].(error)
	return ret0
}

// StoreManifest indicates an expected call of StoreManifest.
func (mr *MockIndexerMetadataStoreMockRecorder) StoreManifest(ctx, manifestID, expiration any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreManifest", reflect.TypeOf((*MockIndexerMetadataStore)(nil).StoreManifest), ctx, manifestID, expiration)
}