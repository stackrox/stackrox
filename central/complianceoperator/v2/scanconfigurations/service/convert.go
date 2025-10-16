package service

import (
	"context"
	"strings"

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
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/protobuf/proto"
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
		storage.ComplianceOperatorReportStatus_WAITING:                     v2.ComplianceReportStatus_WAITING,
		storage.ComplianceOperatorReportStatus_PREPARING:                   v2.ComplianceReportStatus_PREPARING,
		storage.ComplianceOperatorReportStatus_GENERATED:                   v2.ComplianceReportStatus_GENERATED,
		storage.ComplianceOperatorReportStatus_DELIVERED:                   v2.ComplianceReportStatus_DELIVERED,
		storage.ComplianceOperatorReportStatus_FAILURE:                     v2.ComplianceReportStatus_FAILURE,
		storage.ComplianceOperatorReportStatus_PARTIAL_ERROR:               v2.ComplianceReportStatus_PARTIAL_ERROR,
		storage.ComplianceOperatorReportStatus_PARTIAL_SCAN_ERROR_DOWNLOAD: v2.ComplianceReportStatus_PARTIAL_SCAN_ERROR_DOWNLOAD,
		storage.ComplianceOperatorReportStatus_PARTIAL_SCAN_ERROR_EMAIL:    v2.ComplianceReportStatus_PARTIAL_SCAN_ERROR_EMAIL,
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

	bcscs := &v2.BaseComplianceScanConfigurationSettings{}
	bcscs.SetOneTimeScan(scanConfig.GetOneTimeScan())
	bcscs.SetScanSchedule(convertProtoScheduleToV2(scanConfig.GetSchedule()))
	bcscs.SetProfiles(profiles)
	bcscs.SetDescription(scanConfig.GetDescription())
	csc := &v2.ComplianceScanConfiguration{}
	csc.SetId(scanConfig.GetId())
	csc.SetScanName(scanConfig.GetScanConfigName())
	csc.SetClusters(clusters)
	csc.SetScanConfig(bcscs)
	return csc, nil
}

func convertV2NotifierConfigToProto(notifier *v2.NotifierConfiguration) *storage.NotifierConfiguration {
	if notifier == nil {
		return nil
	}

	ret := &storage.NotifierConfiguration{}
	ret.SetId(notifier.GetEmailConfig().GetNotifierId())

	if emailConfig := notifier.GetEmailConfig(); emailConfig != nil {
		enc := &storage.EmailNotifierConfiguration{}
		enc.SetMailingLists(emailConfig.GetMailingLists())
		enc.SetCustomSubject(emailConfig.GetCustomSubject())
		enc.SetCustomBody(emailConfig.GetCustomBody())
		ret.SetEmailConfig(proto.ValueOrDefault(enc))
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

	enc := &v2.EmailNotifierConfiguration{}
	enc.SetNotifierId(notifierConfig.GetId())
	enc.SetMailingLists(notifierConfig.GetEmailConfig().GetMailingLists())
	enc.SetCustomSubject(notifierConfig.GetEmailConfig().GetCustomSubject())
	enc.SetCustomBody(notifierConfig.GetEmailConfig().GetCustomBody())
	nc := &v2.NotifierConfiguration{}
	nc.SetNotifierName(notifierName)
	nc.SetEmailConfig(proto.ValueOrDefault(enc))
	return nc, nil
}

func convertV2ScanConfigToStorage(ctx context.Context, scanConfig *v2.ComplianceScanConfiguration) *storage.ComplianceOperatorScanConfigurationV2 {
	if scanConfig == nil {
		return nil
	}

	profiles := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName, 0, len(scanConfig.GetScanConfig().GetProfiles()))
	for _, profile := range scanConfig.GetScanConfig().GetProfiles() {
		cp := &storage.ComplianceOperatorScanConfigurationV2_ProfileName{}
		cp.SetProfileName(profile)
		profiles = append(profiles, cp)
	}

	clusters := make([]*storage.ComplianceOperatorScanConfigurationV2_Cluster, 0, len(scanConfig.GetClusters()))
	for _, cluster := range scanConfig.GetClusters() {
		cc := &storage.ComplianceOperatorScanConfigurationV2_Cluster{}
		cc.SetClusterId(cluster)
		clusters = append(clusters, cc)
	}

	notifiers := []*storage.NotifierConfiguration{}
	for _, notifier := range scanConfig.GetScanConfig().GetNotifiers() {
		notifierStorage := convertV2NotifierConfigToProto(notifier)
		if notifierStorage != nil {
			notifiers = append(notifiers, notifierStorage)
		}

	}

	coscv2 := &storage.ComplianceOperatorScanConfigurationV2{}
	coscv2.SetId(scanConfig.GetId())
	coscv2.SetScanConfigName(scanConfig.GetScanName())
	coscv2.SetAutoApplyRemediations(false)
	coscv2.SetAutoUpdateRemediations(false)
	coscv2.SetOneTimeScan(false)
	coscv2.SetStrictNodeScan(false)
	coscv2.SetSchedule(ConvertV2ScheduleToProto(scanConfig.GetScanConfig().GetScanSchedule()))
	coscv2.SetProfiles(profiles)
	coscv2.SetModifiedBy(authn.UserFromContext(ctx))
	coscv2.SetDescription(scanConfig.GetScanConfig().GetDescription())
	coscv2.SetClusters(clusters)
	coscv2.SetNotifiers(notifiers)
	return coscv2
}

// ConvertV2ScheduleToProto converts v2.Schedule to storage.Schedule. Does not validate v2.Schedule
func ConvertV2ScheduleToProto(schedule *v2.Schedule) *storage.Schedule {
	if schedule == nil {
		return nil
	}

	ret := &storage.Schedule{}
	ret.SetIntervalType(v2IntervalTypeToStorage[schedule.GetIntervalType()])
	ret.SetHour(schedule.GetHour())
	ret.SetMinute(schedule.GetMinute())
	switch schedule.WhichInterval() {
	case v2.Schedule_DaysOfWeek_case:
		sd := &storage.Schedule_DaysOfWeek{}
		sd.SetDays(schedule.GetDaysOfWeek().GetDays())
		ret.SetDaysOfWeek(proto.ValueOrDefault(sd))
	case v2.Schedule_DaysOfMonth_case:
		sd := &storage.Schedule_DaysOfMonth{}
		sd.SetDays(schedule.GetDaysOfMonth().GetDays())
		ret.SetDaysOfMonth(proto.ValueOrDefault(sd))
	}

	return ret
}

// convertProtoScheduleToV2 converts storage.Schedule to v2.Schedule. Does not validate storage.Schedule
func convertProtoScheduleToV2(schedule *storage.Schedule) *v2.Schedule {
	if schedule == nil {
		return nil
	}
	ret := &v2.Schedule{}
	ret.SetIntervalType(storageIntervalTypeToV2[schedule.GetIntervalType()])
	ret.SetHour(schedule.GetHour())
	ret.SetMinute(schedule.GetMinute())

	switch schedule.WhichInterval() {
	case storage.Schedule_DaysOfWeek_case:
		sd := &v2.Schedule_DaysOfWeek{}
		sd.SetDays(schedule.GetDaysOfWeek().GetDays())
		ret.SetDaysOfWeek(proto.ValueOrDefault(sd))
	case storage.Schedule_Weekly_case:
		sd := &v2.Schedule_DaysOfWeek{}
		sd.SetDays(schedule.GetDaysOfWeek().GetDays())
		ret.SetDaysOfWeek(proto.ValueOrDefault(sd))
	case storage.Schedule_DaysOfMonth_case:
		sd := &v2.Schedule_DaysOfMonth{}
		sd.SetDays(schedule.GetDaysOfMonth().GetDays())
		ret.SetDaysOfMonth(proto.ValueOrDefault(sd))
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

	return v2.ComplianceScanConfigurationStatus_builder{
		Id:       reportData.GetScanConfiguration().GetId(),
		ScanName: reportData.GetScanConfiguration().GetScanConfigName(),
		ScanConfig: v2.BaseComplianceScanConfigurationSettings_builder{
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
		}.Build(),
		ClusterStatus: func() []*v2.ClusterScanStatus {

			ret := make([]*v2.ClusterScanStatus, 0, len(reportData.GetClusterStatus()))
			for _, cluster := range reportData.GetClusterStatus() {
				ret = append(ret, v2.ClusterScanStatus_builder{
					ClusterId:   cluster.GetClusterId(),
					ClusterName: cluster.GetClusterName(),
					Errors:      cluster.GetErrors(),
					SuiteStatus: v2.ClusterScanStatus_SuiteStatus_builder{
						Phase:              cluster.GetSuiteStatus().GetPhase(),
						Result:             cluster.GetSuiteStatus().GetResult(),
						ErrorMessage:       cluster.GetSuiteStatus().GetErrorMessage(),
						LastTransitionTime: cluster.GetSuiteStatus().GetLastTransitionTime(),
					}.Build(),
				}.Build())
			}
			return ret
		}(),
		CreatedTime:     reportData.GetScanConfiguration().GetCreatedTime(),
		LastUpdatedTime: reportData.GetScanConfiguration().GetLastUpdatedTime(),
		ModifiedBy: v2.SlimUser_builder{
			Id:   reportData.GetScanConfiguration().GetModifiedBy().GetId(),
			Name: reportData.GetScanConfiguration().GetModifiedBy().GetName(),
		}.Build(),
		LastExecutedTime: reportData.GetLastExecutedTime(),
	}.Build(), nil
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
		suiteStatus := &v2.ClusterScanStatus_SuiteStatus{}
		suiteStatus.SetPhase(suite.GetStatus().GetPhase())
		suiteStatus.SetResult(suite.GetStatus().GetResult())
		suiteStatus.SetErrorMessage(suite.GetStatus().GetErrorMessage())
		conditions := suite.GetStatus().GetConditions()
		for _, c := range conditions {
			if suiteStatus.GetLastTransitionTime() == nil || protoutils.After(c.GetLastTransitionTime(), suiteStatus.GetLastTransitionTime()) {
				suiteStatus.SetLastTransitionTime(c.GetLastTransitionTime())
			}
		}

		// If the suite is complete, set the last scan time
		if suite.GetStatus().GetPhase() == suiteComplete && (lastScanTime == nil || protoutils.After(suiteStatus.GetLastTransitionTime(), lastScanTime)) {
			lastScanTime = suiteStatus.GetLastTransitionTime()
		}

		clusterToSuiteMap[suite.GetClusterId()] = suiteStatus
	}

	bcscs := &v2.BaseComplianceScanConfigurationSettings{}
	bcscs.SetOneTimeScan(scanConfig.GetOneTimeScan())
	bcscs.SetScanSchedule(convertProtoScheduleToV2(scanConfig.GetSchedule()))
	bcscs.SetProfiles(profiles)
	bcscs.SetDescription(scanConfig.GetDescription())
	bcscs.SetNotifiers(notifiers)
	slimUser := &v2.SlimUser{}
	slimUser.SetId(scanConfig.GetModifiedBy().GetId())
	slimUser.SetName(scanConfig.GetModifiedBy().GetName())
	cscs := &v2.ComplianceScanConfigurationStatus{}
	cscs.SetId(scanConfig.GetId())
	cscs.SetScanName(scanConfig.GetScanConfigName())
	cscs.SetClusterStatus(func() []*v2.ClusterScanStatus {
		clusterStatuses := make([]*v2.ClusterScanStatus, 0, len(scanClusters))
		for _, cluster := range scanClusters {
			var errors []string
			bindings, err := bindingsDS.GetScanSettingBindings(ctx, search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, scanConfig.GetScanConfigName()).
				AddExactMatches(search.ClusterID, cluster.GetClusterId()).ProtoQuery())
			if err != nil {
				continue
			}

			// We may not have received any bindings from sensor
			if len(bindings) != 0 {
				bindingError := getLatestBindingError(bindings[0].GetStatus())
				if bindingError != "" {
					errors = append(errors, bindingError)
				}
			}

			errors = append(errors, cluster.GetErrors()...)
			css := &v2.ClusterScanStatus{}
			css.SetClusterId(cluster.GetClusterId())
			css.SetClusterName(cluster.GetClusterName())
			css.SetErrors(errors)
			css.SetSuiteStatus(clusterToSuiteMap[cluster.GetClusterId()])
			clusterStatuses = append(clusterStatuses, css)
		}
		return clusterStatuses
	}())
	cscs.SetScanConfig(bcscs)
	cscs.SetModifiedBy(slimUser)
	cscs.SetCreatedTime(scanConfig.GetCreatedTime())
	cscs.SetLastUpdatedTime(scanConfig.GetLastUpdatedTime())
	cscs.SetLastExecutedTime(lastScanTime)
	return cscs, nil
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
		(runState == storage.ComplianceOperatorReportStatus_GENERATED ||
			runState == storage.ComplianceOperatorReportStatus_DELIVERED ||
			runState == storage.ComplianceOperatorReportStatus_PARTIAL_ERROR ||
			runState == storage.ComplianceOperatorReportStatus_PARTIAL_SCAN_ERROR_DOWNLOAD)
}

func failedClusterReasonsJoinFunc(reasons []string) string {
	return strings.Join(reasons, "; ")
}

func convertStorageFailedClustersToV2FailedClusters(failedClusters []*storage.ComplianceOperatorReportSnapshotV2_FailedCluster) []*v2.FailedCluster {
	ret := make([]*v2.FailedCluster, 0, len(failedClusters))
	for _, cluster := range failedClusters {
		fc := &v2.FailedCluster{}
		fc.SetClusterId(cluster.GetClusterId())
		fc.SetClusterName(cluster.GetClusterName())
		fc.SetOperatorVersion(cluster.GetOperatorVersion())
		fc.SetReason(failedClusterReasonsJoinFunc(cluster.GetReasons()))
		ret = append(ret, fc)
	}
	return ret
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
		// We need to add the Administration access to read from the BlobStore
		blobCtx := sac.WithGlobalAccessScopeChecker(ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Administration),
			),
		)
		blobResults, err := blobDS.Search(blobCtx, query)
		if err != nil {
			log.Errorf("unable to retrieve blob from the DataStore: %v", err)
		}
		blobs := search.ResultsToIDSet(blobResults)
		isDownloadReady = blobs.Contains(blobName)
	}
	crs := &v2.ComplianceReportStatus{}
	crs.SetRunState(storageReportRunStateToV2[snapshot.GetReportStatus().GetRunState()])
	crs.SetStartedAt(snapshot.GetReportStatus().GetStartedAt())
	crs.SetCompletedAt(snapshot.GetReportStatus().GetCompletedAt())
	crs.SetErrorMsg(snapshot.GetReportStatus().GetErrorMsg())
	crs.SetReportRequestType(storageReportRequestTypeToV2[snapshot.GetReportStatus().GetReportRequestType()])
	crs.SetReportNotificationMethod(storageReportNotificationMethodToV2[snapshot.GetReportStatus().GetReportNotificationMethod()])
	crs.SetFailedClusters(convertStorageFailedClustersToV2FailedClusters(snapshot.GetFailedClusters()))
	slimUser := &v2.SlimUser{}
	slimUser.SetId(snapshot.GetUser().GetId())
	slimUser.SetName(snapshot.GetUser().GetName())
	retSnapshot := &v2.ComplianceReportSnapshot{}
	retSnapshot.SetReportJobId(snapshot.GetReportId())
	retSnapshot.SetScanConfigId(snapshot.GetScanConfigurationId())
	retSnapshot.SetName(snapshot.GetName())
	retSnapshot.SetDescription(snapshot.GetDescription())
	retSnapshot.SetReportStatus(crs)
	retSnapshot.SetReportData(configStatus)
	retSnapshot.SetUser(slimUser)
	retSnapshot.SetIsDownloadAvailable(isDownloadReady)
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
