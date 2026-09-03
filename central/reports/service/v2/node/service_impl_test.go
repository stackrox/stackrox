package node

import (
	"context"
	"testing"

	blobDSMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDSMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNodeReportService(t *testing.T) {
	suite.Run(t, new(NodeReportServiceTestSuite))
}

type NodeReportServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                     context.Context
	reportConfigDataStore   *reportConfigDSMocks.MockDataStore
	reportSnapshotDataStore *reportSnapshotDSMocks.MockDataStore
	collectionDataStore     *collectionDSMocks.MockDataStore
	notifierDataStore       *notifierDSMocks.MockDataStore
	blobStore               *blobDSMocks.MockDatastore
	scheduler               *schedulerMocks.MockScheduler
	service                 Service
}

func (s *NodeReportServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
	s.reportConfigDataStore = reportConfigDSMocks.NewMockDataStore(s.mockCtrl)
	s.reportSnapshotDataStore = reportSnapshotDSMocks.NewMockDataStore(s.mockCtrl)
	s.collectionDataStore = collectionDSMocks.NewMockDataStore(s.mockCtrl)
	s.notifierDataStore = notifierDSMocks.NewMockDataStore(s.mockCtrl)
	s.blobStore = blobDSMocks.NewMockDatastore(s.mockCtrl)
	s.scheduler = schedulerMocks.NewMockScheduler(s.mockCtrl)
	validator := validation.New(s.reportConfigDataStore, s.reportSnapshotDataStore, s.collectionDataStore, s.notifierDataStore)
	s.service = New(s.reportConfigDataStore, s.reportSnapshotDataStore, s.collectionDataStore, s.notifierDataStore, s.scheduler, s.blobStore, validator)
}

func (s *NodeReportServiceTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *NodeReportServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NodeReportServiceTestSuite) getContextForUser(user *storage.SlimUser) context.Context {
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return(user.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(user.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.GetName()).AnyTimes()
	mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	mockRole.EXPECT().GetAccessScope().Return(&storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-1"},
		},
	}).AnyTimes()
	mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).AnyTimes()
	return authn.ContextWithIdentity(s.ctx, mockID, s.T())
}

func (s *NodeReportServiceTestSuite) getValidNodeReportConfig() *apiV2.ReportConfiguration {
	return &apiV2.ReportConfiguration{
		Id:          uuid.NewV4().String(),
		Name:        "test node report",
		Description: "test description",
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
				Query: "Cluster:cluster-1",
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
		Notifiers: []*apiV2.NotifierConfiguration{
			{
				NotifierConfig: &apiV2.NotifierConfiguration_EmailConfig{
					EmailConfig: &apiV2.EmailNotifierConfiguration{
						NotifierId:   "email-notifier-id",
						MailingLists: []string{"test@example.com"},
					},
				},
			},
		},
	}
}

func (s *NodeReportServiceTestSuite) TestPostNodeReportConfiguration() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}

	accessScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-1"},
		},
	}

	requestConfig := s.getValidNodeReportConfig()
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	ctx := authn.ContextWithIdentity(s.ctx, mockID, s.T())

	mockID.EXPECT().UID().Return(creator.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(creator.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(creator.GetName()).AnyTimes()

	mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	mockRole.EXPECT().GetAccessScope().Return(accessScope).Times(1)
	mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)

	s.notifierDataStore.EXPECT().Exists(gomock.Any(), "email-notifier-id").Return(true, nil).Times(1)

	s.reportConfigDataStore.EXPECT().AddReportConfiguration(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, cfg *storage.ReportConfiguration) (string, error) {
			s.Equal(storage.ReportConfiguration_NODE_VULNERABILITY, cfg.GetType())
			protoassert.Equal(s.T(), creator, cfg.GetCreator())
			s.NotNil(cfg.GetNodeVulnReportFilters())
			return cfg.GetId(), nil
		}).Times(1)

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), requestConfig.GetId()).
		DoAndReturn(func(_ context.Context, _ string) (*storage.ReportConfiguration, bool, error) {
			return &storage.ReportConfiguration{
				Id:          requestConfig.GetId(),
				Name:        requestConfig.GetName(),
				Description: requestConfig.GetDescription(),
				Type:        storage.ReportConfiguration_NODE_VULNERABILITY,
				Creator:     creator,
				Filter: &storage.ReportConfiguration_NodeVulnReportFilters{
					NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{
						Query: "Cluster:cluster-1",
						CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{
							AllVuln: true,
						},
					},
				},
			}, true, nil
		}).Times(1)

	s.scheduler.EXPECT().UpsertReportSchedule(gomock.Any()).Return(nil).Times(1)

	result, err := s.service.PostNodeReportConfiguration(ctx, requestConfig)
	s.NoError(err)
	s.Equal(requestConfig.GetId(), result.GetId())
	s.Equal(apiV2.ReportConfiguration_NODE_VULNERABILITY, result.GetType())
}

func (s *NodeReportServiceTestSuite) TestPostNodeReportConfiguration_ValidationError() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	invalidConfig := s.getValidNodeReportConfig()
	invalidConfig.Filter = &apiV2.ReportConfiguration_NodeVulnReportFilters{
		NodeVulnReportFilters: &apiV2.NodeVulnerabilityReportFilters{
			Query:     "Cluster:cluster-1",
			CvesSince: nil,
		},
	}

	s.notifierDataStore.EXPECT().Exists(gomock.Any(), "email-notifier-id").Return(true, nil).Times(1)

	_, err := s.service.PostNodeReportConfiguration(ctx, invalidConfig)
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestGetNodeReportConfiguration() {
	protoReportConfig := &storage.ReportConfiguration{
		Id:          "test-id",
		Name:        "test node report",
		Description: "test description",
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
				Query: "Cluster:cluster-1",
				CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
	}

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "test-id").
		Return(protoReportConfig, true, nil).Times(1)

	result, err := s.service.GetNodeReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: "test-id"})
	s.NoError(err)
	s.Equal("test-id", result.GetId())
	s.Equal(apiV2.ReportConfiguration_NODE_VULNERABILITY, result.GetType())
}

func (s *NodeReportServiceTestSuite) TestGetNodeReportConfiguration_NotFound() {
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "nonexistent").
		Return(nil, false, nil).Times(1)

	_, err := s.service.GetNodeReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: "nonexistent"})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestDeleteNodeReportConfiguration() {
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "test-id").
		Return(&storage.ReportConfiguration{
			Id:   "test-id",
			Type: storage.ReportConfiguration_NODE_VULNERABILITY,
		}, true, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{}, nil).Times(1)
	s.reportConfigDataStore.EXPECT().RemoveReportConfiguration(gomock.Any(), "test-id").
		Return(nil).Times(1)
	s.scheduler.EXPECT().RemoveReportSchedule("test-id").Times(1)

	_, err := s.service.DeleteNodeReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: "test-id"})
	s.NoError(err)
}

func (s *NodeReportServiceTestSuite) TestDeleteNodeReportConfiguration_HasRunningJob() {
	runningSnapshot := &storage.ReportSnapshot{
		ReportId: "test-report-id",
		ReportStatus: &storage.ReportStatus{
			RunState: storage.ReportStatus_PREPARING,
		},
	}

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "test-id").
		Return(&storage.ReportConfiguration{
			Id:   "test-id",
			Type: storage.ReportConfiguration_NODE_VULNERABILITY,
		}, true, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{runningSnapshot}, nil).Times(1)

	_, err := s.service.DeleteNodeReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: "test-id"})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestRunNodeReport() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	configID := "test-config-id"
	protoReportConfig := &storage.ReportConfiguration{
		Id:          configID,
		Name:        "test node report",
		Description: "test description",
		Type:        storage.ReportConfiguration_NODE_VULNERABILITY,
		Creator:     creator,
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
				Query: "Cluster:cluster-1",
				CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
		Notifiers: []*storage.NotifierConfiguration{
			{
				NotifierConfig: &storage.NotifierConfiguration_EmailConfig{
					EmailConfig: &storage.EmailNotifierConfiguration{
						NotifierId:   "email-notifier-id",
						MailingLists: []string{"test@example.com"},
					},
				},
			},
		},
	}

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), configID).
		Return(protoReportConfig, true, nil).AnyTimes()

	s.notifierDataStore.EXPECT().GetManyNotifiers(gomock.Any(), gomock.Any()).
		Return([]*storage.Notifier{
			{
				Id:   "email-notifier-id",
				Name: "Email Notifier",
				Type: "email",
			},
		}, nil).AnyTimes()

	s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
		DoAndReturn(func(_ context.Context, reportReq interface{}, _ bool) (string, error) {
			return "on-demand-report-id", nil
		}).Times(1)

	result, err := s.service.RunNodeReport(ctx, &apiV2.RunReportRequest{
		ReportConfigId: configID,
	})
	s.NoError(err)
	s.Equal(configID, result.GetReportConfigId())
	s.Equal("on-demand-report-id", result.GetReportId())
}

func (s *NodeReportServiceTestSuite) TestCancelNodeReport() {
	reportID := "test-report-id"
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	snapshot := &storage.ReportSnapshot{
		ReportId:              reportID,
		ReportConfigurationId: "test-config-id",
		Requester:             creator,
		ReportStatus: &storage.ReportStatus{
			RunState: storage.ReportStatus_PREPARING,
		},
		Type: storage.ReportSnapshot_NODE_VULNERABILITY,
	}

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportID).
		Return(snapshot, true, nil).Times(1)

	s.scheduler.EXPECT().CancelReportRequest(gomock.Any(), reportID).
		Return(false, nil).Times(1)

	_, err := s.service.CancelNodeReport(ctx, &apiV2.ResourceByID{Id: reportID})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestDeleteNodeReport() {
	reportID := "test-report-id"
	configID := "test-config-id"
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	snapshot := &storage.ReportSnapshot{
		ReportId:              reportID,
		ReportConfigurationId: configID,
		Requester:             creator,
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
		},
		Type: storage.ReportSnapshot_NODE_VULNERABILITY,
	}

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportID).
		Return(snapshot, true, nil).Times(1)

	s.reportSnapshotDataStore.EXPECT().DeleteReportSnapshot(gomock.Any(), reportID).
		Return(nil).Times(1)

	blobPath := common.GetReportBlobPath(configID, reportID)
	s.blobStore.EXPECT().Delete(gomock.Any(), blobPath).Return(nil).Times(1)

	_, err := s.service.DeleteNodeReport(ctx, &apiV2.DeleteReportRequest{
		Id: reportID,
	})
	s.NoError(err)
}

func (s *NodeReportServiceTestSuite) TestPostViewBasedNodeReport() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}

	req := &apiV2.ReportRequestViewBased{
		Type: apiV2.ReportRequestViewBased_NODE_VULNERABILITY,
		Filter: &apiV2.ReportRequestViewBased_NodeVulnReportFilters{
			NodeVulnReportFilters: &apiV2.NodeVulnerabilityReportFilters{
				Query: "Cluster:cluster-1",
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
	}

	mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return(creator.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(creator.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(creator.GetName()).AnyTimes()
	mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).AnyTimes()
	mockRole.EXPECT().GetAccessScope().Return(&storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-1"},
		},
	}).AnyTimes()

	ctxWithIdentity := authn.ContextWithIdentity(s.ctx, mockID, s.T())

	s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
		DoAndReturn(func(_ context.Context, reportReq interface{}, _ bool) (string, error) {
			return "view-based-report-id", nil
		}).Times(1)

	result, err := s.service.PostViewBasedNodeReport(ctxWithIdentity, req)
	s.NoError(err)
	s.NotEmpty(result.GetReportID())
	s.Equal("view-based-report-id", result.GetReportID())
}

func (s *NodeReportServiceTestSuite) TestListNodeReportConfigurations() {
	protoReportConfigs := []*storage.ReportConfiguration{
		{
			Id:   "config-1",
			Name: "Node Report 1",
			Type: storage.ReportConfiguration_NODE_VULNERABILITY,
		},
		{
			Id:   "config-2",
			Name: "Node Report 2",
			Type: storage.ReportConfiguration_NODE_VULNERABILITY,
		},
	}

	s.reportConfigDataStore.EXPECT().GetReportConfigurations(gomock.Any(), gomock.Any()).
		Return(protoReportConfigs, nil).Times(1)

	result, err := s.service.ListNodeReportConfigurations(s.ctx, &apiV2.RawQuery{Query: ""})
	s.NoError(err)
	s.Len(result.GetReportConfigs(), 2)
	assert.Equal(s.T(), apiV2.ReportConfiguration_NODE_VULNERABILITY, result.GetReportConfigs()[0].GetType())
}

func (s *NodeReportServiceTestSuite) TestCountNodeReportConfigurations() {
	s.reportConfigDataStore.EXPECT().Count(gomock.Any(), gomock.Any()).
		Return(5, nil).Times(1)

	result, err := s.service.CountNodeReportConfigurations(s.ctx, &apiV2.RawQuery{Query: ""})
	s.NoError(err)
	s.Equal(int32(5), result.GetCount())
}

func (s *NodeReportServiceTestSuite) TestGetMyNodeReportHistory() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	configID := "test-config-id"
	// Query-time filtering: only return creator's snapshot
	snapshots := []*storage.ReportSnapshot{
		{
			ReportId:              "report-1",
			ReportConfigurationId: configID,
			Name:                  "Report 1",
			Requester:             creator,
			Type:                  storage.ReportSnapshot_NODE_VULNERABILITY,
		},
	}

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return(snapshots, nil).Times(1)

	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()

	result, err := s.service.GetMyNodeReportHistory(ctx, &apiV2.GetReportHistoryRequest{
		Id: configID,
		ReportParamQuery: &apiV2.RawQuery{
			Query: "",
		},
	})
	s.NoError(err)
	s.Len(result.GetReportSnapshots(), 1)
	s.Equal(creator.GetId(), result.GetReportSnapshots()[0].GetUser().GetId())
}

func (s *NodeReportServiceTestSuite) TestUpdateNodeReportConfiguration() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}

	accessScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-1"},
		},
	}

	updateConfig := s.getValidNodeReportConfig()
	updateConfig.Name = "Updated Node Report"

	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	ctx := authn.ContextWithIdentity(s.ctx, mockID, s.T())

	mockID.EXPECT().UID().Return(creator.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(creator.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(creator.GetName()).AnyTimes()

	mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	mockRole.EXPECT().GetAccessScope().Return(accessScope).AnyTimes()
	mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).AnyTimes()

	s.notifierDataStore.EXPECT().Exists(gomock.Any(), "email-notifier-id").Return(true, nil).Times(1)

	existingConfig := &storage.ReportConfiguration{
		Id:      updateConfig.GetId(),
		Name:    "Old Name",
		Type:    storage.ReportConfiguration_NODE_VULNERABILITY,
		Creator: creator,
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
				Query: "Cluster:cluster-1",
				CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{
					AllVuln: true,
				},
			},
		},
	}
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), updateConfig.GetId()).
		Return(existingConfig, true, nil).Times(1)

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{}, nil).Times(1)

	s.reportConfigDataStore.EXPECT().UpdateReportConfiguration(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, cfg *storage.ReportConfiguration) error {
			s.Equal("Updated Node Report", cfg.GetName())
			return nil
		}).Times(1)

	s.scheduler.EXPECT().UpsertReportSchedule(gomock.Any()).Return(nil).Times(1)

	result, err := s.service.UpdateNodeReportConfiguration(ctx, updateConfig)
	s.NoError(err)
	s.NotNil(result)
}

func (s *NodeReportServiceTestSuite) TestGetNodeReportHistory() {
	configID := "test-config-id"
	snapshots := []*storage.ReportSnapshot{
		{
			ReportId:              "report-1",
			ReportConfigurationId: configID,
			Name:                  "Report 1",
			Requester:             &storage.SlimUser{Id: "uid", Name: "name"},
			Type:                  storage.ReportSnapshot_NODE_VULNERABILITY,
			ResourceScope: &storage.ResourceScope{
				ScopeReference: &storage.ResourceScope_EntityScope{
					EntityScope: &storage.EntityScope{},
				},
			},
		},
	}

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return(snapshots, nil).Times(1)

	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()

	result, err := s.service.GetNodeReportHistory(s.ctx, &apiV2.GetReportHistoryRequest{
		Id: configID,
		ReportParamQuery: &apiV2.RawQuery{
			Query: "",
		},
	})
	s.NoError(err)
	s.Len(result.GetReportSnapshots(), 1)
	s.Equal("report-1", result.GetReportSnapshots()[0].GetReportJobId())
}

func (s *NodeReportServiceTestSuite) TestGetNodeReportStatus() {
	reportID := "test-report-id"
	snapshot := &storage.ReportSnapshot{
		ReportId: reportID,
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
		},
		Type: storage.ReportSnapshot_NODE_VULNERABILITY,
	}

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportID).
		Return(snapshot, true, nil).Times(1)

	result, err := s.service.GetNodeReportStatus(s.ctx, &apiV2.ResourceByID{Id: reportID})
	s.NoError(err)
	s.NotNil(result.GetStatus())
	s.Equal(apiV2.ReportStatus_GENERATED, result.GetStatus().GetRunState())
}

func (s *NodeReportServiceTestSuite) TestGetViewBasedNodeReportHistory() {
	snapshots := []*storage.ReportSnapshot{
		{
			ReportId:              "view-report-1",
			ReportConfigurationId: "",
			Name:                  "View Report 1",
			Requester:             &storage.SlimUser{Id: "uid", Name: "name"},
			Type:                  storage.ReportSnapshot_NODE_VULNERABILITY,
			ResourceScope: &storage.ResourceScope{
				ScopeReference: &storage.ResourceScope_EntityScope{
					EntityScope: &storage.EntityScope{},
				},
			},
		},
	}

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return(snapshots, nil).Times(1)

	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()

	result, err := s.service.GetViewBasedNodeReportHistory(s.ctx, &apiV2.GetViewBasedReportHistoryRequest{
		ReportParamQuery: &apiV2.RawQuery{
			Query: "",
		},
	})
	s.NoError(err)
	s.Len(result.GetReportSnapshots(), 1)
}

func (s *NodeReportServiceTestSuite) TestGetViewBasedMyNodeReportHistory() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	// Query-time filtering: only return creator's snapshot
	snapshots := []*storage.ReportSnapshot{
		{
			ReportId:              "view-report-1",
			ReportConfigurationId: "",
			Name:                  "View Report 1",
			Requester:             creator,
			Type:                  storage.ReportSnapshot_NODE_VULNERABILITY,
			ResourceScope: &storage.ResourceScope{
				ScopeReference: &storage.ResourceScope_EntityScope{
					EntityScope: &storage.EntityScope{},
				},
			},
		},
	}

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return(snapshots, nil).Times(1)

	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()

	result, err := s.service.GetViewBasedMyNodeReportHistory(ctx, &apiV2.GetViewBasedReportHistoryRequest{
		ReportParamQuery: &apiV2.RawQuery{
			Query: "",
		},
	})
	s.NoError(err)
	s.Len(result.GetReportSnapshots(), 1)
	s.Equal(creator.GetId(), result.GetReportSnapshots()[0].GetUser().GetId())
}

func (s *NodeReportServiceTestSuite) TestGetNodeReportConfiguration_InvalidType() {
	protoReportConfig := &storage.ReportConfiguration{
		Id:   "test-id",
		Type: storage.ReportConfiguration_VULNERABILITY,
	}

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "test-id").
		Return(protoReportConfig, true, nil).Times(1)

	_, err := s.service.GetNodeReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: "test-id"})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestDeleteNodeReportConfiguration_WrongType() {
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "test-id").
		Return(&storage.ReportConfiguration{
			Id:   "test-id",
			Type: storage.ReportConfiguration_VULNERABILITY,
		}, true, nil).Times(1)

	_, err := s.service.DeleteNodeReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: "test-id"})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestRunNodeReport_WrongType() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "config-id").
		Return(&storage.ReportConfiguration{
			Id:   "config-id",
			Type: storage.ReportConfiguration_VULNERABILITY,
		}, true, nil).Times(1)

	_, err := s.service.RunNodeReport(ctx, &apiV2.RunReportRequest{
		ReportConfigId: "config-id",
	})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestUpdateNodeReportConfiguration_WrongType() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	wrongConfig := s.getValidNodeReportConfig()
	wrongConfig.Type = apiV2.ReportConfiguration_VULNERABILITY

	_, err := s.service.UpdateNodeReportConfiguration(ctx, wrongConfig)
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestPostNodeReportConfiguration_WrongType() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	wrongConfig := s.getValidNodeReportConfig()
	wrongConfig.Type = apiV2.ReportConfiguration_VULNERABILITY

	_, err := s.service.PostNodeReportConfiguration(ctx, wrongConfig)
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestCancelNodeReport_NotFound() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), "nonexistent").
		Return(nil, false, nil).Times(1)

	_, err := s.service.CancelNodeReport(ctx, &apiV2.ResourceByID{Id: "nonexistent"})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestGetNodeReportStatus_NotFound() {
	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), "nonexistent").
		Return(nil, false, nil).Times(1)

	_, err := s.service.GetNodeReportStatus(s.ctx, &apiV2.ResourceByID{Id: "nonexistent"})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestCancelNodeReport_WrongState() {
	reportID := "test-report-id"
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	snapshot := &storage.ReportSnapshot{
		ReportId:              reportID,
		ReportConfigurationId: "test-config-id",
		Requester:             creator,
		ReportStatus: &storage.ReportStatus{
			RunState: storage.ReportStatus_DELIVERED,
		},
		Type: storage.ReportSnapshot_NODE_VULNERABILITY,
	}

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportID).
		Return(snapshot, true, nil).Times(1)

	_, err := s.service.CancelNodeReport(ctx, &apiV2.ResourceByID{Id: reportID})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestCancelNodeReport_WrongType() {
	reportID := "test-report-id"
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	snapshot := &storage.ReportSnapshot{
		ReportId:              reportID,
		ReportConfigurationId: "test-config-id",
		Requester:             creator,
		ReportStatus: &storage.ReportStatus{
			RunState: storage.ReportStatus_PREPARING,
		},
		Type: storage.ReportSnapshot_VULNERABILITY,
	}

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportID).
		Return(snapshot, true, nil).Times(1)

	_, err := s.service.CancelNodeReport(ctx, &apiV2.ResourceByID{Id: reportID})
	s.Error(err)
	s.Contains(err.Error(), "not a node vulnerability report")
}

func (s *NodeReportServiceTestSuite) TestPostViewBasedNodeReport_WrongType() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	req := &apiV2.ReportRequestViewBased{
		Type: apiV2.ReportRequestViewBased_VULNERABILITY,
	}

	_, err := s.service.PostViewBasedNodeReport(ctx, req)
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestDeleteNodeReport_NotFound() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), "nonexistent").
		Return(nil, false, nil).Times(1)

	_, err := s.service.DeleteNodeReport(ctx, &apiV2.DeleteReportRequest{
		Id: "nonexistent",
	})
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestDeleteNodeReport_WrongType() {
	reportID := "test-report-id"
	configID := "test-config-id"
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	snapshot := &storage.ReportSnapshot{
		ReportId:              reportID,
		ReportConfigurationId: configID,
		Requester:             creator,
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_GENERATED,
			ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
		},
		Type: storage.ReportSnapshot_VULNERABILITY,
	}

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportID).
		Return(snapshot, true, nil).Times(1)

	_, err := s.service.DeleteNodeReport(ctx, &apiV2.DeleteReportRequest{
		Id: reportID,
	})
	s.Error(err)
	s.Contains(err.Error(), "not a node vulnerability report")
}

func (s *NodeReportServiceTestSuite) TestUpdateNodeReportConfiguration_NotFound() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	updateConfig := s.getValidNodeReportConfig()

	s.notifierDataStore.EXPECT().Exists(gomock.Any(), "email-notifier-id").Return(true, nil).Times(1)
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), updateConfig.GetId()).
		Return(nil, false, nil).Times(1)

	_, err := s.service.UpdateNodeReportConfiguration(ctx, updateConfig)
	s.Error(err)
}

func (s *NodeReportServiceTestSuite) TestRunNodeReport_ConfigNotFound() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	ctx := s.getContextForUser(creator)

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "nonexistent").
		Return(nil, false, nil).Times(1)

	_, err := s.service.RunNodeReport(ctx, &apiV2.RunReportRequest{
		ReportConfigId: "nonexistent",
	})
	s.Error(err)
}
