package fetcher

import (
	"testing"

	mockClusterDataStore "github.com/stackrox/rox/central/cluster/datastore/mocks"
	mockClusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore/mocks"
	mockCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/cve/matcher"
	mockImageDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	mockImageV2DataStore "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	mockNSDataStore "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
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
