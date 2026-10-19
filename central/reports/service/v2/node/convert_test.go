package node

import (
	"testing"

	blobDSMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDSMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	v2 "github.com/stackrox/rox/central/reports/service/v2"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestConversions(t *testing.T) {
	suite.Run(t, new(ConversionTestSuite))
}

type ConversionTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	service  *serviceImpl
}

func (s *ConversionTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	reportConfigDataStore := reportConfigDSMocks.NewMockDataStore(s.mockCtrl)
	reportSnapshotDataStore := reportSnapshotDSMocks.NewMockDataStore(s.mockCtrl)
	collectionDataStore := collectionDSMocks.NewMockDataStore(s.mockCtrl)
	notifierDataStore := notifierDSMocks.NewMockDataStore(s.mockCtrl)
	blobStore := blobDSMocks.NewMockDatastore(s.mockCtrl)
	scheduler := schedulerMocks.NewMockScheduler(s.mockCtrl)
	validator := validation.New(reportConfigDataStore, reportSnapshotDataStore, collectionDataStore, notifierDataStore)

	s.service = &serviceImpl{
		reportConfigStore: reportConfigDataStore,
		snapshotDatastore: reportSnapshotDataStore,
		notifierDatastore: notifierDataStore,
		scheduler:         scheduler,
		blobStore:         blobStore,
		validator:         validator,
	}
}

func (s *ConversionTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ConversionTestSuite) TestScheduleConversions() {
	testCases := []struct {
		name       string
		v2Schedule *apiV2.ReportSchedule
	}{
		{
			name: "Daily schedule",
			v2Schedule: &apiV2.ReportSchedule{
				IntervalType: apiV2.ReportSchedule_DAILY,
				Hour:         14,
				Minute:       30,
			},
		},
		{
			name: "Weekly schedule",
			v2Schedule: &apiV2.ReportSchedule{
				IntervalType: apiV2.ReportSchedule_WEEKLY,
				Hour:         9,
				Minute:       0,
				Interval: &apiV2.ReportSchedule_DaysOfWeek_{
					DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{
						Days: []int32{1, 3, 5},
					},
				},
			},
		},
		{
			name: "Monthly schedule",
			v2Schedule: &apiV2.ReportSchedule{
				IntervalType: apiV2.ReportSchedule_MONTHLY,
				Hour:         10,
				Minute:       15,
				Interval: &apiV2.ReportSchedule_DaysOfMonth_{
					DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{
						Days: []int32{1, 15},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			protoSchedule := v2.ConvertV2ScheduleToProto(tc.v2Schedule)
			assert.NotNil(s.T(), protoSchedule)
			assert.Equal(s.T(), tc.v2Schedule.GetHour(), protoSchedule.GetHour())
			assert.Equal(s.T(), tc.v2Schedule.GetMinute(), protoSchedule.GetMinute())

			v2Schedule := v2.ConvertProtoScheduleToV2(protoSchedule)
			assert.NotNil(s.T(), v2Schedule)
			assert.Equal(s.T(), tc.v2Schedule.GetIntervalType(), v2Schedule.GetIntervalType())
			assert.Equal(s.T(), tc.v2Schedule.GetHour(), v2Schedule.GetHour())
			assert.Equal(s.T(), tc.v2Schedule.GetMinute(), v2Schedule.GetMinute())

			if tc.v2Schedule.GetDaysOfWeek() != nil {
				assert.Equal(s.T(), tc.v2Schedule.GetDaysOfWeek().GetDays(), v2Schedule.GetDaysOfWeek().GetDays())
			}
			if tc.v2Schedule.GetDaysOfMonth() != nil {
				assert.Equal(s.T(), tc.v2Schedule.GetDaysOfMonth().GetDays(), v2Schedule.GetDaysOfMonth().GetDays())
			}
		})
	}
}

func (s *ConversionTestSuite) TestNotifierConversions() {
	v2Notifier := &apiV2.NotifierConfiguration{
		NotifierConfig: &apiV2.NotifierConfiguration_EmailConfig{
			EmailConfig: &apiV2.EmailNotifierConfiguration{
				NotifierId:    "notifier-123",
				MailingLists:  []string{"test@example.com", "team@example.com"},
				CustomSubject: "Custom Subject",
				CustomBody:    "Custom Body",
			},
		},
	}

	protoNotifier := v2.ConvertV2NotifierConfigToProto(v2Notifier)
	assert.NotNil(s.T(), protoNotifier)
	assert.Equal(s.T(), "notifier-123", protoNotifier.GetId())
	assert.Equal(s.T(), "Custom Subject", protoNotifier.GetEmailConfig().GetCustomSubject())

	// Mock GetNotifier for the conversion back
	notifierDataStore := s.service.notifierDatastore.(*notifierDSMocks.MockDataStore)
	notifierDataStore.EXPECT().GetNotifier(gomock.Any(), "notifier-123").Return(&storage.Notifier{
		Id:   "notifier-123",
		Name: "Test Notifier",
	}, true, nil).Times(1)

	v2NotifierBack, err := v2.ConvertProtoNotifierConfigToV2(s.service.notifierDatastore, protoNotifier)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2NotifierBack)
	assert.Equal(s.T(), "Test Notifier", v2NotifierBack.GetNotifierName())
	assert.Equal(s.T(), v2Notifier.GetEmailConfig().GetNotifierId(), v2NotifierBack.GetEmailConfig().GetNotifierId())
	assert.Equal(s.T(), v2Notifier.GetEmailConfig().GetMailingLists(), v2NotifierBack.GetEmailConfig().GetMailingLists())
}

func (s *ConversionTestSuite) TestNotifierSnapshotConversion() {
	protoSnapshot := &storage.NotifierSnapshot{
		NotifierName: "Email Notifier",
		NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				NotifierId:    "notifier-456",
				MailingLists:  []string{"alerts@example.com"},
				CustomSubject: "Snapshot Subject",
				CustomBody:    "Snapshot Body",
			},
		},
	}

	v2Config := v2.ConvertProtoNotifierSnapshotToV2(protoSnapshot)
	assert.NotNil(s.T(), v2Config)
	assert.Equal(s.T(), "Email Notifier", v2Config.GetNotifierName())
	assert.Equal(s.T(), "notifier-456", v2Config.GetEmailConfig().GetNotifierId())
	assert.Equal(s.T(), "Snapshot Subject", v2Config.GetEmailConfig().GetCustomSubject())
	assert.Equal(s.T(), []string{"alerts@example.com"}, v2Config.GetEmailConfig().GetMailingLists())
}

func (s *ConversionTestSuite) TestEntityScopeConversions() {
	testCases := []struct {
		name       string
		entityType apiV2.ScopeEntity
		field      apiV2.ScopeField
		matchType  apiV2.MatchType
	}{
		{
			name:       "Cluster ID exact match",
			entityType: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:      apiV2.ScopeField_FIELD_ID,
			matchType:  apiV2.MatchType_EXACT,
		},
		{
			name:       "Cluster name regex match",
			entityType: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:      apiV2.ScopeField_FIELD_NAME,
			matchType:  apiV2.MatchType_REGEX,
		},
		{
			name:       "Cluster label exact match",
			entityType: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
			field:      apiV2.ScopeField_FIELD_LABEL,
			matchType:  apiV2.MatchType_EXACT,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			storageType := v2EntityTypeToStorage(tc.entityType)
			v2Type := storageEntityTypeToV2(storageType)
			assert.Equal(s.T(), tc.entityType, v2Type)

			storageField := v2EntityFieldToStorage(tc.field)
			v2Field := storageEntityFieldToV2(storageField)
			assert.Equal(s.T(), tc.field, v2Field)

			storageMatch := v2MatchTypeToStorage(tc.matchType)
			v2Match := storageMatchTypeToV2(storageMatch)
			assert.Equal(s.T(), tc.matchType, v2Match)
		})
	}
}

func (s *ConversionTestSuite) TestReportStatusConversion() {
	testCases := map[string]struct {
		requestType storage.ReportStatus_RunMethod
		want        apiV2.ReportStatus_ReportMethod
	}{
		"on-demand": {
			requestType: storage.ReportStatus_ON_DEMAND,
			want:        apiV2.ReportStatus_ON_DEMAND,
		},
		"scheduled": {
			requestType: storage.ReportStatus_SCHEDULED,
			want:        apiV2.ReportStatus_SCHEDULED,
		},
		"view-based": {
			requestType: storage.ReportStatus_VIEW_BASED,
			want:        apiV2.ReportStatus_ReportMethod(storage.ReportStatus_VIEW_BASED),
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			protoStatus := &storage.ReportStatus{
				RunState:                 storage.ReportStatus_GENERATED,
				ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
				ReportRequestType:        tc.requestType,
				ErrorMsg:                 "test error",
			}

			v2Status := convertPrototoV2Reportstatus(protoStatus)
			assert.NotNil(s.T(), v2Status)
			assert.Equal(s.T(), apiV2.ReportStatus_GENERATED, v2Status.GetRunState())
			assert.Equal(s.T(), apiV2.NotificationMethod_DOWNLOAD, v2Status.GetReportNotificationMethod())
			assert.Equal(s.T(), "test error", v2Status.GetErrorMsg())
			assert.Equal(s.T(), tc.want, v2Status.GetReportRequestType())
		})
	}
}

func (s *ConversionTestSuite) TestNodeFiltersConversion() {
	v2Filters := &apiV2.NodeVulnerabilityReportFilters{
		Query: "Cluster:prod",
		CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
			AllVuln: true,
		},
	}

	accessScopeRules := []*storage.SimpleAccessScope_Rules{
		{
			IncludedClusters: []string{"cluster-1"},
		},
	}

	protoFilters := convertV2NodeReportFiltersToProto(v2Filters, accessScopeRules)
	assert.NotNil(s.T(), protoFilters)
	assert.Equal(s.T(), "Cluster:prod", protoFilters.GetQuery())
	assert.True(s.T(), protoFilters.GetAllVuln())
	protoassert.SlicesEqual(s.T(), accessScopeRules, protoFilters.GetAccessScopeRules())

	v2FiltersBack := convertProtoNodeReportFiltersToV2(protoFilters)
	assert.NotNil(s.T(), v2FiltersBack)
	assert.Equal(s.T(), v2Filters.GetQuery(), v2FiltersBack.GetQuery())
	assert.True(s.T(), v2FiltersBack.GetAllVuln())
}

func (s *ConversionTestSuite) TestReportConfigurationRoundTrip() {
	creator := &storage.SlimUser{
		Id:   "user-123",
		Name: "Test User",
	}

	accessScopeRules := []*storage.SimpleAccessScope_Rules{
		{
			IncludedClusters: []string{"cluster-1", "cluster-2"},
		},
	}

	v2Config := &apiV2.ReportConfiguration{
		Id:          "config-123",
		Name:        "Node Report",
		Description: "Test node vulnerability report",
		Type:        apiV2.ReportConfiguration_NODE_VULNERABILITY,
		ResourceScope: &apiV2.ResourceScope{
			ScopeReference: &apiV2.ResourceScope_EntityScope{
				EntityScope: &apiV2.EntityScope{
					Rules: []*apiV2.EntityScopeRule{
						{
							Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
							Field:  apiV2.ScopeField_FIELD_ID,
							Values: []*apiV2.RuleValue{
								{
									Value:     "cluster-1",
									MatchType: apiV2.MatchType_EXACT,
								},
							},
						},
					},
				},
			},
		},
		Filter: &apiV2.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: &apiV2.NodeVulnerabilityReportFilters{
				Query: "Cluster:prod",
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
		Schedule: &apiV2.ReportSchedule{
			IntervalType: apiV2.ReportSchedule_WEEKLY,
			Hour:         9,
			Minute:       0,
			Interval: &apiV2.ReportSchedule_DaysOfWeek_{
				DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{
					Days: []int32{1, 3, 5},
				},
			},
		},
		Notifiers: []*apiV2.NotifierConfiguration{
			{
				NotifierConfig: &apiV2.NotifierConfiguration_EmailConfig{
					EmailConfig: &apiV2.EmailNotifierConfiguration{
						NotifierId:   "email-1",
						MailingLists: []string{"team@example.com"},
					},
				},
			},
		},
	}

	protoConfig, err := s.service.convertV2ReportConfigurationToProto(v2Config, creator, accessScopeRules)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), protoConfig)
	assert.Equal(s.T(), v2Config.GetId(), protoConfig.GetId())
	assert.Equal(s.T(), v2Config.GetName(), protoConfig.GetName())
	assert.Equal(s.T(), storage.ReportConfiguration_NODE_VULNERABILITY, protoConfig.GetType())
	protoassert.Equal(s.T(), creator, protoConfig.GetCreator())

	// Mock GetNotifier for the conversion back
	notifierDataStore := s.service.notifierDatastore.(*notifierDSMocks.MockDataStore)
	notifierDataStore.EXPECT().GetNotifier(gomock.Any(), "email-1").Return(&storage.Notifier{
		Id:   "email-1",
		Name: "Email Notifier",
	}, true, nil).Times(1)

	v2ConfigBack, err := s.service.convertProtoReportConfigurationToV2(protoConfig)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2ConfigBack)
	assert.Equal(s.T(), v2Config.GetId(), v2ConfigBack.GetId())
	assert.Equal(s.T(), v2Config.GetName(), v2ConfigBack.GetName())
	assert.Equal(s.T(), apiV2.ReportConfiguration_NODE_VULNERABILITY, v2ConfigBack.GetType())
	assert.Len(s.T(), v2ConfigBack.GetNotifiers(), 1)
	assert.Equal(s.T(), "Email Notifier", v2ConfigBack.GetNotifiers()[0].GetNotifierName())
}

func (s *ConversionTestSuite) TestNilConversions() {
	assert.Nil(s.T(), v2.ConvertV2ScheduleToProto(nil))
	assert.Nil(s.T(), v2.ConvertProtoScheduleToV2(nil))
	assert.Nil(s.T(), v2.ConvertV2NotifierConfigToProto(nil))

	result, err := v2.ConvertProtoNotifierConfigToV2(s.service.notifierDatastore, nil)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), result)

	assert.Nil(s.T(), v2.ConvertProtoNotifierSnapshotToV2(nil))
	assert.Nil(s.T(), convertPrototoV2Reportstatus(nil))
	assert.Nil(s.T(), convertV2NodeReportFiltersToProto(nil, nil))
	assert.Nil(s.T(), convertProtoNodeReportFiltersToV2(nil))
}

func (s *ConversionTestSuite) TestEntityTypeEdgeCases() {
	assert.Equal(s.T(), storage.EntityType_ENTITY_TYPE_UNSET, v2EntityTypeToStorage(apiV2.ScopeEntity_SCOPE_ENTITY_UNSET))
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_UNSET, storageEntityTypeToV2(storage.EntityType_ENTITY_TYPE_UNSET))

	assert.Equal(s.T(), storage.EntityField_FIELD_UNSET, v2EntityFieldToStorage(apiV2.ScopeField_FIELD_UNSET))
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_UNSET, storageEntityFieldToV2(storage.EntityField_FIELD_UNSET))

	assert.Equal(s.T(), storage.MatchType_EXACT, v2MatchTypeToStorage(apiV2.MatchType_EXACT))
	assert.Equal(s.T(), apiV2.MatchType_EXACT, storageMatchTypeToV2(storage.MatchType_EXACT))
}

func (s *ConversionTestSuite) TestConvertProtoReportSnapshotToV2() {
	configID := "test-config"
	reportID := "test-report"

	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: configID,
		ReportId:              reportID,
		Name:                  "Test Report",
		Description:           "Test description",
		AreaOfConcern:         "User Workloads",
		Type:                  storage.ReportSnapshot_NODE_VULNERABILITY,
		Requester: &storage.SlimUser{
			Id:   "user-1",
			Name: "Test User",
		},
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
		},
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_EntityScope{
				EntityScope: &storage.EntityScope{},
			},
		},
		Filter: &storage.ReportSnapshot_NodeVulnReportFilters{
			NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{
				Query: "test",
			},
		},
	}

	blobStore := s.service.blobStore.(*blobDSMocks.MockDatastore)
	blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	result, err := s.service.convertProtoReportSnapshotToV2(snapshot, set.NewFrozenStringSet())
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
	assert.Equal(s.T(), reportID, result.GetReportJobId())
	assert.Equal(s.T(), configID, result.GetReportConfigId())
	assert.Equal(s.T(), "Test Report", result.GetName())
	assert.Equal(s.T(), "Test description", result.GetDescription())
	assert.Equal(s.T(), "User Workloads", result.GetAreaOfConcern())
	assert.False(s.T(), result.GetIsDownloadAvailable())
}

func (s *ConversionTestSuite) TestReportSnapshotDownloadAvailable() {
	configID := "test-config"
	reportID := "test-report"

	testCases := map[string]struct {
		snapshot  *storage.ReportSnapshot
		blobNames set.FrozenStringSet
		available bool
	}{
		"config-based with matching blob": {
			snapshot: &storage.ReportSnapshot{
				ReportConfigurationId: configID,
				ReportId:              reportID,
				ReportStatus: &storage.ReportStatus{
					RunState:                 storage.ReportStatus_GENERATED,
					ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
				},
			},
			blobNames: set.NewFrozenStringSet(common.GetReportBlobPath(configID, reportID)),
			available: true,
		},
		"view-based with matching blob": {
			snapshot: &storage.ReportSnapshot{
				ReportId: reportID,
				ReportStatus: &storage.ReportStatus{
					RunState:                 storage.ReportStatus_GENERATED,
					ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
					ReportRequestType:        storage.ReportStatus_VIEW_BASED,
				},
			},
			blobNames: set.NewFrozenStringSet(common.GetReportBlobPath("view-based-report", reportID)),
			available: true,
		},
		"view-based without blob": {
			snapshot: &storage.ReportSnapshot{
				ReportId: reportID,
				ReportStatus: &storage.ReportStatus{
					RunState:                 storage.ReportStatus_GENERATED,
					ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
					ReportRequestType:        storage.ReportStatus_VIEW_BASED,
				},
			},
			blobNames: set.NewFrozenStringSet(),
			available: false,
		},
		"view-based does not match config-id blob path": {
			snapshot: &storage.ReportSnapshot{
				ReportConfigurationId: configID,
				ReportId:              reportID,
				ReportStatus: &storage.ReportStatus{
					RunState:                 storage.ReportStatus_GENERATED,
					ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
					ReportRequestType:        storage.ReportStatus_VIEW_BASED,
				},
			},
			blobNames: set.NewFrozenStringSet(common.GetReportBlobPath(configID, reportID)),
			available: false,
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			result, err := s.service.convertProtoReportSnapshotToV2(tc.snapshot, tc.blobNames)
			assert.NoError(s.T(), err)
			assert.Equal(s.T(), tc.available, result.GetIsDownloadAvailable())
		})
	}
}

func (s *ConversionTestSuite) TestRunStateConversions() {
	assert.Equal(s.T(), apiV2.ReportStatus_WAITING, storageRunStateToV2[storage.ReportStatus_WAITING])
	assert.Equal(s.T(), apiV2.ReportStatus_PREPARING, storageRunStateToV2[storage.ReportStatus_PREPARING])
	assert.Equal(s.T(), apiV2.ReportStatus_GENERATED, storageRunStateToV2[storage.ReportStatus_GENERATED])
	assert.Equal(s.T(), apiV2.ReportStatus_DELIVERED, storageRunStateToV2[storage.ReportStatus_DELIVERED])
	assert.Equal(s.T(), apiV2.ReportStatus_FAILURE, storageRunStateToV2[storage.ReportStatus_FAILURE])
}

func (s *ConversionTestSuite) TestResourceScopeConversions() {
	v2Scope := &apiV2.ResourceScope{
		ScopeReference: &apiV2.ResourceScope_EntityScope{
			EntityScope: &apiV2.EntityScope{
				Rules: []*apiV2.EntityScopeRule{
					{
						Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
						Field:  apiV2.ScopeField_FIELD_NAME,
						Values: []*apiV2.RuleValue{
							{
								Value:     "prod-cluster",
								MatchType: apiV2.MatchType_REGEX,
							},
						},
					},
				},
			},
		},
	}

	protoScope, err := s.service.convertV2ResourceScopeToProto(v2Scope)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), protoScope)

	v2ScopeBack := convertProtoResourceScopeToV2(protoScope)
	assert.NotNil(s.T(), v2ScopeBack)
	assert.NotNil(s.T(), v2ScopeBack.GetEntityScope())
	assert.Len(s.T(), v2ScopeBack.GetEntityScope().GetRules(), 1)
}

func (s *ConversionTestSuite) TestEmptyResourceScope() {
	result, err := s.service.convertV2ResourceScopeToProto(nil)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), result)

	v2Result := convertProtoResourceScopeToV2(nil)
	assert.Nil(s.T(), v2Result)
}

func (s *ConversionTestSuite) TestReportConfigurationWithoutNotifiers() {
	creator := &storage.SlimUser{
		Id:   "user-123",
		Name: "Test User",
	}

	v2Config := &apiV2.ReportConfiguration{
		Id:          "config-no-notifiers",
		Name:        "Node Report No Notifiers",
		Description: "Test without notifiers",
		Type:        apiV2.ReportConfiguration_NODE_VULNERABILITY,
		ResourceScope: &apiV2.ResourceScope{
			ScopeReference: &apiV2.ResourceScope_EntityScope{
				EntityScope: &apiV2.EntityScope{
					Rules: []*apiV2.EntityScopeRule{
						{
							Entity: apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER,
							Field:  apiV2.ScopeField_FIELD_ID,
							Values: []*apiV2.RuleValue{
								{
									Value:     "cluster-1",
									MatchType: apiV2.MatchType_EXACT,
								},
							},
						},
					},
				},
			},
		},
		Filter: &apiV2.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: &apiV2.NodeVulnerabilityReportFilters{
				Query: "Cluster:prod",
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
		Notifiers: nil,
	}

	protoConfig, err := s.service.convertV2ReportConfigurationToProto(v2Config, creator, nil)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), protoConfig)
	assert.Empty(s.T(), protoConfig.GetNotifiers())
}

func (s *ConversionTestSuite) TestProtoReportConfigurationToV2() {
	protoConfig := &storage.ReportConfiguration{
		Id:          "config-proto",
		Name:        "Proto Config",
		Description: "Test proto to v2",
		Type:        storage.ReportConfiguration_NODE_VULNERABILITY,
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_EntityScope{
				EntityScope: &storage.EntityScope{
					Rules: []*storage.EntityScopeRule{
						{
							Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
							Field:  storage.EntityField_FIELD_ID,
							Values: []*storage.RuleValue{
								{
									Value:     "cluster-1",
									MatchType: storage.MatchType_EXACT,
								},
							},
						},
					},
				},
			},
		},
		Filter: &storage.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{
				Query: "test query",
				CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
		Notifiers: []*storage.NotifierConfiguration{
			{
				Ref: &storage.NotifierConfiguration_Id{
					Id: "notifier-1",
				},
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "notifier-1",
						MailingLists: []string{"team@example.com"},
					},
				},
			},
		},
	}

	// Mock GetNotifier for the conversion
	notifierDataStore := s.service.notifierDatastore.(*notifierDSMocks.MockDataStore)
	notifierDataStore.EXPECT().GetNotifier(gomock.Any(), "notifier-1").Return(&storage.Notifier{
		Id:   "notifier-1",
		Name: "Notifier 1",
	}, true, nil).Times(1)

	v2Config, err := s.service.convertProtoReportConfigurationToV2(protoConfig)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2Config)
	assert.Equal(s.T(), "config-proto", v2Config.GetId())
	assert.Equal(s.T(), "Proto Config", v2Config.GetName())
	assert.Equal(s.T(), apiV2.ReportConfiguration_NODE_VULNERABILITY, v2Config.GetType())
	assert.Len(s.T(), v2Config.GetNotifiers(), 1)
	assert.Equal(s.T(), "Notifier 1", v2Config.GetNotifiers()[0].GetNotifierName())
}

func (s *ConversionTestSuite) TestProtoReportConfigurationToV2_SkipsNotifierWithoutEmail() {
	protoConfig := &storage.ReportConfiguration{
		Id:   "config-proto",
		Name: "Proto Config",
		Type: storage.ReportConfiguration_NODE_VULNERABILITY,
		Notifiers: []*storage.NotifierConfiguration{
			{
				Ref: &storage.NotifierConfiguration_Id{Id: "notifier-1"},
			},
		},
	}

	v2Config, err := s.service.convertProtoReportConfigurationToV2(protoConfig)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2Config)
	assert.Empty(s.T(), v2Config.GetNotifiers())
}

func (s *ConversionTestSuite) TestGetExistingBlobNames_NotDownloadMethod() {
	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: "config-1",
		ReportId:              "report-1",
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_EMAIL,
		},
	}

	result, err := s.service.getExistingBlobNames([]*storage.ReportSnapshot{snapshot})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, result.Cardinality())
}

func (s *ConversionTestSuite) TestGetExistingBlobNames_WrongRunState() {
	testCases := []struct {
		name     string
		runState storage.ReportStatus_RunState
	}{
		{"Waiting", storage.ReportStatus_WAITING},
		{"Preparing", storage.ReportStatus_PREPARING},
		{"Failure", storage.ReportStatus_FAILURE},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			snapshot := &storage.ReportSnapshot{
				ReportConfigurationId: "config-1",
				ReportId:              "report-1",
				ReportStatus: &storage.ReportStatus{
					RunState:                 tc.runState,
					ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
				},
			}

			result, err := s.service.getExistingBlobNames([]*storage.ReportSnapshot{snapshot})
			assert.NoError(s.T(), err)
			assert.Equal(s.T(), 0, result.Cardinality())
		})
	}
}

func (s *ConversionTestSuite) TestGetExistingBlobNames_BlobSearchError() {
	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: "config-1",
		ReportId:              "report-1",
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
		},
	}

	blobStore := s.service.blobStore.(*blobDSMocks.MockDatastore)
	blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, assert.AnError).Times(1)

	result, err := s.service.getExistingBlobNames([]*storage.ReportSnapshot{snapshot})
	assert.Error(s.T(), err)
	assert.Equal(s.T(), 0, result.Cardinality())
}

func (s *ConversionTestSuite) TestGetExistingBlobNames_NoBlobFound() {
	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: "config-1",
		ReportId:              "report-1",
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
		},
	}

	blobStore := s.service.blobStore.(*blobDSMocks.MockDatastore)
	blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).Times(1)

	result, err := s.service.getExistingBlobNames([]*storage.ReportSnapshot{snapshot})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, result.Cardinality())
}

func (s *ConversionTestSuite) TestGetExistingBlobNames_BlobFound() {
	testCases := []struct {
		name     string
		runState storage.ReportStatus_RunState
	}{
		{"Generated state", storage.ReportStatus_GENERATED},
		{"Delivered state", storage.ReportStatus_DELIVERED},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			snapshot := &storage.ReportSnapshot{
				ReportConfigurationId: "config-1",
				ReportId:              "report-1",
				ReportStatus: &storage.ReportStatus{
					RunState:                 tc.runState,
					ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
				},
			}

			blobStore := s.service.blobStore.(*blobDSMocks.MockDatastore)
			blobStore.EXPECT().Search(allAccessCtx, gomock.Any()).Return([]search.Result{
				{ID: common.GetReportBlobPath("config-1", "report-1")},
			}, nil).Times(1)

			result, err := s.service.getExistingBlobNames([]*storage.ReportSnapshot{snapshot})
			assert.NoError(s.T(), err)
			assert.True(s.T(), result.Contains(common.GetReportBlobPath("config-1", "report-1")))
		})
	}
}

func (s *ConversionTestSuite) TestGetExistingBlobNames_ViewBased() {
	snapshot := &storage.ReportSnapshot{
		ReportId: "report-1",
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
			ReportRequestType:        storage.ReportStatus_VIEW_BASED,
		},
	}

	blobStore := s.service.blobStore.(*blobDSMocks.MockDatastore)
	blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
		{ID: common.GetReportBlobPath("view-based-report", "report-1")},
	}, nil).Times(1)

	result, err := s.service.getExistingBlobNames([]*storage.ReportSnapshot{snapshot})
	assert.NoError(s.T(), err)
	assert.True(s.T(), result.Contains(common.GetReportBlobPath("view-based-report", "report-1")))
}

func (s *ConversionTestSuite) TestReportBlobParentDir() {
	testCases := map[string]struct {
		snapshot *storage.ReportSnapshot
		want     string
	}{
		"config-based on-demand": {
			snapshot: &storage.ReportSnapshot{
				ReportConfigurationId: "config-1",
				ReportStatus: &storage.ReportStatus{
					ReportRequestType: storage.ReportStatus_ON_DEMAND,
				},
			},
			want: "config-1",
		},
		"config-based scheduled": {
			snapshot: &storage.ReportSnapshot{
				ReportConfigurationId: "config-1",
				ReportStatus: &storage.ReportStatus{
					ReportRequestType: storage.ReportStatus_SCHEDULED,
				},
			},
			want: "config-1",
		},
		"view-based uses shared blob directory": {
			snapshot: &storage.ReportSnapshot{
				ReportConfigurationId: "config-1",
				ReportStatus: &storage.ReportStatus{
					ReportRequestType: storage.ReportStatus_VIEW_BASED,
				},
			},
			want: "view-based-report",
		},
	}

	for name, tc := range testCases {
		s.Run(name, func() {
			assert.Equal(s.T(), tc.want, reportBlobParentDir(tc.snapshot))
		})
	}
}
