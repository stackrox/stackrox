package v2

import (
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
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
)

/*
apiV2 type to storage type conversions
*/

// convertV2ReportConfigurationToProto converts v2.ReportConfiguration to storage.ReportConfiguration
func convertV2ReportConfigurationToProto(config *apiV2.ReportConfiguration) *storage.ReportConfiguration {
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
	}

	if config.GetVulnReportFilters() != nil {
		ret.Filter = &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: convertV2VulnReportFiltersToProto(config.GetVulnReportFilters()),
		}
	}

	for _, notifier := range config.GetNotifiers() {
		ret.Notifiers = append(ret.Notifiers, convertV2NotifierConfigToProto(notifier))
	}

	return ret
}

func convertV2VulnReportFiltersToProto(filters *apiV2.VulnerabilityReportFilters) *storage.VulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &storage.VulnerabilityReportFilters{
		Fixability: storage.VulnerabilityReportFilters_Fixability(filters.GetFixability()),
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

	ret := &storage.NotifierConfiguration{}
	if notifier.GetEmailConfig() != nil {
		emailConfig := &storage.EmailNotifierConfiguration{
			NotifierId: notifier.GetEmailConfig().GetNotifierId(),
		}
		emailConfig.MailingLists = append(emailConfig.MailingLists, notifier.GetEmailConfig().GetMailingLists()...)

		ret.NotifierConfig = &storage.NotifierConfiguration_EmailConfig{
			EmailConfig: emailConfig,
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
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore) *apiV2.ReportConfiguration {
	if config == nil {
		return nil
	}

	resourceScope := convertProtoResourceScopeToV2(config.GetResourceScope(), collectionDatastore)

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
		converted := ConvertProtoNotifierConfigToV2(notifier, notifierDatastore)
		if converted != nil {
			ret.Notifiers = append(ret.Notifiers, converted)
		}
	}

	return ret
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
	collectionDatastore collectionDS.DataStore) *apiV2.ResourceScope {
	if scope == nil {
		return nil
	}

	ret := &apiV2.ResourceScope{}
	if scope.GetScopeReference() != nil {
		collectionName := ""
		collection, found, err := collectionDatastore.Get(allAccessCtx, scope.GetCollectionId())
		if err != nil {
			log.Errorf("Error getting attached collection ID '%s': %s", scope.GetCollectionId(), err)
		} else if !found {
			log.Errorf("Collection with ID %s no longer exists", scope.GetCollectionId())
		}
		if collection != nil {
			collectionName = collection.GetName()
		}

		ret.ScopeReference = &apiV2.ResourceScope_CollectionScope{
			CollectionScope: &apiV2.CollectionReference{
				CollectionId:   scope.GetCollectionId(),
				CollectionName: collectionName,
			},
		}
	}
	return ret
}

// ConvertProtoNotifierConfigToV2 converts storage.NotifierConfiguration to apiV2.NotifierConfiguration
func ConvertProtoNotifierConfigToV2(notifierConfig *storage.NotifierConfiguration,
	notifierDatastore notifierDS.DataStore) *apiV2.NotifierConfiguration {
	if notifierConfig == nil {
		return nil
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

		notifier, found, err := notifierDatastore.GetNotifier(allAccessCtx, notifierConfig.GetEmailConfig().GetNotifierId())
		if err != nil {
			log.Errorf("Error getting attached notifier ID '%s': %s", notifierConfig.GetEmailConfig().GetNotifierId(), err)
		} else if !found {
			log.Errorf("Notifier with ID %s no longer exists", notifierConfig.GetEmailConfig().GetNotifierId())
		}
		if notifier != nil {
			ret.NotifierName = notifier.GetName()
		}
	}
	return ret
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
