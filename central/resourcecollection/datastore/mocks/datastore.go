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
	search "github.com/stackrox/rox/pkg/search"
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

// AddCollection mocks base method.
func (m *MockDataStore) AddCollection(ctx context.Context, collection *storage.ResourceCollection) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddCollection", ctx, collection)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddCollection indicates an expected call of AddCollection.
func (mr *MockDataStoreMockRecorder) AddCollection(ctx, collection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddCollection", reflect.TypeOf((*MockDataStore)(nil).AddCollection), ctx, collection)
}

// Count mocks base method.
func (m *MockDataStore) Count(ctx context.Context, q *v1.Query) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Count", ctx, q)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Count indicates an expected call of Count.
func (mr *MockDataStoreMockRecorder) Count(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Count", reflect.TypeOf((*MockDataStore)(nil).Count), ctx, q)
}

// DeleteCollection mocks base method.
func (m *MockDataStore) DeleteCollection(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCollection", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCollection indicates an expected call of DeleteCollection.
func (mr *MockDataStoreMockRecorder) DeleteCollection(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCollection", reflect.TypeOf((*MockDataStore)(nil).DeleteCollection), ctx, id)
}

// DryRunAddCollection mocks base method.
func (m *MockDataStore) DryRunAddCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DryRunAddCollection", ctx, collection)
	ret0, _ := ret[0].(error)
	return ret0
}

// DryRunAddCollection indicates an expected call of DryRunAddCollection.
func (mr *MockDataStoreMockRecorder) DryRunAddCollection(ctx, collection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DryRunAddCollection", reflect.TypeOf((*MockDataStore)(nil).DryRunAddCollection), ctx, collection)
}

// DryRunUpdateCollection mocks base method.
func (m *MockDataStore) DryRunUpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DryRunUpdateCollection", ctx, collection)
	ret0, _ := ret[0].(error)
	return ret0
}

// DryRunUpdateCollection indicates an expected call of DryRunUpdateCollection.
func (mr *MockDataStoreMockRecorder) DryRunUpdateCollection(ctx, collection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DryRunUpdateCollection", reflect.TypeOf((*MockDataStore)(nil).DryRunUpdateCollection), ctx, collection)
}

// Exists mocks base method.
func (m *MockDataStore) Exists(ctx context.Context, id string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", ctx, id)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exists indicates an expected call of Exists.
func (mr *MockDataStoreMockRecorder) Exists(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockDataStore)(nil).Exists), ctx, id)
}

// Get mocks base method.
func (m *MockDataStore) Get(ctx context.Context, id string) (*storage.ResourceCollection, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id)
	ret0, _ := ret[0].(*storage.ResourceCollection)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockDataStoreMockRecorder) Get(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockDataStore)(nil).Get), ctx, id)
}

// GetMany mocks base method.
func (m *MockDataStore) GetMany(ctx context.Context, id []string) ([]*storage.ResourceCollection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMany", ctx, id)
	ret0, _ := ret[0].([]*storage.ResourceCollection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMany indicates an expected call of GetMany.
func (mr *MockDataStoreMockRecorder) GetMany(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMany", reflect.TypeOf((*MockDataStore)(nil).GetMany), ctx, id)
}

// Search mocks base method.
func (m *MockDataStore) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Search", ctx, q)
	ret0, _ := ret[0].([]search.Result)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Search indicates an expected call of Search.
func (mr *MockDataStoreMockRecorder) Search(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Search", reflect.TypeOf((*MockDataStore)(nil).Search), ctx, q)
}

// SearchCollections mocks base method.
func (m *MockDataStore) SearchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SearchCollections", ctx, q)
	ret0, _ := ret[0].([]*storage.ResourceCollection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SearchCollections indicates an expected call of SearchCollections.
func (mr *MockDataStoreMockRecorder) SearchCollections(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SearchCollections", reflect.TypeOf((*MockDataStore)(nil).SearchCollections), ctx, q)
}

// SearchResults mocks base method.
func (m *MockDataStore) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SearchResults", ctx, q)
	ret0, _ := ret[0].([]*v1.SearchResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SearchResults indicates an expected call of SearchResults.
func (mr *MockDataStoreMockRecorder) SearchResults(ctx, q any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SearchResults", reflect.TypeOf((*MockDataStore)(nil).SearchResults), ctx, q)
}

// UpdateCollection mocks base method.
func (m *MockDataStore) UpdateCollection(ctx context.Context, collection *storage.ResourceCollection) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateCollection", ctx, collection)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateCollection indicates an expected call of UpdateCollection.
func (mr *MockDataStoreMockRecorder) UpdateCollection(ctx, collection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateCollection", reflect.TypeOf((*MockDataStore)(nil).UpdateCollection), ctx, collection)
}

// MockQueryResolver is a mock of QueryResolver interface.
type MockQueryResolver struct {
	ctrl     *gomock.Controller
	recorder *MockQueryResolverMockRecorder
	isgomock struct{}
}

// MockQueryResolverMockRecorder is the mock recorder for MockQueryResolver.
type MockQueryResolverMockRecorder struct {
	mock *MockQueryResolver
}

// NewMockQueryResolver creates a new mock instance.
func NewMockQueryResolver(ctrl *gomock.Controller) *MockQueryResolver {
	mock := &MockQueryResolver{ctrl: ctrl}
	mock.recorder = &MockQueryResolverMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQueryResolver) EXPECT() *MockQueryResolverMockRecorder {
	return m.recorder
}

// ResolveCollectionQuery mocks base method.
func (m *MockQueryResolver) ResolveCollectionQuery(ctx context.Context, collection *storage.ResourceCollection) (*v1.Query, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveCollectionQuery", ctx, collection)
	ret0, _ := ret[0].(*v1.Query)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveCollectionQuery indicates an expected call of ResolveCollectionQuery.
func (mr *MockQueryResolverMockRecorder) ResolveCollectionQuery(ctx, collection any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveCollectionQuery", reflect.TypeOf((*MockQueryResolver)(nil).ResolveCollectionQuery), ctx, collection)
}
