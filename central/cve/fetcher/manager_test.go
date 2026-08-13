package fetcher

import (
	"context"
	"testing"

	mockClusterDataStore "github.com/stackrox/rox/central/cluster/datastore/mocks"
	mockClusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore/mocks"
	mockCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/cve/matcher"
	mockImageDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	mockImageV2DataStore "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	mockNSDataStore "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewOrchestratorIstioCVEManagerImpl(t *testing.T) {
	tests := map[string]struct {
		legacyScannerEnabled bool
		setUp                func(t *testing.T, mockClusters *mockClusterDataStore.MockDataStore, mockCVEs *mockCVEDataStore.MockDataStore)
	}{
		"when LegacyScanner is disabled then no creators registered and initialize not called": {
			legacyScannerEnabled: false,
			setUp: func(t *testing.T, mockClusters *mockClusterDataStore.MockDataStore, mockCVEs *mockCVEDataStore.MockDataStore) {
			},
		},
		"when LegacyScanner is enabled then Clairify creator registered and initialize called": {
			legacyScannerEnabled: true,
			setUp: func(t *testing.T, mockClusters *mockClusterDataStore.MockDataStore, mockCVEs *mockCVEDataStore.MockDataStore) {
				mockClusters.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{}, nil).Times(1)
				mockCVEs.EXPECT().UpsertClusterCVEsInternal(gomock.Any(), storage.CVE_K8S_CVE).Return(nil).Times(1)
				mockCVEs.EXPECT().UpsertClusterCVEsInternal(gomock.Any(), storage.CVE_OPENSHIFT_CVE).Return(nil).Times(1)
				mockCVEs.EXPECT().UpsertClusterCVEsInternal(gomock.Any(), storage.CVE_ISTIO_CVE).Return(nil).Times(1)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.LegacyScanner, tt.legacyScannerEnabled)

			ctrl := gomock.NewController(t)
			mockClusters := mockClusterDataStore.NewMockDataStore(ctrl)
			mockCVEs := mockCVEDataStore.NewMockDataStore(ctrl)
			mockEdges := mockClusterCVEEdgeDataStore.NewMockDataStore(ctrl)
			mockNamespaces := mockNSDataStore.NewMockDataStore(ctrl)
			mockImages := mockImageDataStore.NewMockDataStore(ctrl)
			mockImagesV2 := mockImageV2DataStore.NewMockDataStore(ctrl)

			tt.setUp(t, mockClusters, mockCVEs)

			cveMatcher, err := matcher.NewCVEMatcher(mockClusters, mockNamespaces, mockImages, mockImagesV2)
			require.NoError(t, err)

			mgr, err := NewOrchestratorIstioCVEManagerImpl(mockClusters, mockCVEs, mockEdges, cveMatcher)
			assert.NoError(t, err)
			assert.NotNil(t, mgr)
		})
	}
}

func TestInertManager_ConsumerPathsSafe(t *testing.T) {
	testutils.MustUpdateFeature(t, features.LegacyScanner, false)

	ctrl := gomock.NewController(t)
	mockClusters := mockClusterDataStore.NewMockDataStore(ctrl)
	mockCVEs := mockCVEDataStore.NewMockDataStore(ctrl)
	mockEdges := mockClusterCVEEdgeDataStore.NewMockDataStore(ctrl)
	mockNamespaces := mockNSDataStore.NewMockDataStore(ctrl)
	mockImages := mockImageDataStore.NewMockDataStore(ctrl)
	mockImagesV2 := mockImageV2DataStore.NewMockDataStore(ctrl)

	cveMatcher, err := matcher.NewCVEMatcher(mockClusters, mockNamespaces, mockImages, mockImagesV2)
	require.NoError(t, err)

	mgr, err := NewOrchestratorIstioCVEManagerImpl(mockClusters, mockCVEs, mockEdges, cveMatcher)
	require.NoError(t, err)

	t.Run("GetAffectedClusters queries DB and returns results", func(t *testing.T) {
		ctx := sac.WithAllAccess(context.Background())
		expected := []*storage.Cluster{{Id: "cluster-1", Name: "test-cluster"}}
		mockClusters.EXPECT().SearchRawClusters(gomock.Any(), gomock.Any()).Return(expected, nil).Times(1)

		clusters, err := mgr.GetAffectedClusters(ctx, "CVE-2024-1234", utils.K8s, nil)
		assert.NoError(t, err)
		assert.Len(t, clusters, 1)
		assert.Equal(t, "cluster-1", clusters[0].GetId())
	})

	t.Run("HandleClusterConnection does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			mgr.HandleClusterConnection()
		})
	})

	t.Run("UpsertOrchestratorIntegration returns error for missing creator", func(t *testing.T) {
		err := mgr.UpsertOrchestratorIntegration(&storage.OrchestratorIntegration{
			Id:   "test-id",
			Type: "clairify",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("RemoveIntegration does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			mgr.RemoveIntegration("nonexistent-id")
		})
	})
}
