package service

import (
	"context"

	"github.com/pkg/errors"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	types "github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
)

/*
storage type to apiV2 type conversions
*/

const (
	suiteComplete = "DONE"
)

var (
	v2IntervalTypeToStorage = map[v2.Schedule_IntervalType]storage.Schedule_IntervalType{
		v2.Schedule_UNSET:   storage.Schedule_UNSET,
		v2.Schedule_WEEKLY:  storage.Schedule_WEEKLY,
		v2.Schedule_MONTHLY: storage.Schedule_MONTHLY,
		v2.Schedule_DAILY:   storage.Schedule_DAILY,
	}

	storageIntervalTypeToV2 = map[storage.Schedule_IntervalType]v2.Schedule_IntervalType{
		storage.Schedule_UNSET:   v2.Schedule_UNSET,
		storage.Schedule_DAILY:   v2.Schedule_DAILY,
		storage.Schedule_WEEKLY:  v2.Schedule_WEEKLY,
		storage.Schedule_MONTHLY: v2.Schedule_MONTHLY,
	}
)

func convertStorageScanConfigToV2(ctx context.Context, scanConfig *storage.ComplianceOperatorScanConfigurationV2, configDS complianceDS.DataStore) (*v2.ComplianceScanConfiguration, error) {
	if scanConfig == nil {
		return nil, nil
	}

	scanClusters, err := configDS.GetScanConfigClusterStatus(ctx, scanConfig.GetId())
	if err != nil {
		return nil, err
	}

	clusters := make([]string, 0, len(scanClusters))
	for _, cluster := range scanClusters {
		clusters = append(clusters, cluster.GetClusterId())
	}

	profiles := make([]string, 0, len(scanConfig.GetProfiles()))
	for _, profile := range scanConfig.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}

	return &v2.ComplianceScanConfiguration{
		Id:       scanConfig.GetId(),
		ScanName: scanConfig.GetScanConfigName(),
		Clusters: clusters,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  scanConfig.GetOneTimeScan(),
			ScanSchedule: convertProtoScheduleToV2(scanConfig.GetSchedule()),
			Profiles:     profiles,
			Description:  scanConfig.GetDescription(),
		},
	}, nil
}

func convertV2NotifierConfigToProto(notifier *v2.NotifierConfiguration) *storage.NotifierConfiguration {
	if notifier == nil {
		return nil
	}

	ret := &storage.NotifierConfiguration{
		Ref: &storage.NotifierConfiguration_Id{
			Id: notifier.GetEmailConfig().GetNotifierId(),
		},
	}

	if emailConfig := notifier.GetEmailConfig(); emailConfig != nil {
		ret.NotifierConfig = &storage.NotifierConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				MailingLists:  emailConfig.GetMailingLists(),
				CustomSubject: emailConfig.GetCustomSubject(),
				CustomBody:    emailConfig.GetCustomBody(),
			},
		}
	}
	return ret
}

// convertProtoNotifierConfigToV2 converts storage.NotifierConfiguration to v2.NotifierConfiguration
func convertProtoNotifierConfigToV2(notifierConfig *storage.NotifierConfiguration, notifierName string) (*v2.NotifierConfiguration, error) {
	if notifierConfig == nil {
		return nil, nil
	}

	if notifierConfig.GetEmailConfig() == nil {
		return nil, errors.New("Email notifier is not configured")
	}

	return &v2.NotifierConfiguration{
		NotifierName: notifierName,
		NotifierConfig: &v2.NotifierConfiguration_EmailConfig{
			EmailConfig: &v2.EmailNotifierConfiguration{
				NotifierId:    notifierConfig.GetId(),
				MailingLists:  notifierConfig.GetEmailConfig().GetMailingLists(),
				CustomSubject: notifierConfig.GetEmailConfig().GetCustomSubject(),
				CustomBody:    notifierConfig.GetEmailConfig().GetCustomBody(),
			},
		},
	}, nil
}

func convertV2ScanConfigToStorage(ctx context.Context, scanConfig *v2.ComplianceScanConfiguration) *storage.ComplianceOperatorScanConfigurationV2 {
	if scanConfig == nil {
		return nil
	}

	profiles := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName, 0, len(scanConfig.GetScanConfig().GetProfiles()))
	for _, profile := range scanConfig.GetScanConfig().GetProfiles() {
		profiles = append(profiles, &storage.ComplianceOperatorScanConfigurationV2_ProfileName{
			ProfileName: profile,
		})
	}

	clusters := make([]*storage.ComplianceOperatorScanConfigurationV2_Cluster, 0, len(scanConfig.GetClusters()))
	for _, cluster := range scanConfig.GetClusters() {
		clusters = append(clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{
			ClusterId: cluster,
		})
	}

	notifiers := []*storage.NotifierConfiguration{}
	for _, notifier := range scanConfig.GetScanConfig().GetNotifiers() {
		notifierStorage := convertV2NotifierConfigToProto(notifier)
		if notifierStorage != nil {
			notifiers = append(notifiers, notifierStorage)
		}

	}

	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:                     scanConfig.GetId(),
		ScanConfigName:         scanConfig.GetScanName(),
		AutoApplyRemediations:  false,
		AutoUpdateRemediations: false,
		OneTimeScan:            false,
		StrictNodeScan:         false,
		Schedule:               ConvertV2ScheduleToProto(scanConfig.GetScanConfig().GetScanSchedule()),
		Profiles:               profiles,
		ModifiedBy:             authn.UserFromContext(ctx),
		Description:            scanConfig.GetScanConfig().GetDescription(),
		Clusters:               clusters,
		Notifiers:              notifiers,
	}
}

// ConvertV2ScheduleToProto converts v2.Schedule to storage.Schedule. Does not validate v2.Schedule
func ConvertV2ScheduleToProto(schedule *v2.Schedule) *storage.Schedule {
	if schedule == nil {
		return nil
	}

	ret := &storage.Schedule{
		IntervalType: v2IntervalTypeToStorage[schedule.GetIntervalType()],
		Hour:         schedule.GetHour(),
		Minute:       schedule.GetMinute(),
	}
	switch schedule.Interval.(type) {
	case *v2.Schedule_DaysOfWeek_:
		ret.Interval = &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{Days: schedule.GetDaysOfWeek().GetDays()},
		}
	case *v2.Schedule_DaysOfMonth_:
		ret.Interval = &storage.Schedule_DaysOfMonth_{
			DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}

// convertProtoScheduleToV2 converts storage.Schedule to v2.Schedule. Does not validate storage.Schedule
func convertProtoScheduleToV2(schedule *storage.Schedule) *v2.Schedule {
	if schedule == nil {
		return nil
	}
	ret := &v2.Schedule{
		IntervalType: storageIntervalTypeToV2[schedule.GetIntervalType()],
		Hour:         schedule.GetHour(),
		Minute:       schedule.GetMinute(),
	}

	switch schedule.Interval.(type) {
	case *storage.Schedule_DaysOfWeek_:
		ret.Interval = &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{Days: schedule.GetDaysOfWeek().GetDays()},
		}
	case *storage.Schedule_Weekly:
		ret.Interval = &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{Days: schedule.GetDaysOfWeek().GetDays()},
		}
	case *storage.Schedule_DaysOfMonth_:
		ret.Interval = &v2.Schedule_DaysOfMonth_{
			DaysOfMonth: &v2.Schedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
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

func convertStorageScanConfigToV2ScanStatus(ctx context.Context,
	scanConfig *storage.ComplianceOperatorScanConfigurationV2, configDS complianceDS.DataStore,
	bindingsDS bindingsDS.DataStore, suiteDS suiteDS.DataStore, notifierDS notifierDS.DataStore) (*v2.ComplianceScanConfigurationStatus, error) {
	if scanConfig == nil {
		return nil, nil
	}

	notifiers := []*v2.NotifierConfiguration{}
	for _, notifierConfig := range scanConfig.GetNotifiers() {

		notifier, found, err := notifierDS.GetNotifier(ctx, notifierConfig.GetId())
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.Errorf("Notifier with ID %s no longer exists", notifierConfig.GetId())
		}
		notifierV2, err := convertProtoNotifierConfigToV2(notifierConfig, notifier.GetName())
		if err != nil {
			return nil, err
		}
		notifiers = append(notifiers, notifierV2)
	}

	scanClusters, err := configDS.GetScanConfigClusterStatus(ctx, scanConfig.GetId())
	if err != nil {
		return nil, err
	}

	profiles := make([]string, 0, len(scanConfig.GetProfiles()))
	for _, profile := range scanConfig.GetProfiles() {
		profiles = append(profiles, profile.GetProfileName())
	}

	var lastScanTime *types.Timestamp
	suiteClusters, err := suiteDS.GetSuites(ctx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorSuiteName, scanConfig.GetScanConfigName()).ProtoQuery())
	if err != nil {
		return nil, err
	}
	clusterToSuiteMap := make(map[string]*v2.ClusterScanStatus_SuiteStatus, len(suiteClusters))
	for _, suite := range suiteClusters {
		suiteStatus := &v2.ClusterScanStatus_SuiteStatus{
			Phase:        suite.GetStatus().GetPhase(),
			Result:       suite.GetStatus().GetResult(),
			ErrorMessage: suite.GetStatus().GetErrorMessage(),
		}
		conditions := suite.GetStatus().GetConditions()
		for _, c := range conditions {
			if suiteStatus.LastTransitionTime == nil || protoutils.After(c.LastTransitionTime, suiteStatus.LastTransitionTime) {
				suiteStatus.LastTransitionTime = c.LastTransitionTime
			}
		}

		// If the suite is complete, set the last scan time
		if suite.GetStatus().GetPhase() == suiteComplete && (lastScanTime == nil || protoutils.After(suiteStatus.LastTransitionTime, lastScanTime)) {
			lastScanTime = suiteStatus.LastTransitionTime
		}

		clusterToSuiteMap[suite.ClusterId] = suiteStatus
	}

	return &v2.ComplianceScanConfigurationStatus{
		Id:       scanConfig.GetId(),
		ScanName: scanConfig.GetScanConfigName(),
		ClusterStatus: func() []*v2.ClusterScanStatus {
			clusterStatuses := make([]*v2.ClusterScanStatus, 0, len(scanClusters))
			for _, cluster := range scanClusters {
				var errors []string
				bindings, err := bindingsDS.GetScanSettingBindings(ctx, search.NewQueryBuilder().
					AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfig.GetScanConfigName()).
					AddExactMatches(search.ClusterID, cluster.ClusterId).ProtoQuery())
				if err != nil {
					continue
				}

				// We may not have received any bindings from sensor
				if len(bindings) != 0 {
					bindingError := getLatestBindingError(bindings[0].Status)
					if bindingError != "" {
						errors = append(errors, bindingError)
					}
				}

				errors = append(errors, cluster.GetErrors()...)
				clusterStatuses = append(clusterStatuses, &v2.ClusterScanStatus{
					ClusterId:   cluster.GetClusterId(),
					ClusterName: cluster.GetClusterName(),
					Errors:      errors,
					SuiteStatus: clusterToSuiteMap[cluster.GetClusterId()],
				})
			}
			return clusterStatuses
		}(),
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  scanConfig.GetOneTimeScan(),
			ScanSchedule: convertProtoScheduleToV2(scanConfig.GetSchedule()),
			Profiles:     profiles,
			Description:  scanConfig.GetDescription(),
			Notifiers:    notifiers,
		},
		ModifiedBy: &v2.SlimUser{
			Id:   scanConfig.GetModifiedBy().GetId(),
			Name: scanConfig.GetModifiedBy().GetName(),
		},
		CreatedTime:      scanConfig.GetCreatedTime(),
		LastUpdatedTime:  scanConfig.GetLastUpdatedTime(),
		LastExecutedTime: lastScanTime,
	}, nil
}

func convertStorageScanConfigToV2ScanStatuses(ctx context.Context,
	scanConfigs []*storage.ComplianceOperatorScanConfigurationV2,
	configDS complianceDS.DataStore, bindingDS bindingsDS.DataStore, suiteDS suiteDS.DataStore, notifierDS notifierDS.DataStore) ([]*v2.ComplianceScanConfigurationStatus, error) {
	if scanConfigs == nil {
		return nil, nil
	}

	scanStatuses := make([]*v2.ComplianceScanConfigurationStatus, 0, len(scanConfigs))
	for _, scanConfig := range scanConfigs {
		converted, err := convertStorageScanConfigToV2ScanStatus(ctx, scanConfig, configDS, bindingDS, suiteDS, notifierDS)
		if err != nil {
			return nil, errors.Wrapf(err, "Error converting storage compliance operator scan configuration status with name %s to response", scanConfig.GetScanConfigName())
		}

		scanStatuses = append(scanStatuses, converted)
	}

	return scanStatuses, nil
}
