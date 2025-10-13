package helpers

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetFailedClusters(t *testing.T) {
	ctx := context.Background()
	scanConfigID := "scan-config-id"
	ctrl := gomock.NewController(t)
	snapshotStore := snapshotDS.NewMockDataStore(ctrl)
	scanStore := scanDS.NewMockDataStore(ctrl)
	snapshot := &storage.ComplianceOperatorReportSnapshotV2{
		FailedClusters: []*storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
			{
				ClusterId:       "cluster-id",
				ClusterName:     "cluster-name",
				Reasons:         []string{"some reason"},
				OperatorVersion: "v1.6.0",
				ScanNames:       []string{"scan-2"},
			},
		},
		Scans: []*storage.ComplianceOperatorReportSnapshotV2_Scan{
			{
				ScanRefId: "scan-ref-id-1",
			},
			{
				ScanRefId: "scan-ref-id-2",
			},
		},
	}
	t.Run("failure retrieving snapshot from the store", func(tt *testing.T) {
		snapshotStore.EXPECT().
			GetLastSnapshotFromScanConfig(gomock.Any(), gomock.Eq(scanConfigID)).
			Times(1).Return(nil, errors.New("some error"))
		failedClusters, err := GetFailedClusters(ctx, scanConfigID, snapshotStore, scanStore)
		assert.Error(tt, err)
		assert.Nil(tt, failedClusters)
	})
	t.Run("failure retrieving scans from the store", func(tt *testing.T) {
		snapshotStore.EXPECT().
			GetLastSnapshotFromScanConfig(gomock.Any(), gomock.Eq(scanConfigID)).
			Times(1).Return(snapshot, nil)
		scanStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).
			Times(1).Return(nil, errors.New("some error"))
		failedClusters, err := GetFailedClusters(ctx, scanConfigID, snapshotStore, scanStore)
		assert.Error(tt, err)
		assert.Nil(tt, failedClusters)
	})
	t.Run("populate failed clusters successfully", func(tt *testing.T) {
		snapshotStore.EXPECT().
			GetLastSnapshotFromScanConfig(gomock.Any(), gomock.Eq(scanConfigID)).
			Times(1).Return(snapshot, nil)
		scans := []*storage.ComplianceOperatorScanV2{
			{
				ScanName:  "scan-2",
				ScanRefId: "scan-ref-id-2",
			},
		}
		scanStore.EXPECT().SearchScans(gomock.Any(), gomock.Any()).
			Times(1).Return(scans, nil)
		expectedFailedClusters := map[string]*report.FailedCluster{
			"cluster-id": {
				ClusterId:       "cluster-id",
				ClusterName:     "cluster-name",
				Reasons:         []string{"some reason"},
				OperatorVersion: "v1.6.0",
				FailedScans:     scans,
			},
		}
		failedClusters, err := GetFailedClusters(ctx, scanConfigID, snapshotStore, scanStore)
		assert.NoError(tt, err)
		require.Len(tt, failedClusters, len(expectedFailedClusters))
		for clusterID, expectedCluster := range expectedFailedClusters {
			actualCluster, ok := failedClusters[clusterID]
			require.True(tt, ok)
			assert.Equal(tt, expectedCluster.ClusterId, actualCluster.ClusterId)
			assert.Equal(tt, expectedCluster.ClusterName, actualCluster.ClusterName)
			assert.Equal(tt, expectedCluster.Reasons, actualCluster.Reasons)
			assert.Equal(tt, expectedCluster.OperatorVersion, actualCluster.OperatorVersion)
			protoassert.SlicesEqual(t, expectedCluster.FailedScans, actualCluster.FailedScans)
		}
	})
}

func TestGetClusterData(t *testing.T) {
	ctx := context.Background()
	reportData := &storage.ComplianceOperatorReportData{
		ScanConfiguration: &storage.ComplianceOperatorScanConfigurationV2{
			Id: "scan-config-id",
		},
		ClusterStatus: []*storage.ComplianceOperatorReportData_ClusterStatus{
			{
				ClusterId:   "cluster-1",
				ClusterName: "cluster-1",
			},
			{
				ClusterId:   "cluster-2",
				ClusterName: "cluster-2",
			},
		},
	}
	failedClusters := map[string]*report.FailedCluster{
		"cluster-2": {
			ClusterId:       "cluster-2",
			ClusterName:     "cluster-2",
			Reasons:         []string{"some reason"},
			OperatorVersion: "v1.6.0",
			FailedScans: []*storage.ComplianceOperatorScanV2{
				{
					ScanName: "scan-2",
					Profile: &storage.ProfileShim{
						ProfileRefId: "profile-ref-id",
					},
				},
			},
		},
	}
	ctrl := gomock.NewController(t)
	scanStore := scanDS.NewMockDataStore(ctrl)
	t.Run("empty cluster status", func(tt *testing.T) {
		clusterData, err := GetClusterData(ctx, nil, failedClusters, scanStore)
		assert.NoError(tt, err)
		assert.Len(tt, clusterData, 0)
	})
	t.Run("failure querying the scan store", func(tt *testing.T) {
		scanStore.EXPECT().
			SearchScans(gomock.Any(), gomock.Any()).
			Times(1).Return(nil, errors.New("some error"))
		clusterData, err := GetClusterData(ctx, reportData, failedClusters, scanStore)
		assert.Error(tt, err)
		assert.Nil(tt, clusterData)
	})
	t.Run("no failed clusters", func(tt *testing.T) {
		gomock.InOrder(
			scanStore.EXPECT().
				SearchScans(gomock.Any(), gomock.Any()).
				Times(1).Return([]*storage.ComplianceOperatorScanV2{
				{
					ScanName: "scan-1",
				},
				{
					ScanName: "scan-2",
				},
			}, nil),
			scanStore.EXPECT().
				SearchScans(gomock.Any(), gomock.Any()).
				Times(1).Return([]*storage.ComplianceOperatorScanV2{
				{
					ScanName: "scan-1",
				},
				{
					ScanName: "scan-2",
				},
			}, nil),
		)
		expectedClusterData := map[string]*report.ClusterData{
			"cluster-1": {
				ClusterId:   "cluster-1",
				ClusterName: "cluster-1",
				ScanNames:   []string{"scan-1", "scan-2"},
			},
			"cluster-2": {
				ClusterId:   "cluster-2",
				ClusterName: "cluster-2",
				ScanNames:   []string{"scan-1", "scan-2"},
			},
		}
		clusterData, err := GetClusterData(ctx, reportData, nil, scanStore)
		assert.NoError(tt, err)
		assertClusterData(tt, expectedClusterData, clusterData)
	})
	t.Run("with failed clusters", func(tt *testing.T) {
		gomock.InOrder(
			scanStore.EXPECT().
				SearchScans(gomock.Any(), gomock.Any()).
				Times(1).Return([]*storage.ComplianceOperatorScanV2{
				{
					ScanName: "scan-1",
				},
				{
					ScanName: "scan-2",
				},
			}, nil),
			scanStore.EXPECT().
				SearchScans(gomock.Any(), gomock.Any()).
				Times(1).Return([]*storage.ComplianceOperatorScanV2{
				{
					ScanName: "scan-1",
				},
				{
					ScanName: "scan-2",
				},
			}, nil),
		)
		expectedClusterData := map[string]*report.ClusterData{
			"cluster-1": {
				ClusterId:   "cluster-1",
				ClusterName: "cluster-1",
				ScanNames:   []string{"scan-1", "scan-2"},
			},
			"cluster-2": {
				ClusterId:   "cluster-2",
				ClusterName: "cluster-2",
				ScanNames:   []string{"scan-1", "scan-2"},
				FailedInfo: &report.FailedCluster{
					ClusterId:       "cluster-2",
					ClusterName:     "cluster-2",
					OperatorVersion: "v1.6.0",
					Reasons:         []string{"some reason"},
					FailedScans: []*storage.ComplianceOperatorScanV2{
						{
							ScanName: "scan-2",
							Profile: &storage.ProfileShim{
								ProfileRefId: "profile-ref-id",
							},
						},
					},
				},
			},
		}
		clusterData, err := GetClusterData(ctx, reportData, failedClusters, scanStore)
		assert.NoError(tt, err)
		assertClusterData(tt, expectedClusterData, clusterData)
	})
}

func assertClusterData(t *testing.T, expected map[string]*report.ClusterData, actual map[string]*report.ClusterData) {
	assert.Len(t, actual, len(expected))
	for clusterID, expectedCluster := range expected {
		actualCluster, ok := actual[clusterID]
		require.True(t, ok)
		assert.Equal(t, expectedCluster.ClusterId, actualCluster.ClusterId)
		assert.Equal(t, expectedCluster.ClusterName, actualCluster.ClusterName)
		assert.ElementsMatch(t, expectedCluster.ScanNames, actualCluster.ScanNames)
		if expectedCluster.FailedInfo != nil {
			require.NotNil(t, actualCluster.FailedInfo)
			assert.Equal(t, expectedCluster.FailedInfo.ClusterId, actualCluster.FailedInfo.ClusterId)
			assert.Equal(t, expectedCluster.FailedInfo.ClusterName, actualCluster.FailedInfo.ClusterName)
			assert.Equal(t, expectedCluster.FailedInfo.Reasons, actualCluster.FailedInfo.Reasons)
			assert.Equal(t, expectedCluster.FailedInfo.OperatorVersion, actualCluster.FailedInfo.OperatorVersion)
			protoassert.SlicesEqual(t, expectedCluster.FailedInfo.FailedScans, actualCluster.FailedInfo.FailedScans)
		} else {
			assert.Nil(t, actualCluster.FailedInfo)
		}
	}
}
