package testutils

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	passCount          = 3
	failCount          = 1
	errorCount         = 2
	inconsistentCount  = 1
	infoCount          = 6
	manualCount        = 23
	notApplicableCount = 12
)

var (
	complianceCheckID1  = uuid.NewV4().String()
	complianceCheckID2  = uuid.NewV4().String()
	complianceCheckID3  = uuid.NewV4().String()
	complianceCheckUID1 = uuid.NewV4().String()
	complianceCheckUID2 = uuid.NewV4().String()
	complianceCheckUID3 = uuid.NewV4().String()
	complianceCheckUID4 = uuid.NewV4().String()
	complianceCheckUID5 = uuid.NewV4().String()
	complianceCheckUID6 = uuid.NewV4().String()
	complianceCheckUID7 = uuid.NewV4().String()

	complianceCheckName1 = "check1"
	complianceCheckName2 = "check2"
	complianceCheckName3 = "check3"

	clusterName1 = "cluster1"
	clusterName2 = "cluster2"
	clusterName3 = "cluster3"

	scanConfigName1 = "scanConfig1"
	scanConfigName2 = "scanConfig2"
	scanConfigName3 = "scanConfig3"
)

// GetComplianceStorageResults creates a set of mock check results for testing
func GetComplianceStorageResults(_ *testing.T) []*storage.ComplianceOperatorCheckResultV2 {
	return []*storage.ComplianceOperatorCheckResultV2{
		{
			Id:             complianceCheckUID1,
			CheckId:        complianceCheckID1,
			CheckName:      complianceCheckName1,
			ClusterId:      fixtureconsts.Cluster1,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			ScanConfigName: scanConfigName1,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID2,
			CheckId:        complianceCheckID1,
			CheckName:      complianceCheckName1,
			ClusterId:      fixtureconsts.Cluster2,
			ClusterName:    clusterName2,
			Status:         storage.ComplianceOperatorCheckResultV2_PASS,
			Description:    "description 1",
			Instructions:   "instructions 1",
			ScanConfigName: scanConfigName1,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID3,
			CheckId:        complianceCheckID2,
			CheckName:      complianceCheckName2,
			ClusterId:      fixtureconsts.Cluster2,
			ClusterName:    clusterName2,
			Status:         storage.ComplianceOperatorCheckResultV2_PASS,
			Description:    "description 2",
			Instructions:   "instructions 2",
			ScanConfigName: scanConfigName1,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID4,
			CheckId:        complianceCheckID2,
			CheckName:      complianceCheckName2,
			ClusterId:      fixtureconsts.Cluster1,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 2",
			Instructions:   "instructions 2",
			ScanConfigName: scanConfigName2,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID5,
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      fixtureconsts.Cluster1,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID6,
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      fixtureconsts.Cluster2,
			ClusterName:    clusterName2,
			Status:         storage.ComplianceOperatorCheckResultV2_FAIL,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID7,
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      fixtureconsts.Cluster3,
			ClusterName:    clusterName3,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
			ScanRefId:      "test-ref",
		},
	}
}

// GetOneClusterComplianceStorageResults creates a set of mock check results for testing
func GetOneClusterComplianceStorageResults(_ *testing.T, clusterID string) []*storage.ComplianceOperatorCheckResultV2 {
	return []*storage.ComplianceOperatorCheckResultV2{
		{
			Id:             complianceCheckUID1,
			CheckId:        complianceCheckID1,
			CheckName:      complianceCheckName1,
			ClusterId:      clusterID,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			ScanConfigName: scanConfigName1,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID2,
			CheckId:        complianceCheckID2,
			CheckName:      complianceCheckName2,
			ClusterId:      clusterID,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 2",
			Instructions:   "instructions 2",
			ScanConfigName: scanConfigName2,
			ScanRefId:      "test-ref",
		},
		{
			Id:             complianceCheckUID3,
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      clusterID,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
			ScanRefId:      "test-ref",
		},
	}
}

// GetConvertedComplianceResults retrieves results that match GetComplianceStorageResults
func GetConvertedComplianceResults(_ *testing.T) []*v2.ComplianceScanResult {
	return []*v2.ComplianceScanResult{
		{
			ScanName:     scanConfigName1,
			ScanConfigId: scanConfigName1,
			CheckResults: []*v2.ComplianceClusterCheckStatus{
				{
					CheckId:   complianceCheckID1,
					CheckName: complianceCheckName1,
					Clusters: []*v2.ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster1,
								ClusterName: clusterName1,
							},
							CheckUid: complianceCheckUID1,
							Status:   v2.ComplianceCheckStatus_INFO,
						},
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster2,
								ClusterName: clusterName2,
							},
							CheckUid: complianceCheckUID2,
							Status:   v2.ComplianceCheckStatus_PASS,
						},
					},
					Description:  "description 1",
					Instructions: "instructions 1",
				},
				{
					CheckId:   complianceCheckID2,
					CheckName: complianceCheckName2,
					Clusters: []*v2.ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster2,
								ClusterName: clusterName2,
							},
							CheckUid: complianceCheckUID3,
							Status:   v2.ComplianceCheckStatus_PASS,
						},
					},
					Description:  "description 2",
					Instructions: "instructions 2",
				},
			},
		},
		{
			ScanName:     scanConfigName2,
			ScanConfigId: scanConfigName2,
			CheckResults: []*v2.ComplianceClusterCheckStatus{
				{
					CheckId:   complianceCheckID2,
					CheckName: complianceCheckName2,
					Clusters: []*v2.ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster1,
								ClusterName: clusterName1,
							},
							CheckUid: complianceCheckUID4,
							Status:   v2.ComplianceCheckStatus_INFO,
						},
					},
					Description:  "description 2",
					Instructions: "instructions 2",
				},
			},
		},
		{
			ScanName:     scanConfigName3,
			ScanConfigId: scanConfigName3,
			CheckResults: []*v2.ComplianceClusterCheckStatus{
				{
					CheckId:   complianceCheckID3,
					CheckName: complianceCheckName3,
					Clusters: []*v2.ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster1,
								ClusterName: clusterName1,
							},
							CheckUid: complianceCheckUID5,
							Status:   v2.ComplianceCheckStatus_INFO,
						},
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster2,
								ClusterName: clusterName2,
							},
							CheckUid: complianceCheckUID6,
							Status:   v2.ComplianceCheckStatus_FAIL,
						},
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster3,
								ClusterName: clusterName3,
							},
							CheckUid: complianceCheckUID7,
							Status:   v2.ComplianceCheckStatus_INFO,
						},
					},
					Description:  "description 3",
					Instructions: "instructions 3",
				},
			},
		},
	}
}

// GetConvertedComplianceData retrieves results that match GetComplianceStorageResults
func GetConvertedComplianceData(_ *testing.T) []*v2.ComplianceCheckData {
	return []*v2.ComplianceCheckData{
		{
			ScanName:  scanConfigName1,
			ClusterId: fixtureconsts.Cluster1,
			Result: &v2.ComplianceCheckResult{
				CheckId:      complianceCheckID1,
				CheckName:    complianceCheckName1,
				CheckUid:     complianceCheckUID1,
				Description:  "description 1",
				Instructions: "instructions 1",
				Status:       v2.ComplianceCheckStatus_INFO,
				RuleName:     "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
			},
		},
		{
			ScanName:  scanConfigName1,
			ClusterId: fixtureconsts.Cluster2,
			Result: &v2.ComplianceCheckResult{
				CheckId:      complianceCheckID1,
				CheckName:    complianceCheckName1,
				CheckUid:     complianceCheckUID2,
				Description:  "description 1",
				Instructions: "instructions 1",
				Status:       v2.ComplianceCheckStatus_PASS,
				RuleName:     "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
			},
		},
		{
			ScanName:  scanConfigName1,
			ClusterId: fixtureconsts.Cluster2,
			Result: &v2.ComplianceCheckResult{
				CheckId:      complianceCheckID2,
				CheckName:    complianceCheckName2,
				CheckUid:     complianceCheckUID3,
				Description:  "description 2",
				Instructions: "instructions 2",
				Status:       v2.ComplianceCheckStatus_PASS,
				RuleName:     "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
			},
		},
		{
			ScanName:  scanConfigName2,
			ClusterId: fixtureconsts.Cluster1,
			Result: &v2.ComplianceCheckResult{
				CheckId:      complianceCheckID2,
				CheckName:    complianceCheckName2,
				CheckUid:     complianceCheckUID4,
				Description:  "description 2",
				Instructions: "instructions 2",
				Status:       v2.ComplianceCheckStatus_INFO,
				RuleName:     "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
			},
		},
		{
			ScanName:  scanConfigName3,
			ClusterId: fixtureconsts.Cluster1,
			Result: &v2.ComplianceCheckResult{
				CheckId:      complianceCheckID3,
				CheckName:    complianceCheckName3,
				CheckUid:     complianceCheckUID5,
				Description:  "description 3",
				Instructions: "instructions 3",
				Status:       v2.ComplianceCheckStatus_INFO,
				RuleName:     "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
			},
		},
		{
			ScanName:  scanConfigName3,
			ClusterId: fixtureconsts.Cluster2,
			Result: &v2.ComplianceCheckResult{
				CheckId:      complianceCheckID3,
				CheckName:    complianceCheckName3,
				CheckUid:     complianceCheckUID6,
				Description:  "description 3",
				Instructions: "instructions 3",
				Status:       v2.ComplianceCheckStatus_FAIL,
				RuleName:     "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
			},
		},
		{
			ScanName:  scanConfigName3,
			ClusterId: fixtureconsts.Cluster3,
			Result: &v2.ComplianceCheckResult{
				CheckId:      complianceCheckID3,
				CheckName:    complianceCheckName3,
				CheckUid:     complianceCheckUID7,
				Description:  "description 3",
				Instructions: "instructions 3",
				Status:       v2.ComplianceCheckStatus_INFO,
				RuleName:     "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
			},
		},
	}
}

// GetComplianceStorageProfileScanCount returns mock data shaped like count query would return
func GetComplianceStorageProfileScanCount(_ *testing.T, profileName string) *datastore.ResourceResultCountByProfile {
	return &datastore.ResourceResultCountByProfile{
		PassCount:          passCount,
		FailCount:          failCount,
		ErrorCount:         errorCount,
		InconsistentCount:  inconsistentCount,
		InfoCount:          infoCount,
		ManualCount:        manualCount,
		NotApplicableCount: notApplicableCount,
		ProfileName:        profileName,
	}
}

// GetComplianceStorageProfileResults returns mock data shaped like count query would return
func GetComplianceStorageProfileResults(_ *testing.T, profileName string) *datastore.ResourceResultsByProfile {
	return &datastore.ResourceResultsByProfile{
		PassCount:          passCount,
		FailCount:          failCount,
		ErrorCount:         errorCount,
		InconsistentCount:  inconsistentCount,
		InfoCount:          infoCount,
		ManualCount:        manualCount,
		NotApplicableCount: notApplicableCount,
		ProfileName:        profileName,
		CheckName:          "check-name",
		CheckRationale:     "",
		RuleName:           "rule-name",
	}
}

// GetComplianceStorageClusterScanCount returns mock data shaped like count query would return
func GetComplianceStorageClusterScanCount(_ *testing.T, clusterID string) *datastore.ResourceResultCountByClusterScan {
	return &datastore.ResourceResultCountByClusterScan{
		PassCount:          passCount,
		FailCount:          failCount,
		ErrorCount:         errorCount,
		InconsistentCount:  inconsistentCount,
		InfoCount:          infoCount,
		ManualCount:        manualCount,
		NotApplicableCount: notApplicableCount,
		ClusterName:        clusterName1,
		ClusterID:          clusterID,
		ScanConfigName:     scanConfigName1,
	}
}

// GetComplianceClusterScanV2Count returns V2 count matching that from GetComplianceStorageClusterScanCount
func GetComplianceClusterScanV2Count(_ *testing.T, clusterID string) *v2.ComplianceClusterScanStats {
	return &v2.ComplianceClusterScanStats{
		Cluster: &v2.ComplianceScanCluster{
			ClusterId:   clusterID,
			ClusterName: clusterName1,
		},
		ScanStats: &v2.ComplianceScanStatsShim{
			ScanName:     scanConfigName1,
			ScanConfigId: scanConfigName1,
			CheckStats: []*v2.ComplianceCheckStatusCount{
				{
					Count:  failCount,
					Status: v2.ComplianceCheckStatus_FAIL,
				},
				{
					Count:  infoCount,
					Status: v2.ComplianceCheckStatus_INFO,
				},
				{
					Count:  passCount,
					Status: v2.ComplianceCheckStatus_PASS,
				},
				{
					Count:  errorCount,
					Status: v2.ComplianceCheckStatus_ERROR,
				},
				{
					Count:  manualCount,
					Status: v2.ComplianceCheckStatus_MANUAL,
				},
				{
					Count:  inconsistentCount,
					Status: v2.ComplianceCheckStatus_INCONSISTENT,
				},
				{
					Count:  notApplicableCount,
					Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
				},
			},
		},
	}
}

// GetComplianceProfileScanV2Count returns V2 count matching that from GetComplianceStorageProfileScanCount
func GetComplianceProfileScanV2Count(_ *testing.T, profileName string, benchmarks []*v2.ComplianceBenchmark) *v2.ComplianceProfileScanStats {
	return &v2.ComplianceProfileScanStats{
		ProfileName: profileName,
		Title:       fmt.Sprintf("test_title_%s", profileName),
		Version:     fmt.Sprintf("test_version_%s", profileName),
		Benchmarks:  benchmarks,
		CheckStats: []*v2.ComplianceCheckStatusCount{
			{
				Count:  failCount,
				Status: v2.ComplianceCheckStatus_FAIL,
			},
			{
				Count:  infoCount,
				Status: v2.ComplianceCheckStatus_INFO,
			},
			{
				Count:  passCount,
				Status: v2.ComplianceCheckStatus_PASS,
			},
			{
				Count:  errorCount,
				Status: v2.ComplianceCheckStatus_ERROR,
			},
			{
				Count:  manualCount,
				Status: v2.ComplianceCheckStatus_MANUAL,
			},
			{
				Count:  inconsistentCount,
				Status: v2.ComplianceCheckStatus_INCONSISTENT,
			},
			{
				Count:  notApplicableCount,
				Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
			},
		},
	}
}

// GetComplianceProfileResultsV2 returns V2 count matching that from GetComplianceStorageProfileResults
func GetComplianceProfileResultsV2(_ *testing.T, profileName string) *v2.ListComplianceProfileResults {
	return &v2.ListComplianceProfileResults{
		ProfileName: profileName,
		TotalCount:  1,
		ProfileResults: []*v2.ComplianceCheckResultStatusCount{
			{
				CheckName: "check-name",
				Rationale: "",
				RuleName:  "rule-name",
				Controls: []*v2.ComplianceControl{
					{Standard: "OCP-CIS", Control: "1.2.2"},
					{Standard: "OCP-CIS", Control: "1.3.3"},
					{Standard: "OCP-CIS", Control: "1.4.4"},
				},
				CheckStats: []*v2.ComplianceCheckStatusCount{
					{
						Count:  failCount,
						Status: v2.ComplianceCheckStatus_FAIL,
					},
					{
						Count:  infoCount,
						Status: v2.ComplianceCheckStatus_INFO,
					},
					{
						Count:  passCount,
						Status: v2.ComplianceCheckStatus_PASS,
					},
					{
						Count:  errorCount,
						Status: v2.ComplianceCheckStatus_ERROR,
					},
					{
						Count:  manualCount,
						Status: v2.ComplianceCheckStatus_MANUAL,
					},
					{
						Count:  inconsistentCount,
						Status: v2.ComplianceCheckStatus_INCONSISTENT,
					},
					{
						Count:  notApplicableCount,
						Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
					},
				},
			},
		},
	}
}

// GetComplianceStorageClusterCount returns mock data shaped like count query would return
func GetComplianceStorageClusterCount(_ *testing.T, clusterID string, lastScanTime *time.Time) *datastore.ResultStatusCountByCluster {
	return &datastore.ResultStatusCountByCluster{
		PassCount:          passCount,
		FailCount:          failCount,
		ErrorCount:         errorCount,
		InconsistentCount:  inconsistentCount,
		InfoCount:          infoCount,
		ManualCount:        manualCount,
		NotApplicableCount: notApplicableCount,
		ClusterName:        clusterName1,
		ClusterID:          clusterID,
		LastScanTime:       lastScanTime,
	}
}

// GetComplianceClusterV2Count returns V2 count matching that from GetComplianceStorageClusterCount
func GetComplianceClusterV2Count(_ *testing.T, clusterID string, lastScanTime *types.Timestamp) *v2.ComplianceClusterOverallStats {
	return &v2.ComplianceClusterOverallStats{
		Cluster: &v2.ComplianceScanCluster{
			ClusterId:   clusterID,
			ClusterName: clusterName1,
		},
		ClusterErrors: []string{"test error"},
		CheckStats: []*v2.ComplianceCheckStatusCount{
			{
				Count:  failCount,
				Status: v2.ComplianceCheckStatus_FAIL,
			},
			{
				Count:  infoCount,
				Status: v2.ComplianceCheckStatus_INFO,
			},
			{
				Count:  passCount,
				Status: v2.ComplianceCheckStatus_PASS,
			},
			{
				Count:  errorCount,
				Status: v2.ComplianceCheckStatus_ERROR,
			},
			{
				Count:  manualCount,
				Status: v2.ComplianceCheckStatus_MANUAL,
			},
			{
				Count:  inconsistentCount,
				Status: v2.ComplianceCheckStatus_INCONSISTENT,
			},
			{
				Count:  notApplicableCount,
				Status: v2.ComplianceCheckStatus_NOT_APPLICABLE,
			},
		},
		LastScanTime: lastScanTime,
	}
}

// GetComplianceStorageResult creates a mock check results for testing
func GetComplianceStorageResult(_ *testing.T) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             complianceCheckUID1,
		CheckId:        complianceCheckID1,
		CheckName:      complianceCheckName1,
		ClusterId:      fixtureconsts.Cluster1,
		ClusterName:    clusterName1,
		Status:         storage.ComplianceOperatorCheckResultV2_INFO,
		Description:    "description 1",
		Instructions:   "instructions 1",
		ScanConfigName: scanConfigName1,
		RuleRefId:      "test-ref-id",
		ScanRefId:      "test-ref",
	}
}

// GetConvertedComplianceResult retrieves results that match GetComplianceStorageResult
func GetConvertedComplianceResult(_ *testing.T, lastScanTime *types.Timestamp) *v2.ComplianceClusterCheckStatus {
	return &v2.ComplianceClusterCheckStatus{
		CheckId:   complianceCheckID1,
		CheckName: complianceCheckName1,
		Clusters: []*v2.ClusterCheckStatus{
			{
				Cluster: &v2.ComplianceScanCluster{
					ClusterId:   fixtureconsts.Cluster1,
					ClusterName: clusterName1,
				},
				Status:       v2.ComplianceCheckStatus_INFO,
				CheckUid:     complianceCheckUID1,
				LastScanTime: lastScanTime,
			},
		},
		Description:  "description 1",
		Instructions: "instructions 1",
		Controls: []*v2.ComplianceControl{
			{Standard: "OCP-CIS", Control: "1.2.2"},
			{Standard: "OCP-CIS", Control: "1.3.3"},
			{Standard: "OCP-CIS", Control: "1.4.4"},
		},
	}
}

// GetConvertedCheckResult retrieves results
func GetConvertedCheckResult(_ *testing.T) []*v2.ComplianceCheckResult {
	return []*v2.ComplianceCheckResult{
		{
			CheckId:      complianceCheckID1,
			CheckName:    complianceCheckName1,
			Description:  "description 1",
			Instructions: "instructions 1",
			Status:       v2.ComplianceCheckStatus_INFO,
			CheckUid:     complianceCheckUID1,
			RuleName:     "rule-name",
			Controls: []*v2.ComplianceControl{
				{Standard: "OCP-CIS", Control: "1.2.2"},
				{Standard: "OCP-CIS", Control: "1.3.3"},
				{Standard: "OCP-CIS", Control: "1.4.4"},
			},
		},
	}
}
