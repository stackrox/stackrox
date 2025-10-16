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
	"google.golang.org/protobuf/proto"
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

	ret := &storage.ReportConfiguration{}
	ret.SetId(config.GetId())
	ret.SetName(config.GetName())
	ret.SetDescription(config.GetDescription())
	ret.SetType(storage.ReportConfiguration_ReportType(config.GetType()))
	ret.SetSchedule(s.convertV2ScheduleToProto(config.GetSchedule()))
	ret.SetResourceScope(s.convertV2ResourceScopeToProto(config.GetResourceScope()))
	ret.SetCreator(creator)
	ret.SetVersion(2)

	if config.GetVulnReportFilters() != nil {
		ret.SetVulnReportFilters(proto.ValueOrDefault(s.convertV2VulnReportFiltersToProto(config.GetVulnReportFilters(), accessScopeRules)))
	}

	for _, notifier := range config.GetNotifiers() {
		ret.SetNotifiers(append(ret.GetNotifiers(), s.convertV2NotifierConfigToProto(notifier)))
	}

	return ret
}

func (s *serviceImpl) convertV2VulnReportFiltersToProto(filters *apiV2.VulnerabilityReportFilters,
	accessScopeRules []*storage.SimpleAccessScope_Rules) *storage.VulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &storage.VulnerabilityReportFilters{}
	ret.SetFixability(storage.VulnerabilityReportFilters_Fixability(filters.GetFixability()))
	ret.SetAccessScopeRules(accessScopeRules)
	ret.SetIncludeNvdCvss(filters.GetIncludeNvdCvss())
	ret.SetIncludeEpssProbability(filters.GetIncludeEpssProbability())
	ret.SetIncludeAdvisory(filters.GetIncludeAdvisory())

	for _, severity := range filters.GetSeverities() {
		ret.SetSeverities(append(ret.GetSeverities(), storage.VulnerabilitySeverity(severity)))
	}

	for _, imageType := range filters.GetImageTypes() {
		ret.SetImageTypes(append(ret.GetImageTypes(), storage.VulnerabilityReportFilters_ImageType(imageType)))
	}

	switch filters.WhichCvesSince() {
	case apiV2.VulnerabilityReportFilters_AllVuln_case:
		ret.SetAllVuln(filters.GetAllVuln())

	case apiV2.VulnerabilityReportFilters_SinceLastSentScheduledReport_case:
		ret.SetSinceLastSentScheduledReport(filters.GetSinceLastSentScheduledReport())

	case apiV2.VulnerabilityReportFilters_SinceStartDate_case:
		ret.SetSinceStartDate(proto.ValueOrDefault(filters.GetSinceStartDate()))
	}

	return ret
}

func (s *serviceImpl) convertV2ResourceScopeToProto(scope *apiV2.ResourceScope) *storage.ResourceScope {
	if scope == nil {
		return nil
	}

	ret := &storage.ResourceScope{}
	if scope.GetCollectionScope() != nil {
		ret.SetCollectionId(scope.GetCollectionScope().GetCollectionId())
	}
	return ret
}

func (s *serviceImpl) convertV2NotifierConfigToProto(notifier *apiV2.NotifierConfiguration) *storage.NotifierConfiguration {
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

// convertV2ScheduleToProto converts v2.ReportSchedule to storage.Schedule. Does not validate v2.ReportSchedule
func (s *serviceImpl) convertV2ScheduleToProto(schedule *apiV2.ReportSchedule) *storage.Schedule {
	if schedule == nil {
		return nil
	}
	ret := &storage.Schedule{}
	ret.SetIntervalType(v2IntervalTypeToStorage[schedule.GetIntervalType()])
	ret.SetHour(schedule.GetHour())
	ret.SetMinute(schedule.GetMinute())
	switch schedule.WhichInterval() {
	case apiV2.ReportSchedule_DaysOfWeek_case:
		sd := &storage.Schedule_DaysOfWeek{}
		sd.SetDays(schedule.GetDaysOfWeek().GetDays())
		ret.SetDaysOfWeek(proto.ValueOrDefault(sd))
	case apiV2.ReportSchedule_DaysOfMonth_case:
		sd := &storage.Schedule_DaysOfMonth{}
		sd.SetDays(schedule.GetDaysOfMonth().GetDays())
		ret.SetDaysOfMonth(proto.ValueOrDefault(sd))
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

	ret := &apiV2.ReportConfiguration{}
	ret.SetId(config.GetId())
	ret.SetName(config.GetName())
	ret.SetDescription(config.GetDescription())
	ret.SetType(apiV2.ReportConfiguration_ReportType(config.GetType()))
	ret.SetSchedule(s.convertProtoScheduleToV2(config.GetSchedule()))
	ret.SetResourceScope(resourceScope)

	if config.GetVulnReportFilters() != nil {
		ret.SetVulnReportFilters(proto.ValueOrDefault(s.convertProtoVulnReportFiltersToV2(config.GetVulnReportFilters())))
	}

	for _, notifier := range config.GetNotifiers() {
		converted, err := s.convertProtoNotifierConfigToV2(notifier)
		if err != nil {
			return nil, err
		}
		ret.SetNotifiers(append(ret.GetNotifiers(), converted))
	}

	return ret, nil
}

// convertProtoVulnReportFiltersToV2 converts storaage.VulnerabilityReportFilters to apiV2.VulnerabilityReportFilters
func (s *serviceImpl) convertProtoVulnReportFiltersToV2(filters *storage.VulnerabilityReportFilters) *apiV2.VulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &apiV2.VulnerabilityReportFilters{}
	ret.SetFixability(apiV2.VulnerabilityReportFilters_Fixability(filters.GetFixability()))
	ret.SetIncludeNvdCvss(filters.GetIncludeNvdCvss())
	ret.SetIncludeEpssProbability(filters.GetIncludeEpssProbability())
	ret.SetIncludeAdvisory(filters.GetIncludeAdvisory())

	for _, severity := range filters.GetSeverities() {
		ret.SetSeverities(append(ret.GetSeverities(), apiV2.VulnerabilityReportFilters_VulnerabilitySeverity(severity)))
	}

	for _, imageType := range filters.GetImageTypes() {
		ret.SetImageTypes(append(ret.GetImageTypes(), apiV2.VulnerabilityReportFilters_ImageType(imageType)))
	}

	switch filters.WhichCvesSince() {
	case storage.VulnerabilityReportFilters_AllVuln_case:
		ret.SetAllVuln(filters.GetAllVuln())

	case storage.VulnerabilityReportFilters_SinceLastSentScheduledReport_case:
		ret.SetSinceLastSentScheduledReport(filters.GetSinceLastSentScheduledReport())

	case storage.VulnerabilityReportFilters_SinceStartDate_case:
		ret.SetSinceStartDate(proto.ValueOrDefault(filters.GetSinceStartDate()))
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

		cr := &apiV2.CollectionReference{}
		cr.SetCollectionId(scope.GetCollectionId())
		cr.SetCollectionName(collectionName)
		ret.SetCollectionScope(proto.ValueOrDefault(cr))
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

	enc := &apiV2.EmailNotifierConfiguration{}
	enc.SetNotifierId(notifierConfig.GetId())
	enc.SetMailingLists(notifierConfig.GetEmailConfig().GetMailingLists())
	enc.SetCustomSubject(notifierConfig.GetEmailConfig().GetCustomSubject())
	enc.SetCustomBody(notifierConfig.GetEmailConfig().GetCustomBody())
	nc := &apiV2.NotifierConfiguration{}
	nc.SetNotifierName(notifier.GetName())
	nc.SetEmailConfig(proto.ValueOrDefault(enc))
	return nc, nil
}

// convertProtoScheduleToV2 converts storage.Schedule to v2.ReportSchedule. Does not validate storage.Schedule
func (s *serviceImpl) convertProtoScheduleToV2(schedule *storage.Schedule) *apiV2.ReportSchedule {
	if schedule == nil {
		return nil
	}
	ret := &apiV2.ReportSchedule{}
	ret.SetIntervalType(storageIntervalTypeToV2[schedule.GetIntervalType()])
	ret.SetHour(schedule.GetHour())
	ret.SetMinute(schedule.GetMinute())

	switch schedule.WhichInterval() {
	case storage.Schedule_DaysOfWeek_case:
		rd := &apiV2.ReportSchedule_DaysOfWeek{}
		rd.SetDays(schedule.GetDaysOfWeek().GetDays())
		ret.SetDaysOfWeek(proto.ValueOrDefault(rd))
	case storage.Schedule_DaysOfMonth_case:
		rd := &apiV2.ReportSchedule_DaysOfMonth{}
		rd.SetDays(schedule.GetDaysOfMonth().GetDays())
		ret.SetDaysOfMonth(proto.ValueOrDefault(rd))
	}

	return ret
}

func (s *serviceImpl) convertPrototoV2Reportstatus(status *storage.ReportStatus) *apiV2.ReportStatus {
	if status == nil {
		return nil
	}
	rs := &apiV2.ReportStatus{}
	rs.SetReportRequestType(apiV2.ReportStatus_ReportMethod(status.GetReportRequestType()))
	rs.SetCompletedAt(status.GetCompletedAt())
	rs.SetRunState(storageRunStateToV2[status.GetRunState()])
	rs.SetReportNotificationMethod(apiV2.NotificationMethod(status.GetReportNotificationMethod()))
	rs.SetErrorMsg(status.GetErrorMsg())
	return rs
}

func (s *serviceImpl) convertProtoReportCollectiontoV2(collection *storage.CollectionSnapshot) *apiV2.CollectionSnapshot {
	if collection == nil {
		return nil
	}

	cs := &apiV2.CollectionSnapshot{}
	cs.SetId(collection.GetId())
	cs.SetName(collection.GetName())
	return cs
}

// convertProtoNotifierSnapshotToV2 converts notifiersnapshot proto to v2
func (s *serviceImpl) convertProtoNotifierSnapshotToV2(notifierSnapshot *storage.NotifierSnapshot) *apiV2.NotifierConfiguration {
	if notifierSnapshot == nil {
		return nil
	}
	if notifierSnapshot.GetEmailConfig() == nil {
		return &apiV2.NotifierConfiguration{}
	}

	enc := &apiV2.EmailNotifierConfiguration{}
	enc.SetNotifierId(notifierSnapshot.GetEmailConfig().GetNotifierId())
	enc.SetMailingLists(notifierSnapshot.GetEmailConfig().GetMailingLists())
	enc.SetCustomSubject(notifierSnapshot.GetEmailConfig().GetCustomSubject())
	enc.SetCustomBody(notifierSnapshot.GetEmailConfig().GetCustomBody())
	nc := &apiV2.NotifierConfiguration{}
	nc.SetNotifierName(notifierSnapshot.GetNotifierName())
	nc.SetEmailConfig(proto.ValueOrDefault(enc))
	return nc
}

// convertViewBasedPrototoV2ReportSnapshot converts storage.ReportSnapshot to apiV2.ReportSnapshot for view based reports
func (s *serviceImpl) convertViewBasedProtoReportSnapshotstoV2(snapshots []*storage.ReportSnapshot) ([]*apiV2.ReportSnapshot, error) {
	if snapshots == nil {
		return nil, nil
	}
	blobNames, err := s.getExistingBlobNames(snapshots)
	if err != nil {
		return nil, err
	}
	v2snaps := make([]*apiV2.ReportSnapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		viewBasedFilters := &apiV2.ViewBasedVulnerabilityReportFilters{}
		viewBasedFilters.SetQuery(snapshot.GetViewBasedVulnReportFilters().GetQuery())
		slimUser := &apiV2.SlimUser{}
		slimUser.SetId(snapshot.GetRequester().GetId())
		slimUser.SetName(snapshot.GetRequester().GetName())
		snapshotv2 := &apiV2.ReportSnapshot{}
		snapshotv2.SetReportStatus(s.convertPrototoV2Reportstatus(snapshot.GetReportStatus()))
		snapshotv2.SetReportConfigId(snapshot.GetReportConfigurationId())
		snapshotv2.SetReportJobId(snapshot.GetReportId())
		snapshotv2.SetName(snapshot.GetName())
		snapshotv2.SetDescription(snapshot.GetDescription())
		snapshotv2.SetAreaOfConcern(snapshot.GetAreaOfConcern())
		snapshotv2.SetUser(slimUser)
		snapshotv2.SetViewBasedVulnReportFilters(proto.ValueOrDefault(viewBasedFilters))
		snapshotv2.SetIsDownloadAvailable(blobNames.Contains(common.GetReportBlobPath("view-based-report", snapshot.GetReportId())))
		v2snaps = append(v2snaps, snapshotv2)
	}

	return v2snaps, nil
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
		slimUser := &apiV2.SlimUser{}
		slimUser.SetId(snapshot.GetRequester().GetId())
		slimUser.SetName(snapshot.GetRequester().GetName())
		snapshotv2 := &apiV2.ReportSnapshot{}
		snapshotv2.SetReportStatus(s.convertPrototoV2Reportstatus(snapshot.GetReportStatus()))
		snapshotv2.SetReportConfigId(snapshot.GetReportConfigurationId())
		snapshotv2.SetReportJobId(snapshot.GetReportId())
		snapshotv2.SetName(snapshot.GetName())
		snapshotv2.SetDescription(snapshot.GetDescription())
		snapshotv2.SetCollectionSnapshot(s.convertProtoReportCollectiontoV2(snapshot.GetCollection()))
		snapshotv2.SetUser(slimUser)
		snapshotv2.SetSchedule(s.convertProtoScheduleToV2(snapshot.GetSchedule()))
		snapshotv2.SetVulnReportFilters(proto.ValueOrDefault(s.convertProtoVulnReportFiltersToV2(snapshot.GetVulnReportFilters())))
		snapshotv2.SetIsDownloadAvailable(blobNames.Contains(common.GetReportBlobPath(snapshot.GetReportConfigurationId(), snapshot.GetReportId())))
		for _, notifier := range snapshot.GetNotifiers() {
			converted := s.convertProtoNotifierSnapshotToV2(notifier)
			if converted != nil {
				snapshotv2.SetNotifiers(append(snapshotv2.GetNotifiers(), converted))
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
				parentDir := snap.GetReportConfigurationId()
				if snap.GetViewBasedVulnReportFilters() != nil {
					parentDir = "view-based-report"
				}
				blobNames = append(blobNames, common.GetReportBlobPath(parentDir, snap.GetReportId()))
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
