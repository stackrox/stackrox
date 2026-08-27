package node

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/reports/common"
	v2 "github.com/stackrox/rox/central/reports/service/v2"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	v2IntervalTypeToStorage = map[apiV2.ReportSchedule_IntervalType]storage.Schedule_IntervalType{
		apiV2.ReportSchedule_UNSET:   storage.Schedule_UNSET,
		apiV2.ReportSchedule_WEEKLY:  storage.Schedule_WEEKLY,
		apiV2.ReportSchedule_MONTHLY: storage.Schedule_MONTHLY,
		apiV2.ReportSchedule_DAILY:   storage.Schedule_DAILY,
	}

	storageIntervalTypeToV2 = map[storage.Schedule_IntervalType]apiV2.ReportSchedule_IntervalType{
		storage.Schedule_UNSET:   apiV2.ReportSchedule_UNSET,
		storage.Schedule_DAILY:   apiV2.ReportSchedule_DAILY,
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
)

// convertV2ReportConfigurationToProto converts v2.ReportConfiguration to storage.ReportConfiguration for node reports
func (s *serviceImpl) convertV2ReportConfigurationToProto(config *apiV2.ReportConfiguration, creator *storage.SlimUser,
	accessScopeRules []*storage.SimpleAccessScope_Rules) (*storage.ReportConfiguration, error) {
	if config == nil {
		return nil, nil
	}

	resourceScope, err := s.convertV2ResourceScopeToProto(config.GetResourceScope())
	if err != nil {
		return nil, err
	}

	ret := &storage.ReportConfiguration{
		Id:            config.GetId(),
		Name:          config.GetName(),
		Description:   config.GetDescription(),
		Type:          storage.ReportConfiguration_NODE_VULNERABILITY,
		Schedule:      v2.ConvertV2ScheduleToProto(config.GetSchedule()),
		ResourceScope: resourceScope,
		Creator:       creator,
		Version:       2,
	}

	if config.GetNodeVulnReportFilters() != nil {
		ret.Filter = &storage.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: s.convertV2NodeReportFiltersToProto(config.GetNodeVulnReportFilters(), accessScopeRules),
		}
	}

	for _, notifier := range config.GetNotifiers() {
		ret.Notifiers = append(ret.Notifiers, s.convertV2NotifierConfigToProto(notifier))
	}

	return ret, nil
}

func (s *serviceImpl) convertV2NodeReportFiltersToProto(filters *apiV2.NodeVulnerabilityReportFilters,
	accessScopeRules []*storage.SimpleAccessScope_Rules) *storage.NodeVulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &storage.NodeVulnerabilityReportFilters{
		AccessScopeRules: accessScopeRules,
		Query:            filters.GetQuery(),
	}

	switch filters.GetCvesSince().(type) {
	case *apiV2.NodeVulnerabilityReportFilters_AllVuln:
		ret.CvesSince = &storage.NodeVulnerabilityReportFilters_AllVuln{
			AllVuln: filters.GetAllVuln(),
		}
	}

	return ret
}

func (s *serviceImpl) convertV2ResourceScopeToProto(scope *apiV2.ResourceScope) (*storage.ResourceScope, error) {
	if scope == nil {
		return nil, nil
	}

	ret := &storage.ResourceScope{}
	switch ref := scope.GetScopeReference().(type) {
	case *apiV2.ResourceScope_CollectionScope:
		if ref.CollectionScope == nil {
			return nil, errors.New("collection scope reference is nil")
		}
		ret.ScopeReference = &storage.ResourceScope_CollectionId{
			CollectionId: ref.CollectionScope.GetCollectionId(),
		}
	case *apiV2.ResourceScope_EntityScope:
		ret.ScopeReference = &storage.ResourceScope_EntityScope{
			EntityScope: convertV2EntityScopeToStorage(ref.EntityScope),
		}
	case nil:
		return nil, errors.New("resource scope reference is required")
	default:
		return nil, errors.Errorf("unsupported resource scope type: %T", ref)
	}
	return ret, nil
}

func convertV2EntityScopeToStorage(es *apiV2.EntityScope) *storage.EntityScope {
	if es == nil {
		return nil
	}
	rules := make([]*storage.EntityScopeRule, 0, len(es.GetRules()))
	for _, rule := range es.GetRules() {
		sr := &storage.EntityScopeRule{
			Entity: v2EntityTypeToStorage(rule.GetEntity()),
			Field:  v2EntityFieldToStorage(rule.GetField()),
		}
		for _, rv := range rule.GetValues() {
			sr.Values = append(sr.Values, &storage.RuleValue{
				Value:     rv.GetValue(),
				MatchType: v2MatchTypeToStorage(rv.GetMatchType()),
			})
		}
		rules = append(rules, sr)
	}
	return &storage.EntityScope{Rules: rules}
}

func v2EntityTypeToStorage(e apiV2.ScopeEntity) storage.EntityType {
	switch e {
	case apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER:
		return storage.EntityType_ENTITY_TYPE_CLUSTER
	default:
		return storage.EntityType_ENTITY_TYPE_UNSET
	}
}

func v2EntityFieldToStorage(f apiV2.ScopeField) storage.EntityField {
	switch f {
	case apiV2.ScopeField_FIELD_ID:
		return storage.EntityField_FIELD_ID
	case apiV2.ScopeField_FIELD_NAME:
		return storage.EntityField_FIELD_NAME
	case apiV2.ScopeField_FIELD_LABEL:
		return storage.EntityField_FIELD_LABEL
	default:
		return storage.EntityField_FIELD_UNSET
	}
}

func v2MatchTypeToStorage(m apiV2.MatchType) storage.MatchType {
	switch m {
	case apiV2.MatchType_REGEX:
		return storage.MatchType_REGEX
	default:
		return storage.MatchType_EXACT
	}
}

func (s *serviceImpl) convertV2NotifierConfigToProto(config *apiV2.NotifierConfiguration) *storage.NotifierConfiguration {
	if config == nil {
		return nil
	}

	ret := &storage.NotifierConfiguration{}

	if emailConfig := config.GetEmailConfig(); emailConfig != nil {
		ret.NotifierConfig = &storage.NotifierConfiguration_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				NotifierId:    emailConfig.GetNotifierId(),
				MailingLists:  emailConfig.GetMailingLists(),
				CustomSubject: emailConfig.GetCustomSubject(),
				CustomBody:    emailConfig.GetCustomBody(),
			},
		}
	}

	return ret
}

// Proto to V2 conversions

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
		Type:          apiV2.ReportConfiguration_NODE_VULNERABILITY,
		Schedule:      v2.ConvertProtoScheduleToV2(config.GetSchedule()),
		ResourceScope: resourceScope,
	}

	if config.GetNodeVulnReportFilters() != nil {
		ret.Filter = &apiV2.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: s.convertProtoNodeReportFiltersToV2(config.GetNodeVulnReportFilters()),
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

func (s *serviceImpl) convertProtoNodeReportFiltersToV2(filters *storage.NodeVulnerabilityReportFilters) *apiV2.NodeVulnerabilityReportFilters {
	if filters == nil {
		return nil
	}

	ret := &apiV2.NodeVulnerabilityReportFilters{
		Query: filters.GetQuery(),
	}

	switch filters.GetCvesSince().(type) {
	case *storage.NodeVulnerabilityReportFilters_AllVuln:
		ret.CvesSince = &apiV2.NodeVulnerabilityReportFilters_AllVuln{
			AllVuln: filters.GetAllVuln(),
		}
	}

	return ret
}

func (s *serviceImpl) convertProtoResourceScopeToV2(scope *storage.ResourceScope) (*apiV2.ResourceScope, error) {
	if scope == nil {
		return nil, nil
	}

	ret := &apiV2.ResourceScope{}
	if ref, ok := scope.GetScopeReference().(*storage.ResourceScope_EntityScope); ok {
		ret.ScopeReference = &apiV2.ResourceScope_EntityScope{EntityScope: convertStorageEntityScopeToV2(ref.EntityScope)}
	}
	return ret, nil
}

func convertStorageEntityScopeToV2(es *storage.EntityScope) *apiV2.EntityScope {
	if es == nil {
		return nil
	}
	rules := make([]*apiV2.EntityScopeRule, 0, len(es.GetRules()))
	for _, rule := range es.GetRules() {
		sr := &apiV2.EntityScopeRule{
			Entity: storageEntityTypeToV2(rule.GetEntity()),
			Field:  storageEntityFieldToV2(rule.GetField()),
		}
		for _, rv := range rule.GetValues() {
			sr.Values = append(sr.Values, &apiV2.RuleValue{
				Value:     rv.GetValue(),
				MatchType: storageMatchTypeToV2(rv.GetMatchType()),
			})
		}
		rules = append(rules, sr)
	}
	return &apiV2.EntityScope{Rules: rules}
}

func storageEntityTypeToV2(e storage.EntityType) apiV2.ScopeEntity {
	switch e {
	case storage.EntityType_ENTITY_TYPE_CLUSTER:
		return apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER
	default:
		return apiV2.ScopeEntity_SCOPE_ENTITY_UNSET
	}
}

func storageEntityFieldToV2(f storage.EntityField) apiV2.ScopeField {
	switch f {
	case storage.EntityField_FIELD_ID:
		return apiV2.ScopeField_FIELD_ID
	case storage.EntityField_FIELD_NAME:
		return apiV2.ScopeField_FIELD_NAME
	case storage.EntityField_FIELD_LABEL:
		return apiV2.ScopeField_FIELD_LABEL
	default:
		return apiV2.ScopeField_FIELD_UNSET
	}
}

func storageMatchTypeToV2(m storage.MatchType) apiV2.MatchType {
	switch m {
	case storage.MatchType_REGEX:
		return apiV2.MatchType_REGEX
	default:
		return apiV2.MatchType_EXACT
	}
}

func (s *serviceImpl) convertProtoNotifierConfigToV2(config *storage.NotifierConfiguration) (*apiV2.NotifierConfiguration, error) {
	if config == nil {
		return nil, nil
	}

	ret := &apiV2.NotifierConfiguration{}

	if emailConfig := config.GetEmailConfig(); emailConfig != nil {
		ret.NotifierConfig = &apiV2.NotifierConfiguration_EmailConfig{
			EmailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:    emailConfig.GetNotifierId(),
				MailingLists:  emailConfig.GetMailingLists(),
				CustomSubject: emailConfig.GetCustomSubject(),
				CustomBody:    emailConfig.GetCustomBody(),
			},
		}
	}

	return ret, nil
}

func isDownloadAvailable(blobNames set.FrozenStringSet, configID, reportID string) bool {
	return blobNames.Contains(common.GetReportBlobPath(configID, reportID))
}

func (s *serviceImpl) convertProtoReportSnapshotToV2(snapshot *storage.ReportSnapshot, blobNames set.FrozenStringSet) (*apiV2.ReportSnapshot, error) {
	if snapshot == nil {
		return nil, nil
	}

	resourceScope, err := s.convertProtoResourceScopeToV2(snapshot.GetResourceScope())
	if err != nil {
		return nil, err
	}

	ret := &apiV2.ReportSnapshot{
		ReportStatus:   convertPrototoV2Reportstatus(snapshot.GetReportStatus()),
		ReportConfigId: snapshot.GetReportConfigurationId(),
		ReportJobId:    snapshot.GetReportId(),
		Name:           snapshot.GetName(),
		Description:    snapshot.GetDescription(),
		Type:           apiV2.ReportSnapshot_NODE_VULNERABILITY,
		User: &apiV2.SlimUser{
			Id:   snapshot.GetRequester().GetId(),
			Name: snapshot.GetRequester().GetName(),
		},
		Schedule:            v2.ConvertProtoScheduleToV2(snapshot.GetSchedule()),
		ResourceScope:       resourceScope,
		IsDownloadAvailable: isDownloadAvailable(blobNames, snapshot.GetReportConfigurationId(), snapshot.GetReportId()),
	}

	if snapshot.GetNodeVulnReportFilters() != nil {
		ret.Filter = &apiV2.ReportSnapshot_NodeVulnReportFilters{
			NodeVulnReportFilters: s.convertProtoNodeReportFiltersToV2(snapshot.GetNodeVulnReportFilters()),
		}
	}

	for _, notifier := range snapshot.GetNotifiers() {
		converted := s.convertProtoNotifierSnapshotToV2(notifier)
		if converted != nil {
			ret.Notifiers = append(ret.Notifiers, converted)
		}
	}

	return ret, nil
}

func (s *serviceImpl) convertProtoNotifierSnapshotToV2(notifier *storage.NotifierSnapshot) *apiV2.NotifierConfiguration {
	if notifier == nil {
		return nil
	}

	ret := &apiV2.NotifierConfiguration{
		NotifierName: notifier.GetNotifierName(),
	}

	if emailConfig := notifier.GetEmailConfig(); emailConfig != nil {
		ret.NotifierConfig = &apiV2.NotifierConfiguration_EmailConfig{
			EmailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:    emailConfig.GetNotifierId(),
				MailingLists:  emailConfig.GetMailingLists(),
				CustomSubject: emailConfig.GetCustomSubject(),
				CustomBody:    emailConfig.GetCustomBody(),
			},
		}
	}

	return ret
}

func convertPrototoV2Reportstatus(status *storage.ReportStatus) *apiV2.ReportStatus {
	if status == nil {
		return nil
	}

	ret := &apiV2.ReportStatus{
		ReportNotificationMethod: apiV2.NotificationMethod(status.GetReportNotificationMethod()),
		RunState:                 storageRunStateToV2[status.GetRunState()],
		ErrorMsg:                 status.GetErrorMsg(),
		CompletedAt:              status.GetCompletedAt(),
	}

	return ret
}

func (s *serviceImpl) getExistingBlobNames(ctx context.Context, snapshots []*storage.ReportSnapshot) (set.FrozenStringSet, error) {
	if len(snapshots) == 0 {
		return set.NewFrozenStringSet(), nil
	}

	blobNames := make([]string, 0, len(snapshots))
	for _, snapshot := range snapshots {
		status := snapshot.GetReportStatus()
		if status.GetReportNotificationMethod() == storage.ReportStatus_DOWNLOAD {
			if status.GetRunState() == storage.ReportStatus_GENERATED ||
				status.GetRunState() == storage.ReportStatus_DELIVERED {
				blobNames = append(blobNames, common.GetReportBlobPath(snapshot.GetReportConfigurationId(), snapshot.GetReportId()))
			}
		}
	}

	if len(blobNames) == 0 {
		return set.NewFrozenStringSet(), nil
	}

	query := search.NewQueryBuilder().AddExactMatches(search.BlobName, blobNames...).ProtoQuery()
	results, err := s.blobStore.Search(ctx, query)
	if err != nil {
		return set.NewFrozenStringSet(), errors.Wrap(err, "failed to search for report blobs")
	}

	return search.ResultsToIDSet(results).Freeze(), nil
}
