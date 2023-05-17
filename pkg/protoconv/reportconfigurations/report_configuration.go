package reportconfigurations

import (
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

var (
	v2IntervalTypeToStorage = map[v2.ReportSchedule_IntervalType]storage.Schedule_IntervalType{
		v2.ReportSchedule_UNSET:   storage.Schedule_UNSET,
		v2.ReportSchedule_WEEKLY:  storage.Schedule_WEEKLY,
		v2.ReportSchedule_MONTHLY: storage.Schedule_MONTHLY,
	}

	storageIntervalTypeToV2 = map[storage.Schedule_IntervalType]v2.ReportSchedule_IntervalType{
		storage.Schedule_UNSET:   v2.ReportSchedule_UNSET,
		storage.Schedule_DAILY:   v2.ReportSchedule_UNSET,
		storage.Schedule_WEEKLY:  v2.ReportSchedule_WEEKLY,
		storage.Schedule_MONTHLY: v2.ReportSchedule_MONTHLY,
	}
)

// ConvertV2ReportConfigurationToProto converts v2.ReportConfiguration to storage.ReportConfiguration
func ConvertV2ReportConfigurationToProto(config *v2.ReportConfiguration) *storage.ReportConfiguration {
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

func convertV2VulnReportFiltersToProto(filters *v2.VulnerabilityReportFilters) *storage.VulnerabilityReportFilters {
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
	case *v2.VulnerabilityReportFilters_AllVuln:
		ret.CvesSince = &storage.VulnerabilityReportFilters_AllVuln{
			AllVuln: filters.GetAllVuln(),
		}

	case *v2.VulnerabilityReportFilters_LastSuccessfulReport:
		ret.CvesSince = &storage.VulnerabilityReportFilters_LastSuccessfulReport{
			LastSuccessfulReport: filters.GetLastSuccessfulReport(),
		}

	case *v2.VulnerabilityReportFilters_StartDate:
		ret.CvesSince = &storage.VulnerabilityReportFilters_StartDate{
			StartDate: filters.GetStartDate(),
		}
	}

	return ret
}

func convertV2ResourceScopeToProto(scope *v2.ResourceScope) *storage.ResourceScope {
	if scope == nil {
		return nil
	}

	ret := &storage.ResourceScope{}
	if scope.GetScopeReference() != nil {
		ret.ScopeReference = &storage.ResourceScope_CollectionId{CollectionId: scope.GetCollectionId()}
	}
	return ret
}

func convertV2NotifierConfigToProto(notifier *v2.NotifierConfiguration) *storage.NotifierConfiguration {
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

// ConvertProtoReportConfigurationToV2 converts storage.ReportConfiguration to v2.ReportConfiguration
func ConvertProtoReportConfigurationToV2(config *storage.ReportConfiguration) *v2.ReportConfiguration {
	if config == nil {
		return nil
	}

	ret := &v2.ReportConfiguration{
		Id:            config.GetId(),
		Name:          config.GetName(),
		Description:   config.GetDescription(),
		Type:          v2.ReportConfiguration_ReportType(config.GetType()),
		Schedule:      convertProtoScheduleToV2(config.GetSchedule()),
		ResourceScope: convertProtoResourceScopeToV2(config.GetResourceScope()),
	}

	if config.GetVulnReportFilters() != nil {
		ret.Filter = &v2.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: convertProtoVulnReportFiltersToV2(config.GetVulnReportFilters()),
		}
	}

	for _, notifier := range config.GetNotifiers() {
		ret.Notifiers = append(ret.Notifiers, convertProtoNotifierConfigToV2(notifier))
	}

	return ret
}

func convertProtoVulnReportFiltersToV2(filters *storage.VulnerabilityReportFilters) *v2.VulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &v2.VulnerabilityReportFilters{
		Fixability: v2.VulnerabilityReportFilters_Fixability(filters.GetFixability()),
	}

	for _, severity := range filters.GetSeverities() {
		ret.Severities = append(ret.Severities, v2.VulnerabilityReportFilters_VulnerabilitySeverity(severity))
	}

	for _, imageType := range filters.GetImageTypes() {
		ret.ImageTypes = append(ret.ImageTypes, v2.VulnerabilityReportFilters_ImageType(imageType))
	}

	switch filters.CvesSince.(type) {
	case *storage.VulnerabilityReportFilters_AllVuln:
		ret.CvesSince = &v2.VulnerabilityReportFilters_AllVuln{
			AllVuln: filters.GetAllVuln(),
		}

	case *storage.VulnerabilityReportFilters_LastSuccessfulReport:
		ret.CvesSince = &v2.VulnerabilityReportFilters_LastSuccessfulReport{
			LastSuccessfulReport: filters.GetLastSuccessfulReport(),
		}

	case *storage.VulnerabilityReportFilters_StartDate:
		ret.CvesSince = &v2.VulnerabilityReportFilters_StartDate{
			StartDate: filters.GetStartDate(),
		}
	}

	return ret
}

func convertProtoResourceScopeToV2(scope *storage.ResourceScope) *v2.ResourceScope {
	if scope == nil {
		return nil
	}

	ret := &v2.ResourceScope{}
	if scope.GetScopeReference() != nil {
		ret.ScopeReference = &v2.ResourceScope_CollectionId{CollectionId: scope.GetCollectionId()}
	}
	return ret
}

func convertProtoNotifierConfigToV2(notifier *storage.NotifierConfiguration) *v2.NotifierConfiguration {
	if notifier == nil {
		return nil
	}

	ret := &v2.NotifierConfiguration{}
	if notifier.GetEmailConfig() != nil {
		emailConfig := &v2.EmailNotifierConfiguration{
			NotifierId: notifier.GetEmailConfig().GetNotifierId(),
		}
		emailConfig.MailingLists = append(emailConfig.MailingLists, notifier.GetEmailConfig().GetMailingLists()...)

		ret.NotifierConfig = &v2.NotifierConfiguration_EmailConfig{
			EmailConfig: emailConfig,
		}
	}
	return ret
}

// convertV2ScheduleToProto converts v2.ReportSchedule to storage.Schedule. Does not validate v2.ReportSchedule
func convertV2ScheduleToProto(schedule *v2.ReportSchedule) *storage.Schedule {
	if schedule == nil {
		return nil
	}
	ret := &storage.Schedule{
		IntervalType: v2IntervalTypeToStorage[schedule.GetIntervalType()],
		Hour:         schedule.GetHour(),
		Minute:       schedule.GetMinute(),
	}
	switch schedule.Interval.(type) {
	case *v2.ReportSchedule_DaysOfWeek_:
		var days []int32
		// Convert to numbering starting from 0
		for _, d := range schedule.GetDaysOfWeek().GetDays() {
			days = append(days, d-1)
		}
		ret.Interval = &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{Days: days},
		}
	case *v2.ReportSchedule_DaysOfMonth_:
		ret.Interval = &storage.Schedule_DaysOfMonth_{
			DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}

// convertProtoScheduleToV2 converts storage.Schedule to v2.ReportSchedule. Does not validate storage.Schedule
func convertProtoScheduleToV2(schedule *storage.Schedule) *v2.ReportSchedule {
	if schedule == nil {
		return nil
	}
	ret := &v2.ReportSchedule{
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
		ret.Interval = &v2.ReportSchedule_DaysOfWeek_{
			DaysOfWeek: &v2.ReportSchedule_DaysOfWeek{Days: days},
		}
	case *storage.Schedule_DaysOfMonth_:
		ret.Interval = &v2.ReportSchedule_DaysOfMonth_{
			DaysOfMonth: &v2.ReportSchedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}
