package storagetov2

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	types "github.com/stackrox/rox/pkg/protocompat"
)

// ComplianceV2ClusterStats converts the counts to the v2 stats
func ComplianceV2ClusterStats(resultCounts []*datastore.ResourceResultCountByClusterScan, scanToScanID map[string]string) []*v2.ComplianceClusterScanStats {
	var convertedResults []*v2.ComplianceClusterScanStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, &v2.ComplianceClusterScanStats{
			Cluster: &v2.ComplianceScanCluster{
				ClusterId:   resultCount.ClusterID,
				ClusterName: resultCount.ClusterName,
			},
			ScanStats: &v2.ComplianceScanStatsShim{
				ScanName:     resultCount.ScanConfigName,
				ScanConfigId: scanToScanID[resultCount.ScanConfigName],
				CheckStats: []*v2.ComplianceCheckStatusCount{
					{
						Count:  int32(resultCount.FailCount),
						Status: v2.ComplianceCheckStatus_FAIL,
					},
					{
						Count:  int32(resultCount.InfoCount),
						Status: v2.ComplianceCheckStatus_INFO,
					},
					{
						Count:  int32(resultCount.PassCount),
						Status: v2.ComplianceCheckStatus_PASS,
					},
					{
						Count:  int32(resultCount.ErrorCount),
						Status: v2.ComplianceCheckStatus_ERROR,
					},
					{
						Count:  int32(resultCount.ManualCount),
						Status: v2.ComplianceCheckStatus_MANUAL,
					},
					{
						Count:  int32(resultCount.InconsistentCount),
						Status: v2.ComplianceCheckStatus_INCONSISTENT,
					},
					{
						Count:  int32(resultCount.NotApplicableCount),
						Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
					},
				},
			},
		})
	}
	return convertedResults
}

// ComplianceV2ClusterOverallStats converts the counts to the v2 stats
func ComplianceV2ClusterOverallStats(resultCounts []*datastore.ResultStatusCountByCluster, clusterErrors map[string][]string, clusterLastScan map[string]*types.Timestamp) []*v2.ComplianceClusterOverallStats {
	var convertedResults []*v2.ComplianceClusterOverallStats

	for _, resultCount := range resultCounts {
		var lastScanTime *types.Timestamp
		if clusterLastScan != nil {
			lastScanTime = clusterLastScan[resultCount.ClusterID]
		}

		convertedResults = append(convertedResults, &v2.ComplianceClusterOverallStats{
			Cluster: &v2.ComplianceScanCluster{
				ClusterId:   resultCount.ClusterID,
				ClusterName: resultCount.ClusterName,
			},
			ClusterErrors: clusterErrors[resultCount.ClusterID],
			CheckStats: []*v2.ComplianceCheckStatusCount{
				{
					Count:  int32(resultCount.FailCount),
					Status: v2.ComplianceCheckStatus_FAIL,
				},
				{
					Count:  int32(resultCount.InfoCount),
					Status: v2.ComplianceCheckStatus_INFO,
				},
				{
					Count:  int32(resultCount.PassCount),
					Status: v2.ComplianceCheckStatus_PASS,
				},
				{
					Count:  int32(resultCount.ErrorCount),
					Status: v2.ComplianceCheckStatus_ERROR,
				},
				{
					Count:  int32(resultCount.ManualCount),
					Status: v2.ComplianceCheckStatus_MANUAL,
				},
				{
					Count:  int32(resultCount.InconsistentCount),
					Status: v2.ComplianceCheckStatus_INCONSISTENT,
				},
				{
					Count:  int32(resultCount.NotApplicableCount),
					Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
				},
			},
			LastScanTime: lastScanTime,
		})
	}
	return convertedResults
}

// ComplianceV2ProfileStats converts the counts to the v2 stats
func ComplianceV2ProfileStats(resultCounts []*datastore.ResourceResultCountByProfile, profileMap map[string]*storage.ComplianceOperatorProfileV2) []*v2.ComplianceProfileScanStats {
	var convertedResults []*v2.ComplianceProfileScanStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, &v2.ComplianceProfileScanStats{
			ProfileName: resultCount.ProfileName,
			Title:       profileMap[resultCount.ProfileName].GetTitle(),
			Version:     profileMap[resultCount.ProfileName].GetProfileVersion(),
			CheckStats: []*v2.ComplianceCheckStatusCount{
				{
					Count:  int32(resultCount.FailCount),
					Status: v2.ComplianceCheckStatus_FAIL,
				},
				{
					Count:  int32(resultCount.InfoCount),
					Status: v2.ComplianceCheckStatus_INFO,
				},
				{
					Count:  int32(resultCount.PassCount),
					Status: v2.ComplianceCheckStatus_PASS,
				},
				{
					Count:  int32(resultCount.ErrorCount),
					Status: v2.ComplianceCheckStatus_ERROR,
				},
				{
					Count:  int32(resultCount.ManualCount),
					Status: v2.ComplianceCheckStatus_MANUAL,
				},
				{
					Count:  int32(resultCount.InconsistentCount),
					Status: v2.ComplianceCheckStatus_INCONSISTENT,
				},
				{
					Count:  int32(resultCount.NotApplicableCount),
					Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
				},
			},
		})
	}
	return convertedResults
}
