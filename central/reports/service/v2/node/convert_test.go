package node

import (
	"context"
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
	ctx      context.Context
}

func (s *ConversionTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	reportConfigDataStore := reportConfigDSMocks.NewMockDataStore(s.mockCtrl)
	reportSnapshotDataStore := reportSnapshotDSMocks.NewMockDataStore(s.mockCtrl)
	collectionDataStore := collectionDSMocks.NewMockDataStore(s.mockCtrl)
	notifierDataStore := notifierDSMocks.NewMockDataStore(s.mockCtrl)
	blobStore := blobDSMocks.NewMockDatastore(s.mockCtrl)
	scheduler := schedulerMocks.NewMockScheduler(s.mockCtrl)
	validator := validation.New(reportConfigDataStore, reportSnapshotDataStore, collectionDataStore, notifierDataStore)

	s.service = &serviceImpl{
		reportConfigStore:   reportConfigDataStore,
		snapshotDatastore:   reportSnapshotDataStore,
		collectionDatastore: collectionDataStore,
		notifierDatastore:   notifierDataStore,
		scheduler:           scheduler,
		blobStore:           blobStore,
		validator:           validator,
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

	protoNotifier := s.service.convertV2NotifierConfigToProto(v2Notifier)
	assert.NotNil(s.T(), protoNotifier)
	assert.Equal(s.T(), "notifier-123", protoNotifier.GetEmailConfig().GetNotifierId())
	assert.Equal(s.T(), "Custom Subject", protoNotifier.GetEmailConfig().GetCustomSubject())

	v2NotifierBack, err := s.service.convertProtoNotifierConfigToV2(protoNotifier)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2NotifierBack)
	assert.Equal(s.T(), v2Notifier.GetEmailConfig().GetNotifierId(), v2NotifierBack.GetEmailConfig().GetNotifierId())
	assert.Equal(s.T(), v2Notifier.GetEmailConfig().GetMailingLists(), v2NotifierBack.GetEmailConfig().GetMailingLists())
}

func (s *ConversionTestSuite) TestNotifierSnapshotConversion() {
	protoSnapshot := &storage.NotifierSnapshot{
		NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
			EmailConfig: &storage.EmailNotifierConfiguration{
				NotifierId:    "notifier-456",
				MailingLists:  []string{"alerts@example.com"},
				CustomSubject: "Snapshot Subject",
				CustomBody:    "Snapshot Body",
			},
		},
	}

	v2Config := s.service.convertProtoNotifierSnapshotToV2(protoSnapshot)
	assert.NotNil(s.T(), v2Config)
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
	protoStatus := &storage.ReportStatus{
		RunState:                 storage.ReportStatus_GENERATED,
		ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
		ErrorMsg:                 "test error",
		CompletedAt:              nil,
	}

	v2Status := convertPrototoV2Reportstatus(protoStatus)
	assert.NotNil(s.T(), v2Status)
	assert.Equal(s.T(), apiV2.ReportStatus_GENERATED, v2Status.GetRunState())
	assert.Equal(s.T(), apiV2.NotificationMethod_DOWNLOAD, v2Status.GetReportNotificationMethod())
	assert.Equal(s.T(), "test error", v2Status.GetErrorMsg())
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

	protoFilters := s.service.convertV2NodeReportFiltersToProto(v2Filters, accessScopeRules)
	assert.NotNil(s.T(), protoFilters)
	assert.Equal(s.T(), "Cluster:prod", protoFilters.GetQuery())
	assert.True(s.T(), protoFilters.GetAllVuln())
	protoassert.SlicesEqual(s.T(), accessScopeRules, protoFilters.GetAccessScopeRules())

	v2FiltersBack := s.service.convertProtoNodeReportFiltersToV2(protoFilters)
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

	v2ConfigBack, err := s.service.convertProtoReportConfigurationToV2(protoConfig)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2ConfigBack)
	assert.Equal(s.T(), v2Config.GetId(), v2ConfigBack.GetId())
	assert.Equal(s.T(), v2Config.GetName(), v2ConfigBack.GetName())
	assert.Equal(s.T(), apiV2.ReportConfiguration_NODE_VULNERABILITY, v2ConfigBack.GetType())
}

func (s *ConversionTestSuite) TestNilConversions() {
	assert.Nil(s.T(), v2.ConvertV2ScheduleToProto(nil))
	assert.Nil(s.T(), v2.ConvertProtoScheduleToV2(nil))
	assert.Nil(s.T(), s.service.convertV2NotifierConfigToProto(nil))

	result, err := s.service.convertProtoNotifierConfigToV2(nil)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), result)

	assert.Nil(s.T(), s.service.convertProtoNotifierSnapshotToV2(nil))
	assert.Nil(s.T(), convertPrototoV2Reportstatus(nil))
	assert.Nil(s.T(), s.service.convertV2NodeReportFiltersToProto(nil, nil))
	assert.Nil(s.T(), s.service.convertProtoNodeReportFiltersToV2(nil))
}

func (s *ConversionTestSuite) TestEntityTypeEdgeCases() {
	assert.Equal(s.T(), storage.EntityType_ENTITY_TYPE_UNSET, v2EntityTypeToStorage(apiV2.ScopeEntity_SCOPE_ENTITY_UNSET))
	assert.Equal(s.T(), apiV2.ScopeEntity_SCOPE_ENTITY_UNSET, storageEntityTypeToV2(storage.EntityType_ENTITY_TYPE_UNSET))

	assert.Equal(s.T(), storage.EntityField_FIELD_UNSET, v2EntityFieldToStorage(apiV2.ScopeField_FIELD_UNSET))
	assert.Equal(s.T(), apiV2.ScopeField_FIELD_UNSET, storageEntityFieldToV2(storage.EntityField_FIELD_UNSET))

	assert.Equal(s.T(), storage.MatchType_EXACT, v2MatchTypeToStorage(apiV2.MatchType_EXACT))
	assert.Equal(s.T(), apiV2.MatchType_EXACT, storageMatchTypeToV2(storage.MatchType_EXACT))
}

func (s *ConversionTestSuite) TestReportSnapshotWithDownloadAvailable() {
	configID := "test-config"
	reportID := "test-report"

	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: configID,
		ReportId:              reportID,
		Name:                  "Test Report",
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
}

func (s *ConversionTestSuite) TestIntervalTypeConversions() {
	assert.Equal(s.T(), storage.Schedule_UNSET, v2IntervalTypeToStorage[apiV2.ReportSchedule_UNSET])
	assert.Equal(s.T(), storage.Schedule_DAILY, v2IntervalTypeToStorage[apiV2.ReportSchedule_DAILY])
	assert.Equal(s.T(), storage.Schedule_WEEKLY, v2IntervalTypeToStorage[apiV2.ReportSchedule_WEEKLY])
	assert.Equal(s.T(), storage.Schedule_MONTHLY, v2IntervalTypeToStorage[apiV2.ReportSchedule_MONTHLY])

	assert.Equal(s.T(), apiV2.ReportSchedule_UNSET, storageIntervalTypeToV2[storage.Schedule_UNSET])
	assert.Equal(s.T(), apiV2.ReportSchedule_DAILY, storageIntervalTypeToV2[storage.Schedule_DAILY])
	assert.Equal(s.T(), apiV2.ReportSchedule_WEEKLY, storageIntervalTypeToV2[storage.Schedule_WEEKLY])
	assert.Equal(s.T(), apiV2.ReportSchedule_MONTHLY, storageIntervalTypeToV2[storage.Schedule_MONTHLY])
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

	v2ScopeBack, err := s.service.convertProtoResourceScopeToV2(protoScope)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2ScopeBack)
	assert.NotNil(s.T(), v2ScopeBack.GetEntityScope())
	assert.Len(s.T(), v2ScopeBack.GetEntityScope().GetRules(), 1)
}

func (s *ConversionTestSuite) TestEmptyResourceScope() {
	result, err := s.service.convertV2ResourceScopeToProto(nil)
	assert.NoError(s.T(), err)
	assert.Nil(s.T(), result)

	v2Result, err := s.service.convertProtoResourceScopeToV2(nil)
	assert.NoError(s.T(), err)
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
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "notifier-1",
						MailingLists: []string{"team@example.com"},
					},
				},
			},
		},
	}

	v2Config, err := s.service.convertProtoReportConfigurationToV2(protoConfig)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), v2Config)
	assert.Equal(s.T(), "config-proto", v2Config.GetId())
	assert.Equal(s.T(), "Proto Config", v2Config.GetName())
	assert.Equal(s.T(), apiV2.ReportConfiguration_NODE_VULNERABILITY, v2Config.GetType())
	assert.Len(s.T(), v2Config.GetNotifiers(), 1)
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

	result, err := s.service.getExistingBlobNames(s.ctx, []*storage.ReportSnapshot{snapshot})
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

			result, err := s.service.getExistingBlobNames(s.ctx, []*storage.ReportSnapshot{snapshot})
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

	result, err := s.service.getExistingBlobNames(s.ctx, []*storage.ReportSnapshot{snapshot})
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

	result, err := s.service.getExistingBlobNames(s.ctx, []*storage.ReportSnapshot{snapshot})
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
			blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{
				{ID: common.GetReportBlobPath("config-1", "report-1")},
			}, nil).Times(1)

			result, err := s.service.getExistingBlobNames(s.ctx, []*storage.ReportSnapshot{snapshot})
			assert.NoError(s.T(), err)
			assert.True(s.T(), result.Contains(common.GetReportBlobPath("config-1", "report-1")))
		})
	}
}
