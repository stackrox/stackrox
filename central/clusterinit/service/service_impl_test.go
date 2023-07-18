package service

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	dsMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/clusterinit/backend/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetInitBundlesWithBackendError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := dsMocks.NewMockDataStore(mockCtrl)
	backend := mocks.NewMockBackend(mockCtrl)
	service := New(backend, store)

	backend.EXPECT().GetAll(gomock.Any()).Return(nil, errors.New("some error"))

	bundles, err := service.GetInitBundles(context.Background(), nil)
	assert.Error(t, err)
	assert.Empty(t, bundles.GetItems())
}

func TestGetInitBundlesWithClusterStoreError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := dsMocks.NewMockDataStore(mockCtrl)
	backend := mocks.NewMockBackend(mockCtrl)
	service := New(backend, store)

	backend.EXPECT().GetAll(gomock.Any()).Return([]*storage.InitBundleMeta{
		{Id: "1", IsRevoked: false},
		{Id: "2", IsRevoked: true},
		{Id: "3", IsRevoked: false},
	}, nil)

	store.EXPECT().GetClusters(gomock.Any()).Return(nil, errors.New("some error"))

	bundles, err := service.GetInitBundles(context.Background(), nil)
	assert.Error(t, err)
	assert.Empty(t, bundles.GetItems())
}

func TestGetInitBundlesShouldReturnBundlesWithImpactedClusters(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := dsMocks.NewMockDataStore(mockCtrl)
	backend := mocks.NewMockBackend(mockCtrl)
	service := New(backend, store)

	backend.EXPECT().GetAll(gomock.Any()).Return([]*storage.InitBundleMeta{
		{Id: "1", IsRevoked: false},
		{Id: "2", IsRevoked: true},
		{Id: "3", IsRevoked: false},
	}, nil)

	store.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{
		{Id: "cluster-1", Name: "one", InitBundleId: "1"},
		{Id: "cluster-2", Name: "two", InitBundleId: "2"},
		{Id: "cluster-3", Name: "three", InitBundleId: "3"},
		{Id: "cluster-4", Name: "four", InitBundleId: "1"},
		{Id: "cluster-5", Name: "five", InitBundleId: "2"},
	}, nil)

	expected := []v1.InitBundleMeta{
		{Id: "1", ImpactedClusters: []*v1.InitBundleMeta_ImpactedCluster{
			{Id: "cluster-1", Name: "one"}, {Id: "cluster-4", Name: "four"}}},
		{Id: "2", ImpactedClusters: []*v1.InitBundleMeta_ImpactedCluster{
			{Id: "cluster-2", Name: "two"}, {Id: "cluster-5", Name: "five"}}},
		{Id: "3", ImpactedClusters: []*v1.InitBundleMeta_ImpactedCluster{
			{Id: "cluster-3", Name: "three"}}},
	}

	bundles, err := service.GetInitBundles(context.Background(), nil)
	assert.NoError(t, err)
	for i, bundle := range bundles.GetItems() {
		assert.Equal(t, expected[i].GetId(), bundle.GetId())
		assert.Equal(t, expected[i].GetName(), bundle.GetName())
		assert.ElementsMatch(t, expected[i].ImpactedClusters, bundle.ImpactedClusters)
	}
}

func TestRevokeInitBundles(t *testing.T) {
	testCase := []struct {
		name     string
		request  *v1.InitBundleRevokeRequest
		response *v1.InitBundleRevokeResponse
	}{
		{
			name:     "nil request => empty response",
			request:  nil,
			response: &v1.InitBundleRevokeResponse{},
		},
		{
			name:     "empty request => empty response",
			request:  &v1.InitBundleRevokeRequest{},
			response: &v1.InitBundleRevokeResponse{},
		},
		{
			name: "missing impacted cluster ids leads to error",
			request: &v1.InitBundleRevokeRequest{
				Ids: []string{"1"},
			},
			response: &v1.InitBundleRevokeResponse{
				InitBundleRevocationErrors: []*v1.InitBundleRevokeResponse_InitBundleRevocationError{
					{Id: "1", Error: "not all clusters were confirmed",
						ImpactedClusters: []*v1.InitBundleMeta_ImpactedCluster{
							{Id: "cluster-1", Name: "one"}, {Id: "cluster-4", Name: "four"},
						}},
				},
			},
		},
		{
			name: "impacted clusters match => revoke",
			request: &v1.InitBundleRevokeRequest{
				Ids:                        []string{"1"},
				ConfirmImpactedClustersIds: []string{"cluster-1", "cluster-4"},
			},
			response: &v1.InitBundleRevokeResponse{
				InitBundleRevokedIds: []string{"1"},
			},
		},
		{
			name: "multiple IDs request",
			request: &v1.InitBundleRevokeRequest{
				Ids:                        []string{"1", "2", "3", "4", "unknown"},
				ConfirmImpactedClustersIds: []string{"cluster-1", "cluster-2", "cluster-4", "cluster-5"},
			},
			response: &v1.InitBundleRevokeResponse{
				InitBundleRevocationErrors: []*v1.InitBundleRevokeResponse_InitBundleRevocationError{
					{Id: "3", Error: "not all clusters were confirmed",
						ImpactedClusters: []*v1.InitBundleMeta_ImpactedCluster{
							{Id: "cluster-3", Name: "three"},
						}},
					{Id: "unknown", Error: "some error"},
				},
				InitBundleRevokedIds: []string{"1", "2", "4"},
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			store := dsMocks.NewMockDataStore(mockCtrl)
			backend := mocks.NewMockBackend(mockCtrl)
			service := New(backend, store)

			store.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{
				{Id: "cluster-1", Name: "one", InitBundleId: "1"},
				{Id: "cluster-2", Name: "two", InitBundleId: "2"},
				{Id: "cluster-3", Name: "three", InitBundleId: "3"},
				{Id: "cluster-4", Name: "four", InitBundleId: "1"},
				{Id: "cluster-5", Name: "five", InitBundleId: "2"},
			}, nil)

			backend.EXPECT().Revoke(gomock.Any(), "unknown").AnyTimes().Return(errors.New("some error"))
			for _, id := range tc.response.GetInitBundleRevokedIds() {
				backend.EXPECT().Revoke(gomock.Any(), id).Return(nil)
			}

			response, err := service.RevokeInitBundle(context.Background(), tc.request)

			assert.NoError(t, err)
			assert.ElementsMatch(t, tc.response.GetInitBundleRevokedIds(), response.GetInitBundleRevokedIds())
			assert.ElementsMatch(t, tc.response.GetInitBundleRevocationErrors(), response.GetInitBundleRevocationErrors())
		})
	}
}

func TestRevokeInitBundlesWithClusterStoreErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	store := dsMocks.NewMockDataStore(mockCtrl)
	backend := mocks.NewMockBackend(mockCtrl)
	service := New(backend, store)

	store.EXPECT().GetClusters(gomock.Any()).Return(nil, errors.New("some error"))

	response, err := service.RevokeInitBundle(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, response)
}
