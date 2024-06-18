package storagetov2

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
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
func ComplianceV2ClusterOverallStats(resultCounts []*datastore.ResultStatusCountByCluster, clusterErrors map[string][]string) []*v2.ComplianceClusterOverallStats {
	var convertedResults []*v2.ComplianceClusterOverallStats

	for _, resultCount := range resultCounts {
		var lastScanTime *types.Timestamp
		if resultCount.LastScanTime != nil {
			lastScanTime = protoconv.ConvertTimeToTimestampOrNil(*resultCount.LastScanTime)
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
func ComplianceV2ProfileStats(resultCounts []*datastore.ResourceResultCountByProfile, profileMap map[string]*storage.ComplianceOperatorProfileV2, profileBenchmarkMap map[string][]*storage.ComplianceOperatorBenchmarkV2) []*v2.ComplianceProfileScanStats {
	var convertedResults []*v2.ComplianceProfileScanStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, &v2.ComplianceProfileScanStats{
			ProfileName: resultCount.ProfileName,
			Title:       profileMap[resultCount.ProfileName].GetTitle(),
			Version:     profileMap[resultCount.ProfileName].GetProfileVersion(),
			Benchmarks:  convertBenchmarks(profileBenchmarkMap[resultCount.ProfileName]),
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

func convertBenchmarks(incoming []*storage.ComplianceOperatorBenchmarkV2) []*v2.ComplianceBenchmark {
	var convertedBenchmarks []*v2.ComplianceBenchmark
	for _, benchmark := range incoming {
		convertedBenchmarks = append(convertedBenchmarks, &v2.ComplianceBenchmark{
			Name:        benchmark.GetName(),
			Version:     benchmark.GetVersion(),
			Description: benchmark.GetDescription(),
			Provider:    benchmark.GetProvider(),
			ShortName:   benchmark.GetShortName(),
		})
	}

	return convertedBenchmarks
}
