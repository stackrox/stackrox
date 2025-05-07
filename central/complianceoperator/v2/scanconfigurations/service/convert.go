package service

import (
	"context"

	"github.com/pkg/errors"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	bindingsDS "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore"
	suiteDS "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
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

	storageReportRunStateToV2 = map[storage.ComplianceOperatorReportStatus_RunState]v2.ComplianceReportStatus_RunState{
		storage.ComplianceOperatorReportStatus_WAITING:       v2.ComplianceReportStatus_WAITING,
		storage.ComplianceOperatorReportStatus_PREPARING:     v2.ComplianceReportStatus_PREPARING,
		storage.ComplianceOperatorReportStatus_GENERATED:     v2.ComplianceReportStatus_GENERATED,
		storage.ComplianceOperatorReportStatus_DELIVERED:     v2.ComplianceReportStatus_DELIVERED,
		storage.ComplianceOperatorReportStatus_FAILURE:       v2.ComplianceReportStatus_FAILURE,
		storage.ComplianceOperatorReportStatus_PARTIAL_ERROR: v2.ComplianceReportStatus_PARTIAL_ERROR,
	}

	storageReportRequestTypeToV2 = map[storage.ComplianceOperatorReportStatus_RunMethod]v2.ComplianceReportStatus_ReportMethod{
		storage.ComplianceOperatorReportStatus_ON_DEMAND: v2.ComplianceReportStatus_ON_DEMAND,
		storage.ComplianceOperatorReportStatus_SCHEDULED: v2.ComplianceReportStatus_SCHEDULED,
	}

	storageReportNotificationMethodToV2 = map[storage.ComplianceOperatorReportStatus_NotificationMethod]v2.NotificationMethod{
		storage.ComplianceOperatorReportStatus_EMAIL:    v2.NotificationMethod_EMAIL,
		storage.ComplianceOperatorReportStatus_DOWNLOAD: v2.NotificationMethod_DOWNLOAD,
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

func convertStorageReportDataToV2ScanStatus(ctx context.Context, reportData *storage.ComplianceOperatorReportData, notifierDS notifierDS.DataStore) (*v2.ComplianceScanConfigurationStatus, error) {
	if reportData == nil {
		return nil, nil
	}

	notifiers := make([]*v2.NotifierConfiguration, 0, len(reportData.GetScanConfiguration().GetNotifiers()))
	for _, notifierConfig := range reportData.GetScanConfiguration().GetNotifiers() {

		// The storage.NotifierConfiguration does not contain the notifier name.
		// We need to retrieve the storage.Notifier from the data store to grab the notifier name.
		notifier, found, err := notifierDS.GetNotifier(ctx, notifierConfig.GetId())
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.Errorf("Notifier with ID %s no longer exists", notifierConfig.GetEmailConfig().GetNotifierId())
		}

		notifierV2, err := convertProtoNotifierConfigToV2(notifierConfig, notifier.GetName())
		if err != nil {
			return nil, err
		}
		notifiers = append(notifiers, notifierV2)
	}

	return &v2.ComplianceScanConfigurationStatus{
		Id:       reportData.GetScanConfiguration().GetId(),
		ScanName: reportData.GetScanConfiguration().GetScanConfigName(),
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: reportData.GetScanConfiguration().GetOneTimeScan(),
			Profiles: func() []string {

				ret := make([]string, 0, len(reportData.GetScanConfiguration().GetProfiles()))
				for _, profile := range reportData.GetScanConfiguration().GetProfiles() {
					ret = append(ret, profile.GetProfileName())
				}
				return ret
			}(),
			ScanSchedule: convertProtoScheduleToV2(reportData.GetScanConfiguration().GetSchedule()),
			Description:  reportData.GetScanConfiguration().GetDescription(),
			Notifiers:    notifiers,
		},
		ClusterStatus: func() []*v2.ClusterScanStatus {

			ret := make([]*v2.ClusterScanStatus, 0, len(reportData.GetClusterStatus()))
			for _, cluster := range reportData.GetClusterStatus() {
				ret = append(ret, &v2.ClusterScanStatus{
					ClusterId:   cluster.GetClusterId(),
					ClusterName: cluster.GetClusterName(),
					Errors:      cluster.GetErrors(),
					SuiteStatus: &v2.ClusterScanStatus_SuiteStatus{
						Phase:              cluster.GetSuiteStatus().GetPhase(),
						Result:             cluster.GetSuiteStatus().GetResult(),
						ErrorMessage:       cluster.GetSuiteStatus().GetErrorMessage(),
						LastTransitionTime: cluster.GetSuiteStatus().GetLastTransitionTime(),
					},
				})
			}
			return ret
		}(),
		CreatedTime:     reportData.GetScanConfiguration().GetCreatedTime(),
		LastUpdatedTime: reportData.GetScanConfiguration().GetLastUpdatedTime(),
		ModifiedBy: &v2.SlimUser{
			Id:   reportData.GetScanConfiguration().GetModifiedBy().GetId(),
			Name: reportData.GetScanConfiguration().GetModifiedBy().GetName(),
		},
		LastExecutedTime: reportData.GetLastExecutedTime(),
	}, nil
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

func convertStorageSnapshotsToV2Snapshots(ctx context.Context, snapshots []*storage.ComplianceOperatorReportSnapshotV2,
	configDS complianceDS.DataStore, bindingDS bindingsDS.DataStore, suiteDS suiteDS.DataStore, notifierDS notifierDS.DataStore, blobDS blobDS.Datastore) ([]*v2.ComplianceReportSnapshot, error) {
	if snapshots == nil {
		return nil, nil
	}
	retSnapshots := make([]*v2.ComplianceReportSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		converted, err := convertStorageSnapshotToV2Snapshot(ctx, snapshot, configDS, bindingDS, suiteDS, notifierDS, blobDS)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to convert storage compliance operator report snapshot %s to response", snapshot.GetName())
		}
		if converted == nil {
			continue
		}
		retSnapshots = append(retSnapshots, converted)
	}
	return retSnapshots, nil
}

func shallCheckDownload(reportStatus *storage.ComplianceOperatorReportStatus) bool {
	runState := reportStatus.GetRunState()
	return reportStatus.GetReportNotificationMethod() == storage.ComplianceOperatorReportStatus_DOWNLOAD &&
		(runState == storage.ComplianceOperatorReportStatus_GENERATED || runState == storage.ComplianceOperatorReportStatus_DELIVERED)
}

func convertStorageSnapshotToV2Snapshot(ctx context.Context, snapshot *storage.ComplianceOperatorReportSnapshotV2,
	configDS complianceDS.DataStore, bindingDS bindingsDS.DataStore, suiteDS suiteDS.DataStore, notifierDS notifierDS.DataStore, blobDS blobDS.Datastore) (*v2.ComplianceReportSnapshot, error) {
	if snapshot == nil {
		return nil, nil
	}
	config, found, err := configDS.GetScanConfiguration(ctx, snapshot.GetScanConfigurationId())
	if err != nil {
		return nil, errors.Wrap(err, "Unable to retrieve the ScanConfiguration")
	}
	if !found {
		return nil, errors.New("ScanConfiguration not found")
	}
	configStatus, err := convertStorageReportDataToV2ScanStatus(ctx, snapshot.GetReportData(), notifierDS)
	if err != nil {
		log.Warnf("unable to convert the report snapshot scan config to v2: %v", err)
	}
	if configStatus == nil {
		configStatus, err = convertStorageScanConfigToV2ScanStatus(ctx, config, configDS, bindingDS, suiteDS, notifierDS)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to convert ScanConfiguration to ScanStatus")
		}
	}
	isDownloadReady := false
	if shallCheckDownload(snapshot.GetReportStatus()) {
		blobName := common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshot.GetReportId())
		query := search.NewQueryBuilder().AddExactMatches(search.BlobName, blobName).ProtoQuery()
		blobResults, err := blobDS.Search(ctx, query)
		if err != nil {
			log.Errorf("unable to retrieve blob from the DataStore: %v", err)
		}
		blobs := search.ResultsToIDSet(blobResults)
		isDownloadReady = blobs.Contains(blobName)
	}
	retSnapshot := &v2.ComplianceReportSnapshot{
		ReportJobId:  snapshot.GetReportId(),
		ScanConfigId: snapshot.GetScanConfigurationId(),
		Name:         snapshot.GetName(),
		Description:  snapshot.GetDescription(),
		ReportStatus: &v2.ComplianceReportStatus{
			RunState:                 storageReportRunStateToV2[snapshot.GetReportStatus().GetRunState()],
			StartedAt:                snapshot.GetReportStatus().GetStartedAt(),
			CompletedAt:              snapshot.GetReportStatus().GetCompletedAt(),
			ErrorMsg:                 snapshot.GetReportStatus().GetErrorMsg(),
			ReportRequestType:        storageReportRequestTypeToV2[snapshot.GetReportStatus().GetReportRequestType()],
			ReportNotificationMethod: storageReportNotificationMethodToV2[snapshot.GetReportStatus().GetReportNotificationMethod()],
		},
		ReportData: configStatus,
		User: &v2.SlimUser{
			Id:   snapshot.GetUser().GetId(),
			Name: snapshot.GetUser().GetName(),
		},
		IsDownloadAvailable: isDownloadReady,
	}
	return retSnapshot, nil
}

func convertNotificationMethodToStorage(method v2.NotificationMethod) (storage.ComplianceOperatorReportStatus_NotificationMethod, error) {
	var ret storage.ComplianceOperatorReportStatus_NotificationMethod
	switch method {
	case v2.NotificationMethod_EMAIL:
		ret = storage.ComplianceOperatorReportStatus_EMAIL
	case v2.NotificationMethod_DOWNLOAD:
		ret = storage.ComplianceOperatorReportStatus_DOWNLOAD
	default:
		return ret, errors.New("unknown notification method")
	}
	return ret, nil
}
