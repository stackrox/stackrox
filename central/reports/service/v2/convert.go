package v2

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reports/common"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
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

	storageRunStateToV2 = map[storage.ReportStatus_RunState]apiV2.ReportStatus_RunState{
		storage.ReportStatus_WAITING:   apiV2.ReportStatus_WAITING,
		storage.ReportStatus_PREPARING: apiV2.ReportStatus_PREPARING,
		storage.ReportStatus_GENERATED: apiV2.ReportStatus_GENERATED,
		storage.ReportStatus_DELIVERED: apiV2.ReportStatus_DELIVERED,
		storage.ReportStatus_FAILURE:   apiV2.ReportStatus_FAILURE,
	}

	// Use this context only to populate notifier, collection names and IsDownloadAvailable fields in converted responses
	allAccessCtx = sac.WithAllAccess(context.Background())
)

/*
apiV2 type to storage type conversions
*/

// convertV2ReportConfigurationToProto converts v2.ReportConfiguration to storage.ReportConfiguration
func (s *serviceImpl) convertV2ReportConfigurationToProto(config *apiV2.ReportConfiguration, creator *storage.SlimUser,
	accessScopeRules []*storage.SimpleAccessScope_Rules) *storage.ReportConfiguration {
	if config == nil {
		return nil
	}

	ret := &storage.ReportConfiguration{
		Id:            config.GetId(),
		Name:          config.GetName(),
		Description:   config.GetDescription(),
		Type:          storage.ReportConfiguration_ReportType(config.GetType()),
		Schedule:      s.convertV2ScheduleToProto(config.GetSchedule()),
		ResourceScope: s.convertV2ResourceScopeToProto(config.GetResourceScope()),
		Creator:       creator,
		Version:       2,
	}

	if config.GetVulnReportFilters() != nil {
		ret.Filter = &storage.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: s.convertV2VulnReportFiltersToProto(config.GetVulnReportFilters(), accessScopeRules),
		}
	}

	for _, notifier := range config.GetNotifiers() {
		ret.Notifiers = append(ret.Notifiers, s.convertV2NotifierConfigToProto(notifier))
	}

	return ret
}

func (s *serviceImpl) convertV2VulnReportFiltersToProto(filters *apiV2.VulnerabilityReportFilters,
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

func (s *serviceImpl) convertV2ResourceScopeToProto(scope *apiV2.ResourceScope) *storage.ResourceScope {
	if scope == nil {
		return nil
	}

	ret := &storage.ResourceScope{}
	if scope.GetCollectionScope() != nil {
		ret.ScopeReference = &storage.ResourceScope_CollectionId{CollectionId: scope.GetCollectionScope().GetCollectionId()}
	}
	return ret
}

func (s *serviceImpl) convertV2NotifierConfigToProto(notifier *apiV2.NotifierConfiguration) *storage.NotifierConfiguration {
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

// convertV2ScheduleToProto converts v2.ReportSchedule to storage.Schedule. Does not validate v2.ReportSchedule
func (s *serviceImpl) convertV2ScheduleToProto(schedule *apiV2.ReportSchedule) *storage.Schedule {
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
		ret.Interval = &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{Days: schedule.GetDaysOfWeek().GetDays()},
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
func (s *serviceImpl) convertProtoReportConfigurationToV2(config *storage.ReportConfiguration) (*apiV2.ReportConfiguration, error) {
	if config == nil {
		return nil, nil
	}

	resourceScope, err := s.convertProtoResourceScopeToV2(config.GetResourceScope())
	if err != nil {
		return nil, err
	}

	ret := &apiV2.ReportConfiguration{
		Id:            config.GetId(),
		Name:          config.GetName(),
		Description:   config.GetDescription(),
		Type:          apiV2.ReportConfiguration_ReportType(config.GetType()),
		Schedule:      s.convertProtoScheduleToV2(config.GetSchedule()),
		ResourceScope: resourceScope,
	}

	if config.GetVulnReportFilters() != nil {
		ret.Filter = &apiV2.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: s.convertProtoVulnReportFiltersToV2(config.GetVulnReportFilters()),
		}
	}

	for _, notifier := range config.GetNotifiers() {
		converted, err := s.convertProtoNotifierConfigToV2(notifier)
		if err != nil {
			return nil, err
		}
		ret.Notifiers = append(ret.Notifiers, converted)
	}

	return ret, nil
}

// convertProtoVulnReportFiltersToV2 converts storaage.VulnerabilityReportFilters to apiV2.VulnerabilityReportFilters
func (s *serviceImpl) convertProtoVulnReportFiltersToV2(filters *storage.VulnerabilityReportFilters) *apiV2.VulnerabilityReportFilters {
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

func (s *serviceImpl) convertProtoResourceScopeToV2(scope *storage.ResourceScope) (*apiV2.ResourceScope, error) {
	if scope == nil {
		return nil, nil
	}

	ret := &apiV2.ResourceScope{}
	if scope.GetScopeReference() != nil {
		var collectionName string
		collection, found, err := s.collectionDatastore.Get(allAccessCtx, scope.GetCollectionId())
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

// convertProtoNotifierConfigToV2 converts storage.NotifierConfiguration to apiV2.NotifierConfiguration
func (s *serviceImpl) convertProtoNotifierConfigToV2(notifierConfig *storage.NotifierConfiguration) (*apiV2.NotifierConfiguration, error) {
	if notifierConfig == nil {
		return nil, nil
	}

	if notifierConfig.GetEmailConfig() == nil {
		return nil, nil
	}

	notifier, found, err := s.notifierDatastore.GetNotifier(allAccessCtx, notifierConfig.GetId())
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Errorf("Notifier with ID %s no longer exists", notifierConfig.GetId())
	}

	return &apiV2.NotifierConfiguration{
		NotifierName: notifier.GetName(),
		NotifierConfig: &apiV2.NotifierConfiguration_EmailConfig{
			EmailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:    notifierConfig.GetId(),
				MailingLists:  notifierConfig.GetEmailConfig().GetMailingLists(),
				CustomSubject: notifierConfig.GetEmailConfig().GetCustomSubject(),
				CustomBody:    notifierConfig.GetEmailConfig().GetCustomBody(),
			},
		},
	}, nil
}

// convertProtoScheduleToV2 converts storage.Schedule to v2.ReportSchedule. Does not validate storage.Schedule
func (s *serviceImpl) convertProtoScheduleToV2(schedule *storage.Schedule) *apiV2.ReportSchedule {
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
		ret.Interval = &apiV2.ReportSchedule_DaysOfWeek_{
			DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{Days: schedule.GetDaysOfWeek().GetDays()},
		}
	case *storage.Schedule_DaysOfMonth_:
		ret.Interval = &apiV2.ReportSchedule_DaysOfMonth_{
			DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{Days: schedule.GetDaysOfMonth().GetDays()},
		}
	}

	return ret
}

func (s *serviceImpl) convertPrototoV2Reportstatus(status *storage.ReportStatus) *apiV2.ReportStatus {
	if status == nil {
		return nil
	}
	return &apiV2.ReportStatus{
		ReportRequestType:        apiV2.ReportStatus_ReportMethod(status.GetReportRequestType()),
		CompletedAt:              status.GetCompletedAt(),
		RunState:                 storageRunStateToV2[status.GetRunState()],
		ReportNotificationMethod: apiV2.NotificationMethod(status.GetReportNotificationMethod()),
		ErrorMsg:                 status.GetErrorMsg(),
	}
}

func (s *serviceImpl) convertProtoReportCollectiontoV2(collection *storage.CollectionSnapshot) *apiV2.CollectionSnapshot {
	if collection == nil {
		return nil
	}

	return &apiV2.CollectionSnapshot{
		Id:   collection.GetId(),
		Name: collection.GetName(),
	}
}

// convertProtoNotifierSnapshotToV2 converts notifiersnapshot proto to v2
func (s *serviceImpl) convertProtoNotifierSnapshotToV2(notifierSnapshot *storage.NotifierSnapshot) *apiV2.NotifierConfiguration {
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
				NotifierId:    notifierSnapshot.GetEmailConfig().GetNotifierId(),
				MailingLists:  notifierSnapshot.GetEmailConfig().GetMailingLists(),
				CustomSubject: notifierSnapshot.GetEmailConfig().GetCustomSubject(),
				CustomBody:    notifierSnapshot.GetEmailConfig().GetCustomBody(),
			},
		},
	}
}

// convertPrototoV2ReportSnapshot converts storage.ReportSnapshot to apiV2.ReportSnapshot
func (s *serviceImpl) convertProtoReportSnapshotstoV2(snapshots []*storage.ReportSnapshot) ([]*apiV2.ReportSnapshot, error) {
	if snapshots == nil {
		return nil, nil
	}
	blobNames, err := s.getExistingBlobNames(snapshots)
	if err != nil {
		return nil, err
	}
	v2snaps := make([]*apiV2.ReportSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		snapshotv2 := &apiV2.ReportSnapshot{
			ReportStatus:       s.convertPrototoV2Reportstatus(snapshot.GetReportStatus()),
			ReportConfigId:     snapshot.GetReportConfigurationId(),
			ReportJobId:        snapshot.GetReportId(),
			Name:               snapshot.GetName(),
			Description:        snapshot.GetDescription(),
			CollectionSnapshot: s.convertProtoReportCollectiontoV2(snapshot.GetCollection()),
			User: &apiV2.SlimUser{
				Id:   snapshot.GetRequester().GetId(),
				Name: snapshot.GetRequester().GetName(),
			},
			Schedule: s.convertProtoScheduleToV2(snapshot.GetSchedule()),
			Filter: &apiV2.ReportSnapshot_VulnReportFilters{
				VulnReportFilters: s.convertProtoVulnReportFiltersToV2(snapshot.GetVulnReportFilters()),
			},
			IsDownloadAvailable: blobNames.Contains(common.GetReportBlobPath(snapshot.GetReportConfigurationId(), snapshot.GetReportId())),
		}
		for _, notifier := range snapshot.GetNotifiers() {
			converted := s.convertProtoNotifierSnapshotToV2(notifier)
			if converted != nil {
				snapshotv2.Notifiers = append(snapshotv2.Notifiers, converted)
			}
		}
		v2snaps = append(v2snaps, snapshotv2)
	}

	return v2snaps, nil
}

func (s *serviceImpl) getExistingBlobNames(snapshots []*storage.ReportSnapshot) (set.StringSet, error) {
	blobNames := make([]string, 0)
	for _, snap := range snapshots {
		status := snap.GetReportStatus()
		if status.GetReportNotificationMethod() == storage.ReportStatus_DOWNLOAD {
			if status.GetRunState() == storage.ReportStatus_GENERATED ||
				status.GetRunState() == storage.ReportStatus_DELIVERED {
				blobNames = append(blobNames, common.GetReportBlobPath(snap.GetReportConfigurationId(), snap.GetReportId()))
			}
		}
	}

	query := search.NewQueryBuilder().AddExactMatches(search.BlobName, blobNames...).ProtoQuery()
	results, err := s.blobStore.Search(allAccessCtx, query)
	if err != nil {
		return nil, err
	}

	return search.ResultsToIDSet(results), nil
}
