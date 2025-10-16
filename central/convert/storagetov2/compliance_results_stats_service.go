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
		convertedResults = append(convertedResults, v2.ComplianceClusterScanStats_builder{
			Cluster: v2.ComplianceScanCluster_builder{
				ClusterId:   resultCount.ClusterID,
				ClusterName: resultCount.ClusterName,
			}.Build(),
			ScanStats: v2.ComplianceScanStatsShim_builder{
				ScanName:     resultCount.ScanConfigName,
				ScanConfigId: scanToScanID[resultCount.ScanConfigName],
				CheckStats: []*v2.ComplianceCheckStatusCount{
					v2.ComplianceCheckStatusCount_builder{
						Count:  int32(resultCount.FailCount),
						Status: v2.ComplianceCheckStatus_FAIL,
					}.Build(),
					v2.ComplianceCheckStatusCount_builder{
						Count:  int32(resultCount.InfoCount),
						Status: v2.ComplianceCheckStatus_INFO,
					}.Build(),
					v2.ComplianceCheckStatusCount_builder{
						Count:  int32(resultCount.PassCount),
						Status: v2.ComplianceCheckStatus_PASS,
					}.Build(),
					v2.ComplianceCheckStatusCount_builder{
						Count:  int32(resultCount.ErrorCount),
						Status: v2.ComplianceCheckStatus_ERROR,
					}.Build(),
					v2.ComplianceCheckStatusCount_builder{
						Count:  int32(resultCount.ManualCount),
						Status: v2.ComplianceCheckStatus_MANUAL,
					}.Build(),
					v2.ComplianceCheckStatusCount_builder{
						Count:  int32(resultCount.InconsistentCount),
						Status: v2.ComplianceCheckStatus_INCONSISTENT,
					}.Build(),
					v2.ComplianceCheckStatusCount_builder{
						Count:  int32(resultCount.NotApplicableCount),
						Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
					}.Build(),
				},
			}.Build(),
		}.Build())
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
		convertedResults = append(convertedResults, v2.ComplianceClusterOverallStats_builder{
			Cluster: v2.ComplianceScanCluster_builder{
				ClusterId:   resultCount.ClusterID,
				ClusterName: resultCount.ClusterName,
			}.Build(),
			ClusterErrors: clusterErrors[resultCount.ClusterID],
			CheckStats: []*v2.ComplianceCheckStatusCount{
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.FailCount),
					Status: v2.ComplianceCheckStatus_FAIL,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.InfoCount),
					Status: v2.ComplianceCheckStatus_INFO,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.PassCount),
					Status: v2.ComplianceCheckStatus_PASS,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.ErrorCount),
					Status: v2.ComplianceCheckStatus_ERROR,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.ManualCount),
					Status: v2.ComplianceCheckStatus_MANUAL,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.InconsistentCount),
					Status: v2.ComplianceCheckStatus_INCONSISTENT,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.NotApplicableCount),
					Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
				}.Build(),
			},
			LastScanTime: lastScanTime,
		}.Build())
	}
	return convertedResults
}

// ComplianceV2ProfileStats converts the counts to the v2 stats
func ComplianceV2ProfileStats(resultCounts []*datastore.ResourceResultCountByProfile, profileMap map[string]*storage.ComplianceOperatorProfileV2, profileBenchmarkMap map[string][]*storage.ComplianceOperatorBenchmarkV2) []*v2.ComplianceProfileScanStats {
	var convertedResults []*v2.ComplianceProfileScanStats

	for _, resultCount := range resultCounts {
		convertedResults = append(convertedResults, v2.ComplianceProfileScanStats_builder{
			ProfileName: resultCount.ProfileName,
			Title:       profileMap[resultCount.ProfileName].GetTitle(),
			Version:     profileMap[resultCount.ProfileName].GetProfileVersion(),
			Benchmarks:  convertBenchmarks(profileBenchmarkMap[resultCount.ProfileName]),
			CheckStats: []*v2.ComplianceCheckStatusCount{
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.FailCount),
					Status: v2.ComplianceCheckStatus_FAIL,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.InfoCount),
					Status: v2.ComplianceCheckStatus_INFO,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.PassCount),
					Status: v2.ComplianceCheckStatus_PASS,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.ErrorCount),
					Status: v2.ComplianceCheckStatus_ERROR,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.ManualCount),
					Status: v2.ComplianceCheckStatus_MANUAL,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.InconsistentCount),
					Status: v2.ComplianceCheckStatus_INCONSISTENT,
				}.Build(),
				v2.ComplianceCheckStatusCount_builder{
					Count:  int32(resultCount.NotApplicableCount),
					Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
				}.Build(),
			},
		}.Build())
	}
	return convertedResults
}

func convertBenchmarks(incoming []*storage.ComplianceOperatorBenchmarkV2) []*v2.ComplianceBenchmark {
	var convertedBenchmarks []*v2.ComplianceBenchmark
	for _, benchmark := range incoming {
		cb := &v2.ComplianceBenchmark{}
		cb.SetName(benchmark.GetName())
		cb.SetVersion(benchmark.GetVersion())
		cb.SetDescription(benchmark.GetDescription())
		cb.SetProvider(benchmark.GetProvider())
		cb.SetShortName(benchmark.GetShortName())
		convertedBenchmarks = append(convertedBenchmarks, cb)
	}

	return convertedBenchmarks
}
