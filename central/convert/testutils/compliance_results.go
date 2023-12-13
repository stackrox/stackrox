package testutils

import (
	"testing"

	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
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
	complianceCheckID1 = uuid.NewV4().String()
	complianceCheckID2 = uuid.NewV4().String()
	complianceCheckID3 = uuid.NewV4().String()

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
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID1,
			CheckName:      complianceCheckName1,
			ClusterId:      fixtureconsts.Cluster1,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			ScanConfigName: scanConfigName1,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID1,
			CheckName:      complianceCheckName1,
			ClusterId:      fixtureconsts.Cluster2,
			ClusterName:    clusterName2,
			Status:         storage.ComplianceOperatorCheckResultV2_PASS,
			Description:    "description 1",
			Instructions:   "instructions 1",
			ScanConfigName: scanConfigName1,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID2,
			CheckName:      complianceCheckName2,
			ClusterId:      fixtureconsts.Cluster2,
			ClusterName:    clusterName2,
			Status:         storage.ComplianceOperatorCheckResultV2_PASS,
			Description:    "description 2",
			Instructions:   "instructions 2",
			ScanConfigName: scanConfigName1,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID2,
			CheckName:      complianceCheckName2,
			ClusterId:      fixtureconsts.Cluster1,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 2",
			Instructions:   "instructions 2",
			ScanConfigName: scanConfigName2,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      fixtureconsts.Cluster1,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      fixtureconsts.Cluster2,
			ClusterName:    clusterName2,
			Status:         storage.ComplianceOperatorCheckResultV2_FAIL,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      fixtureconsts.Cluster3,
			ClusterName:    clusterName3,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
		},
	}
}

// GetOneClusterComplianceStorageResults creates a set of mock check results for testing
func GetOneClusterComplianceStorageResults(_ *testing.T, clusterID string) []*storage.ComplianceOperatorCheckResultV2 {
	return []*storage.ComplianceOperatorCheckResultV2{
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID1,
			CheckName:      complianceCheckName1,
			ClusterId:      clusterID,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 1",
			Instructions:   "instructions 1",
			ScanConfigName: scanConfigName1,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID2,
			CheckName:      complianceCheckName2,
			ClusterId:      clusterID,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 2",
			Instructions:   "instructions 2",
			ScanConfigName: scanConfigName2,
		},
		{
			Id:             uuid.NewV4().String(),
			CheckId:        complianceCheckID3,
			CheckName:      complianceCheckName3,
			ClusterId:      clusterID,
			ClusterName:    clusterName1,
			Status:         storage.ComplianceOperatorCheckResultV2_INFO,
			Description:    "description 3",
			Instructions:   "instructions 3",
			ScanConfigName: scanConfigName3,
		},
	}
}

// GetConvertedComplianceResults retrieves results that match GetComplianceStorageResults
func GetConvertedComplianceResults(_ *testing.T) []*v2.ComplianceScanResult {
	return []*v2.ComplianceScanResult{
		{
			ScanName: scanConfigName1,
			CheckResults: []*v2.ComplianceCheckResult{
				{
					CheckId:   complianceCheckID1,
					CheckName: complianceCheckName1,
					Clusters: []*v2.ComplianceCheckResult_ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster1,
								ClusterName: clusterName1,
							},
							Status: v2.ComplianceCheckStatus_INFO,
						},
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster2,
								ClusterName: clusterName2,
							},
							Status: v2.ComplianceCheckStatus_PASS,
						},
					},
					Description:  "description 1",
					Instructions: "instructions 1",
				},
				{
					CheckId:   complianceCheckID2,
					CheckName: complianceCheckName2,
					Clusters: []*v2.ComplianceCheckResult_ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster2,
								ClusterName: clusterName2,
							},
							Status: v2.ComplianceCheckStatus_PASS,
						},
					},
					Description:  "description 2",
					Instructions: "instructions 2",
				},
			},
		},
		{
			ScanName: scanConfigName2,
			CheckResults: []*v2.ComplianceCheckResult{
				{
					CheckId:   complianceCheckID2,
					CheckName: complianceCheckName2,
					Clusters: []*v2.ComplianceCheckResult_ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster1,
								ClusterName: clusterName1,
							},
							Status: v2.ComplianceCheckStatus_INFO,
						},
					},
					Description:  "description 2",
					Instructions: "instructions 2",
				},
			},
		},
		{
			ScanName: scanConfigName3,
			CheckResults: []*v2.ComplianceCheckResult{
				{
					CheckId:   complianceCheckID3,
					CheckName: complianceCheckName3,
					Clusters: []*v2.ComplianceCheckResult_ClusterCheckStatus{
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster1,
								ClusterName: clusterName1,
							},
							Status: v2.ComplianceCheckStatus_INFO,
						},
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster2,
								ClusterName: clusterName2,
							},
							Status: v2.ComplianceCheckStatus_FAIL,
						},
						{
							Cluster: &v2.ComplianceScanCluster{
								ClusterId:   fixtureconsts.Cluster3,
								ClusterName: clusterName3,
							},
							Status: v2.ComplianceCheckStatus_INFO,
						},
					},
					Description:  "description 3",
					Instructions: "instructions 3",
				},
			},
		},
	}
}

// GetComplianceStorageCount returns mock data shaped like count query would return
func GetComplianceStorageCount(_ *testing.T, clusterID string) *datastore.ResourceCountByResultByCluster {
	return &datastore.ResourceCountByResultByCluster{
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

// GetComplianceClusterV2Count returns V2 count matching that from GetComplianceStorageCount
func GetComplianceClusterV2Count(_ *testing.T, clusterID string) *v2.ComplianceClusterScanStats {
	return &v2.ComplianceClusterScanStats{
		Cluster: &v2.ComplianceScanCluster{
			ClusterId:   clusterID,
			ClusterName: clusterName1,
		},
		ScanStats: &v2.ComplianceScanStatsShim{
			ScanName: scanConfigName1,
			CheckStats: []*v2.ComplianceScanStatsShim_ComplianceCheckStatusCount{
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
