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
	snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_FAILURE
	snapshot.GetReportStatus().ErrorMsg = err.Error()
	snapshot.GetReportStatus().CompletedAt = protocompat.TimestampNow()
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

// ConvertScanConfigurationToReportSnapshotScanConfig converts a given ComplianceOperatorScanConfigurationV2 to a ComplianceOperatorReportSnapshotV2_ScanConfig
func ConvertScanConfigurationToReportSnapshotScanConfig(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, scanConfigDS scanConfigDS.DataStore, suiteDS suiteDS.DataStore, bindingsDS bindingsDS.DataStore) (*storage.ComplianceOperatorReportSnapshotV2_ScanConfig, error) {
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
	clusterToSuiteMap := make(map[string]*storage.ComplianceOperatorReportSnapshotV2_SuiteStatus, len(suiteClusters))
	for _, suite := range suiteClusters {
		status := &storage.ComplianceOperatorReportSnapshotV2_SuiteStatus{
			Phase:        suite.GetStatus().GetPhase(),
			Result:       suite.GetStatus().GetResult(),
			ErrorMessage: suite.GetStatus().GetErrorMessage(),
		}
		conditions := suite.GetStatus().GetConditions()
		for _, c := range conditions {
			if status.GetLastTransitionTime() == nil || protoutils.After(c.GetLastTransitionTime(), status.GetLastTransitionTime()) {
				status.LastTransitionTime = c.LastTransitionTime
			}
		}
		if suite.GetStatus().GetPhase() == "DONE" && (lastExecutedTime == nil || protoutils.After(status.GetLastTransitionTime(), lastExecutedTime)) {
			lastExecutedTime = status.GetLastTransitionTime()
		}
		clusterToSuiteMap[suite.GetClusterId()] = status
	}
	return &storage.ComplianceOperatorReportSnapshotV2_ScanConfig{
		Id:       scanConfig.GetId(),
		ScanName: scanConfig.GetScanConfigName(),
		ScanConfig: &storage.ComplianceOperatorReportSnapshotV2_BaseSettings{
			OneTimeScan: scanConfig.GetOneTimeScan(),
			Profiles: func() []string {
				ret := make([]string, 0, len(scanConfig.GetProfiles()))
				for _, profile := range scanConfig.GetProfiles() {
					ret = append(ret, profile.GetProfileName())
				}
				return ret
			}(),
			ScanSchedule: scanConfig.GetSchedule(),
			Description:  scanConfig.GetDescription(),
			Notifiers: func() []*storage.ComplianceOperatorReportSnapshotV2_NotifierInfo {
				ret := make([]*storage.ComplianceOperatorReportSnapshotV2_NotifierInfo, 0, len(scanConfig.GetNotifiers()))
				for _, notifier := range scanConfig.GetNotifiers() {
					ret = append(ret, &storage.ComplianceOperatorReportSnapshotV2_NotifierInfo{
						NotifierConfig: &storage.ComplianceOperatorReportSnapshotV2_NotifierInfo_EmailConfig{
							EmailConfig: &storage.EmailNotifierConfiguration{
								NotifierId:    notifier.GetId(),
								MailingLists:  notifier.GetEmailConfig().GetMailingLists(),
								CustomSubject: notifier.GetEmailConfig().GetCustomSubject(),
								CustomBody:    notifier.GetEmailConfig().GetCustomBody(),
							},
						},
					})
				}
				return ret
			}(),
		},
		ClusterStatus: func() []*storage.ComplianceOperatorReportSnapshotV2_ClusterStatus {
			clusterStatutes := make([]*storage.ComplianceOperatorReportSnapshotV2_ClusterStatus, 0, len(clusters))
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
				clusterStatutes = append(clusterStatutes, &storage.ComplianceOperatorReportSnapshotV2_ClusterStatus{
					ClusterId:   cluster.GetClusterId(),
					ClusterName: cluster.GetClusterName(),
					Errors:      append(clusterErrors, cluster.GetErrors()...),
					SuiteStatus: clusterToSuiteMap[cluster.GetClusterId()],
				})
			}
			return clusterStatutes
		}(),
		CreatedTime:      scanConfig.GetCreatedTime(),
		LastUpdatedTime:  scanConfig.GetLastUpdatedTime(),
		ModifiedBy:       scanConfig.GetModifiedBy(),
		LastExecutedTime: lastExecutedTime,
	}, nil
}
