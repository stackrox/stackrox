package v2

import (
	"context"

	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	v2IntervalTypeToStorage = map[apiV2.ReportSchedule_IntervalType]storage.Schedule_IntervalType{
		apiV2.ReportSchedule_UNSET:   storage.Schedule_UNSET,
		apiV2.ReportSchedule_WEEKLY:  storage.Schedule_WEEKLY,
		apiV2.ReportSchedule_MONTHLY: storage.Schedule_MONTHLY,
	}

	storageIntervalTypeToV2 = map[storage.Schedule_IntervalType]apiV2.ReportSchedule_IntervalType{
		storage.Schedule_UNSET:   apiV2.ReportSchedule_UNSET,
		storage.Schedule_DAILY:   apiV2.ReportSchedule_UNSET,
		storage.Schedule_WEEKLY:  apiV2.ReportSchedule_WEEKLY,
		storage.Schedule_MONTHLY: apiV2.ReportSchedule_MONTHLY,
	}

	// Use this context only to populate notifier and collection names before returning v2.ReportConfiguration response
	allAccessCtx = sac.WithAllAccess(context.Background())
)

/*
apiV2 type to storage type conversions
*/

// convertV2ReportConfigurationToProto converts v2.ReportConfiguration to storage.ReportConfiguration
func convertV2ReportConfigurationToProto(config *apiV2.ReportConfiguration, creator *storage.SlimUser,
	accessScopeRules []*storage.SimpleAccessScope_Rules) *storage.ReportConfiguration {
	if config == nil {
		return nil
	}

	ret := &storage.ReportConfiguration{
		Id:            config.GetId(),
		Name:          config.GetName(),
		Description:   config.GetDescription(),
		Type:          storage.ReportConfiguration_ReportType(config.GetType()),
		Schedule:      convertV2ScheduleToProto(config.GetSchedule()),
		ResourceScope: convertV2ResourceScopeToProto(config.GetResourceScope()),
		Creator:       creator,
	}

	if config.GetVulnReportFilters() != nil {
		ret.Filter = &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: convertV2VulnReportFiltersToProto(config.GetVulnReportFilters(), accessScopeRules),
		}
	}

	for _, notifier := range config.GetNotifiers() {
		ret.Notifiers = append(ret.Notifiers, convertV2NotifierConfigToProto(notifier))
	}

	return ret
}

func convertV2VulnReportFiltersToProto(filters *apiV2.VulnerabilityReportFilters,
	accessScopeRules []*storage.SimpleAccessScope_Rules) *storage.VulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &storage.VulnerabilityReportFilters{
		Fixability:       storage.VulnerabilityReportFilters_Fixability(filters.GetFixability()),
		AccessScopeRules: accessScopeRules,
	}

	for _, severity := range filters.GetSeverities() {
		ret.Severities = append(ret.Severities, storage.VulnerabilitySeverity(severity))
	}

	for _, imageType := range filters.GetImageTypes() {
		ret.ImageTypes = append(ret.ImageTypes, storage.VulnerabilityReportFilters_ImageType(imageType))
	}

	switch filters.CvesSince.(type) {
	case *apiV2.VulnerabilityReportFilters_AllVuln:
		ret.CvesSince = &storage.VulnerabilityReportFilters_AllVuln{
			AllVuln: filters.GetAllVuln(),
		}

	case *apiV2.VulnerabilityReportFilters_SinceLastSentScheduledReport:
		ret.CvesSince = &storage.VulnerabilityReportFilters_SinceLastSentScheduledReport{
			SinceLastSentScheduledReport: filters.GetSinceLastSentScheduledReport(),
		}

	case *apiV2.VulnerabilityReportFilters_SinceStartDate:
		ret.CvesSince = &storage.VulnerabilityReportFilters_SinceStartDate{
			SinceStartDate: filters.GetSinceStartDate(),
		}
	}

	return ret
}

func convertV2ResourceScopeToProto(scope *apiV2.ResourceScope) *storage.ResourceScope {
	if scope == nil {
		return nil
	}

	ret := &storage.ResourceScope{}
	if scope.GetCollectionScope() != nil {
		ret.ScopeReference = &storage.ResourceScope_CollectionId{CollectionId: scope.GetCollectionScope().GetCollectionId()}
	}
	return ret
}

func convertV2NotifierConfigToProto(notifier *apiV2.NotifierConfiguration) *storage.NotifierConfiguration {
	if notifier == nil {
		return nil
	}

	ret := &storage.NotifierConfiguration{
		Ref: &storage.NotifierConfiguration_Id{
			Id: notifier.GetEmailConfig().GetNotifierId(),
		},
	}
	if notifier.GetEmailConfig() != nil {
		ret.NotifierConfig = &storage.NotifierConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				MailingLists: notifier.GetEmailConfig().GetMailingLists(),
			},
		}
	}
	return ret
}

// convertV2ScheduleToProto converts v2.ReportSchedule to storage.Schedule. Does not validate v2.ReportSchedule
func convertV2ScheduleToProto(schedule *apiV2.ReportSchedule) *storage.Schedule {
	if schedule == nil {
		return nil
	}
	ret := &storage.Schedule{
		IntervalType: v2IntervalTypeToStorage[schedule.GetIntervalType()],
		Hour:         schedule.GetHour(),
		Minute:       schedule.GetMinute(),
	}
	switch schedule.Interval.(type) {
	case *apiV2.ReportSchedule_DaysOfWeek_:
		var days []int32
		// Convert to numbering starting from 0
		for _, d := range schedule.GetDaysOfWeek().GetDays() {
			days = append(days, d-1)
		}
		ret.Interval = &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{Days: days},
		}
	case *apiV2.ReportSchedule_DaysOfMonth_:
		ret.Interval = &storage.Schedule_DaysOfMonth_{
			DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}

/*
storage type to apiV2 type conversions
*/

// convertProtoReportConfigurationToV2 converts storage.ReportConfiguration to v2.ReportConfiguration
func convertProtoReportConfigurationToV2(config *storage.ReportConfiguration,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore) (*apiV2.ReportConfiguration, error) {
	if config == nil {
		return nil, nil
	}

	resourceScope, err := convertProtoResourceScopeToV2(config.GetResourceScope(), collectionDatastore)
	if err != nil {
		return nil, err
	}

	ret := &apiV2.ReportConfiguration{
		Id:            config.GetId(),
		Name:          config.GetName(),
		Description:   config.GetDescription(),
		Type:          apiV2.ReportConfiguration_ReportType(config.GetType()),
		Schedule:      ConvertProtoScheduleToV2(config.GetSchedule()),
		ResourceScope: resourceScope,
	}

	if config.GetVulnReportFilters() != nil {
		ret.Filter = &apiV2.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: ConvertProtoVulnReportFiltersToV2(config.GetVulnReportFilters()),
		}
	}

	for _, notifier := range config.GetNotifiers() {
		converted, err := ConvertProtoNotifierConfigToV2(notifier, notifierDatastore)
		if err != nil {
			return nil, err
		}
		ret.Notifiers = append(ret.Notifiers, converted)
	}

	return ret, nil
}

// ConvertProtoVulnReportFiltersToV2 converts storaage.VulnerabilityReportFilters to apiV2.VulnerabilityReportFilters
func ConvertProtoVulnReportFiltersToV2(filters *storage.VulnerabilityReportFilters) *apiV2.VulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &apiV2.VulnerabilityReportFilters{
		Fixability: apiV2.VulnerabilityReportFilters_Fixability(filters.GetFixability()),
	}

	for _, severity := range filters.GetSeverities() {
		ret.Severities = append(ret.Severities, apiV2.VulnerabilityReportFilters_VulnerabilitySeverity(severity))
	}

	for _, imageType := range filters.GetImageTypes() {
		ret.ImageTypes = append(ret.ImageTypes, apiV2.VulnerabilityReportFilters_ImageType(imageType))
	}

	switch filters.CvesSince.(type) {
	case *storage.VulnerabilityReportFilters_AllVuln:
		ret.CvesSince = &apiV2.VulnerabilityReportFilters_AllVuln{
			AllVuln: filters.GetAllVuln(),
		}

	case *storage.VulnerabilityReportFilters_SinceLastSentScheduledReport:
		ret.CvesSince = &apiV2.VulnerabilityReportFilters_SinceLastSentScheduledReport{
			SinceLastSentScheduledReport: filters.GetSinceLastSentScheduledReport(),
		}

	case *storage.VulnerabilityReportFilters_SinceStartDate:
		ret.CvesSince = &apiV2.VulnerabilityReportFilters_SinceStartDate{
			SinceStartDate: filters.GetSinceStartDate(),
		}
	}

	return ret
}

func convertProtoResourceScopeToV2(scope *storage.ResourceScope,
	collectionDatastore collectionDS.DataStore) (*apiV2.ResourceScope, error) {
	if scope == nil {
		return nil, nil
	}

	ret := &apiV2.ResourceScope{}
	if scope.GetScopeReference() != nil {
		var collectionName string
		collection, found, err := collectionDatastore.Get(allAccessCtx, scope.GetCollectionId())
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.Errorf("Collection with ID %s no longer exists", scope.GetCollectionId())
		}
		collectionName = collection.GetName()

		ret.ScopeReference = &apiV2.ResourceScope_CollectionScope{
			CollectionScope: &apiV2.CollectionReference{
				CollectionId:   scope.GetCollectionId(),
				CollectionName: collectionName,
			},
		}
	}
	return ret, nil
}

// ConvertProtoNotifierConfigToV2 converts storage.NotifierConfiguration to apiV2.NotifierConfiguration
func ConvertProtoNotifierConfigToV2(notifierConfig *storage.NotifierConfiguration,
	notifierDatastore notifierDS.DataStore) (*apiV2.NotifierConfiguration, error) {
	if notifierConfig == nil {
		return nil, nil
	}

	ret := &apiV2.NotifierConfiguration{}
	if notifierConfig.GetEmailConfig() != nil {
		emailConfig := &apiV2.EmailNotifierConfiguration{
			NotifierId: notifierConfig.GetEmailConfig().GetNotifierId(),
		}
		emailConfig.MailingLists = append(emailConfig.MailingLists, notifierConfig.GetEmailConfig().GetMailingLists()...)

		ret.NotifierConfig = &apiV2.NotifierConfiguration_EmailConfig{
			EmailConfig: emailConfig,
		}

		var notifierName string
		notifier, found, err := notifierDatastore.GetNotifier(allAccessCtx, notifierConfig.GetEmailConfig().GetNotifierId())
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.Errorf("Notifier with ID %s no longer exists", notifierConfig.GetEmailConfig().GetNotifierId())
		}
		notifierName = notifier.GetName()
		ret.NotifierName = notifierName
	}
	return ret, nil
}

// ConvertProtoScheduleToV2 converts storage.Schedule to v2.ReportSchedule. Does not validate storage.Schedule
func ConvertProtoScheduleToV2(schedule *storage.Schedule) *apiV2.ReportSchedule {
	if schedule == nil {
		return nil
	}
	ret := &apiV2.ReportSchedule{
		IntervalType: storageIntervalTypeToV2[schedule.GetIntervalType()],
		Hour:         schedule.GetHour(),
		Minute:       schedule.GetMinute(),
	}

	switch schedule.Interval.(type) {
	case *storage.Schedule_DaysOfWeek_:
		var days []int32
		// Convert to numbering starting from 1
		for _, d := range schedule.GetDaysOfWeek().GetDays() {
			days = append(days, d+1)
		}
		ret.Interval = &apiV2.ReportSchedule_DaysOfWeek_{
			DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{Days: days},
		}
	case *storage.Schedule_DaysOfMonth_:
		ret.Interval = &apiV2.ReportSchedule_DaysOfMonth_{
			DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}

func convertPrototoV2Reportstatus(status *storage.ReportStatus) *apiV2.ReportStatus {
	if status == nil {
		return nil
	}
	return &apiV2.ReportStatus{
		ReportRequestType:        apiV2.ReportStatus_ReportMethod(status.GetReportRequestType()),
		CompletedAt:              status.GetCompletedAt(),
		RunState:                 apiV2.ReportStatus_RunState(status.GetRunState()),
		ReportNotificationMethod: apiV2.NotificationMethod(status.GetReportNotificationMethod()),
		ErrorMsg:                 status.GetErrorMsg(),
	}

}

func convertProtoReportCollectiontoV2(collection *storage.CollectionSnapshot) *apiV2.CollectionSnapshot {
	if collection == nil {
		return nil
	}

	return &apiV2.CollectionSnapshot{
		Id:   collection.GetId(),
		Name: collection.GetName(),
	}
}

// ConvertProtoNotifierSnapshotToV2 converts notifiersnapshot proto to v2
func ConvertProtoNotifierSnapshotToV2(notifierSnapshot *storage.NotifierSnapshot) *apiV2.NotifierConfiguration {
	if notifierSnapshot == nil {
		return nil
	}
	if notifierSnapshot.GetEmailConfig() == nil {
		return &apiV2.NotifierConfiguration{}
	}

	return &apiV2.NotifierConfiguration{
		NotifierName: notifierSnapshot.GetNotifierName(),
		NotifierConfig: &apiV2.NotifierConfiguration_EmailConfig{
			EmailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:   notifierSnapshot.GetEmailConfig().GetNotifierId(),
				MailingLists: notifierSnapshot.GetEmailConfig().GetMailingLists(),
			},
		},
	}
}

// convertPrototoV2ReportSnapshot converts storage.ReportSnapshot to apiV2.ReportSnapshot
func convertProtoReportSnapshotstoV2(snapshots []*storage.ReportSnapshot) []*apiV2.ReportSnapshot {
	if snapshots == nil {
		return nil
	}
	res := make([]*apiV2.ReportSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotv2 := &apiV2.ReportSnapshot{
			ReportStatus:       convertPrototoV2Reportstatus(snapshot.GetReportStatus()),
			ReportConfigId:     snapshot.GetReportConfigurationId(),
			ReportJobId:        snapshot.GetReportId(),
			Name:               snapshot.GetName(),
			Description:        snapshot.GetDescription(),
			CollectionSnapshot: convertProtoReportCollectiontoV2(snapshot.GetCollection()),
			User: &apiV2.SlimUser{
				Id:   snapshot.GetRequester().GetId(),
				Name: snapshot.GetRequester().GetId(),
			},
			Schedule: ConvertProtoScheduleToV2(snapshot.GetSchedule()),
			Filter: &apiV2.ReportSnapshot_VulnReportFilters{
				VulnReportFilters: ConvertProtoVulnReportFiltersToV2(snapshot.GetVulnReportFilters()),
			},
		}
		for _, notifier := range snapshot.GetNotifiers() {
			converted := ConvertProtoNotifierSnapshotToV2(notifier)
			if converted != nil {
				snapshotv2.Notifiers = append(snapshotv2.Notifiers, converted)
			}
		}
		res = append(res, snapshotv2)

	}

	return res

}
