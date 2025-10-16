package helpers

import (
	"context"

	snapshotDS "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UpdateSnapshotOnError updates the state of a given snapshot to FAILURE
func UpdateSnapshotOnError(ctx context.Context, snapshot *storage.ComplianceOperatorReportSnapshotV2, err error, store snapshotDS.DataStore) error {
	if snapshot == nil {
		return nil
	}
	snapshot.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_FAILURE)
	snapshot.GetReportStatus().SetErrorMsg(err.Error())
	snapshot.GetReportStatus().SetCompletedAt(protocompat.TimestampNow())
	if dbErr := store.UpsertSnapshot(ctx, snapshot); dbErr != nil {
		return dbErr
	}
	return nil
}

func getLatestBindingError(status *storage.ComplianceOperatorStatus) string {
	conditions := status.GetConditions()
	for _, c := range conditions {
		// If this either an invalid or suspended condition, only then is this an error case
		if c.GetType() == "READY" && c.GetStatus() == "False" {
			return c.GetMessage()
		}
	}
	return ""
}

// ConvertScanConfigurationToReportData converts a given ComplianceOperatorScanConfigurationV2 to a ComplianceOperatorReportSnapshotV2_ScanConfig
func ConvertScanConfigurationToReportData(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, scanConfigDS scanConfigDS.DataStore, suiteDS suiteDS.DataStore, bindingsDS bindingsDS.DataStore) (*storage.ComplianceOperatorReportData, error) {
	clusters, err := scanConfigDS.GetScanConfigClusterStatus(ctx, scanConfig.GetId())
	if err != nil {
		return nil, err
	}

	suiteClusters, err := suiteDS.GetSuites(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorSuiteName, scanConfig.GetScanConfigName()).ProtoQuery())
	if err != nil {
		return nil, err
	}

	var lastExecutedTime *timestamppb.Timestamp
	clusterToSuiteMap := make(map[string]*storage.ComplianceOperatorReportData_SuiteStatus, len(suiteClusters))
	for _, suite := range suiteClusters {
		status := &storage.ComplianceOperatorReportData_SuiteStatus{}
		status.SetPhase(suite.GetStatus().GetPhase())
		status.SetResult(suite.GetStatus().GetResult())
		status.SetErrorMessage(suite.GetStatus().GetErrorMessage())

		conditions := suite.GetStatus().GetConditions()
		for _, c := range conditions {
			if status.GetLastTransitionTime() == nil || protoutils.After(c.GetLastTransitionTime(), status.GetLastTransitionTime()) {
				status.SetLastTransitionTime(c.GetLastTransitionTime())
			}
		}

		if suite.GetStatus().GetPhase() == "DONE" && (lastExecutedTime == nil || protoutils.After(status.GetLastTransitionTime(), lastExecutedTime)) {
			lastExecutedTime = status.GetLastTransitionTime()
		}

		clusterToSuiteMap[suite.GetClusterId()] = status
	}

	cord := &storage.ComplianceOperatorReportData{}
	cord.SetScanConfiguration(scanConfig)
	cord.SetClusterStatus(func() []*storage.ComplianceOperatorReportData_ClusterStatus {
		clusterStatutes := make([]*storage.ComplianceOperatorReportData_ClusterStatus, 0, len(clusters))
		var clusterErrors []string
		for _, cluster := range clusters {
			bindings, err := bindingsDS.GetScanSettingBindings(ctx, search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfig.GetScanConfigName()).
				AddExactMatches(search.ClusterID, cluster.GetClusterId()).ProtoQuery())
			if err != nil {
				continue
			}

			if len(bindings) != 0 {
				bindingError := getLatestBindingError(bindings[0].GetStatus())
				if bindingError != "" {
					clusterErrors = append(clusterErrors, bindingError)
				}
			}

			cc := &storage.ComplianceOperatorReportData_ClusterStatus{}
			cc.SetClusterId(cluster.GetClusterId())
			cc.SetClusterName(cluster.GetClusterName())
			cc.SetErrors(append(clusterErrors, cluster.GetErrors()...))
			cc.SetSuiteStatus(clusterToSuiteMap[cluster.GetClusterId()])
			clusterStatutes = append(clusterStatutes, cc)
		}
		return clusterStatutes
	}())
	cord.SetLastExecutedTime(lastExecutedTime)
	return cord, nil
}
