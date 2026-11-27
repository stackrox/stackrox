package node

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/notifiers"
	pgNotify "github.com/stackrox/rox/pkg/postgres/notify"
	"github.com/stackrox/rox/pkg/sac/resources"
	"go.uber.org/mock/gomock"
)

func (s *NodeReportServiceTestSuite) expectNotify(ctx context.Context, channel, payload string) {
	s.db.EXPECT().Exec(ctx, "SELECT pg_notify($1, $2)", channel, payload).
		Return(pgconn.NewCommandTag("SELECT 1"), nil).Times(1)
}

func (s *NodeReportServiceTestSuite) TestPostNodeReportConfigurationWithCentralWorker() {
	s.T().Setenv(env.CentralWorkerEnabled.EnvVar(), "true")

	creator := &storage.SlimUser{Id: "uid", Name: "name"}
	accessScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{IncludedClusters: []string{"cluster-1"}},
	}
	requestConfig := s.getValidNodeReportConfig()
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	ctx := authn.ContextWithIdentity(s.ctx, mockID, s.T())
	mockID.EXPECT().UID().Return(creator.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(creator.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(creator.GetName()).AnyTimes()
	mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	mockRole.EXPECT().GetAccessScope().Return(accessScope).Times(1)
	mockRole.EXPECT().GetPermissions().Return(map[string]storage.Access{
		resources.Node.String(): storage.Access_READ_ACCESS,
	}).Times(1)
	mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)

	s.notifierDataStore.EXPECT().GetScrubbedNotifier(gomock.Any(), "email-notifier-id").Return(&storage.Notifier{Id: "email-notifier-id", Type: notifiers.EmailType}, true, nil).Times(1)
	s.reportConfigDataStore.EXPECT().AddReportConfiguration(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, cfg *storage.ReportConfiguration) (string, error) {
			return cfg.GetId(), nil
		}).Times(1)
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), requestConfig.GetId()).
		Return(&storage.ReportConfiguration{
			Id:      requestConfig.GetId(),
			Name:    requestConfig.GetName(),
			Type:    storage.ReportConfiguration_NODE_VULNERABILITY,
			Creator: creator,
			Filter: &storage.ReportConfiguration_NodeVulnReportFilters{
				NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{
					Query:     "Cluster:cluster-1",
					CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{AllVuln: true},
				},
			},
		}, true, nil).Times(1)
	s.expectNotify(ctx, pgNotify.ReportConfigChanged, requestConfig.GetId())

	result, err := s.service.PostNodeReportConfiguration(ctx, requestConfig)
	s.NoError(err)
	s.Equal(requestConfig.GetId(), result.GetId())
}

func (s *NodeReportServiceTestSuite) TestUpdateNodeReportConfigurationWithCentralWorker() {
	s.T().Setenv(env.CentralWorkerEnabled.EnvVar(), "true")

	creator := &storage.SlimUser{Id: "uid", Name: "name"}
	updateConfig := s.getValidNodeReportConfig()
	ctx := s.getContextForUser(creator)

	s.notifierDataStore.EXPECT().GetScrubbedNotifier(gomock.Any(), "email-notifier-id").Return(&storage.Notifier{Id: "email-notifier-id", Type: notifiers.EmailType}, true, nil).Times(1)
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), updateConfig.GetId()).
		Return(&storage.ReportConfiguration{
			Id:      updateConfig.GetId(),
			Name:    "Old Name",
			Type:    storage.ReportConfiguration_NODE_VULNERABILITY,
			Creator: creator,
			Filter: &storage.ReportConfiguration_NodeVulnReportFilters{
				NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{
					Query:     "Cluster:cluster-1",
					CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{AllVuln: true},
				},
			},
		}, true, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{}, nil).Times(1)
	s.reportConfigDataStore.EXPECT().UpdateReportConfiguration(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.expectNotify(ctx, pgNotify.ReportConfigChanged, updateConfig.GetId())

	_, err := s.service.UpdateNodeReportConfiguration(ctx, updateConfig)
	s.NoError(err)
}

func (s *NodeReportServiceTestSuite) TestDeleteNodeReportConfigurationWithCentralWorker() {
	s.T().Setenv(env.CentralWorkerEnabled.EnvVar(), "true")

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), "test-id").
		Return(&storage.ReportConfiguration{
			Id:   "test-id",
			Type: storage.ReportConfiguration_NODE_VULNERABILITY,
		}, true, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{}, nil).Times(1)
	s.reportConfigDataStore.EXPECT().RemoveReportConfiguration(gomock.Any(), "test-id").
		Return(nil).Times(1)
	s.expectNotify(s.ctx, pgNotify.ReportConfigChanged, "test-id")

	_, err := s.service.DeleteNodeReportConfiguration(s.ctx, &apiV2.ResourceByID{Id: "test-id"})
	s.NoError(err)
}

func (s *NodeReportServiceTestSuite) TestRunNodeReportWithCentralWorker() {
	s.T().Setenv(env.CentralWorkerEnabled.EnvVar(), "true")

	creator := &storage.SlimUser{Id: "uid", Name: "name"}
	ctx := s.getContextForUser(creator)
	configID := "test-config-id"
	protoReportConfig := &storage.ReportConfiguration{
		Id:      configID,
		Name:    "test node report",
		Type:    storage.ReportConfiguration_NODE_VULNERABILITY,
		Creator: creator,
		ResourceScope: &storage.ResourceScope{
			ScopeReference: &storage.ResourceScope_EntityScope{
				EntityScope: &storage.EntityScope{
					Rules: []*storage.EntityScopeRule{
						{
							Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
							Field:  storage.EntityField_FIELD_ID,
							Values: []*storage.RuleValue{{Value: "cluster-1", MatchType: storage.MatchType_EXACT}},
						},
					},
				},
			},
		},
		Filter: &storage.ReportConfiguration_NodeVulnReportFilters{
			NodeVulnReportFilters: &storage.NodeVulnerabilityReportFilters{
				Query:     "Cluster:cluster-1",
				CvesSince: &storage.NodeVulnerabilityReportFilters_AllVuln{AllVuln: true},
			},
		},
	}

	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), configID).
		Return(protoReportConfig, true, nil).AnyTimes()
	s.notifierDataStore.EXPECT().GetManyNotifiers(gomock.Any(), gomock.Any()).
		Return(nil, nil).AnyTimes()
	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{}, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().AddReportSnapshot(gomock.Any(), gomock.Any()).
		Return("on-demand-report-id", nil).Times(1)
	s.db.EXPECT().Exec(gomock.Any(), "SELECT pg_notify($1, $2)", pgNotify.ReportRequestSubmitted, "on-demand-report-id").
		Return(pgconn.NewCommandTag("SELECT 1"), nil).Times(1)

	result, err := s.service.RunNodeReport(ctx, &apiV2.RunReportRequest{
		ReportConfigId:           configID,
		ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
	})
	s.NoError(err)
	s.Equal("on-demand-report-id", result.GetReportId())
}

func (s *NodeReportServiceTestSuite) TestPostViewBasedNodeReportWithCentralWorker() {
	s.T().Setenv(env.CentralWorkerEnabled.EnvVar(), "true")

	creator := &storage.SlimUser{Id: "uid", Name: "name"}
	ctx := s.getContextForUser(creator)
	req := &apiV2.ReportRequestViewBased{
		Type: apiV2.ReportRequestViewBased_NODE_VULNERABILITY,
		Filter: &apiV2.ReportRequestViewBased_NodeVulnReportFilters{
			NodeVulnReportFilters: &apiV2.NodeVulnerabilityReportFilters{
				Query:     "Cluster:cluster-1",
				CvesSince: &apiV2.NodeVulnerabilityReportFilters_AllVuln{AllVuln: true},
			},
		},
	}

	s.reportSnapshotDataStore.EXPECT().Count(gomock.Any(), gomock.Any()).
		Return(0, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().AddReportSnapshot(gomock.Any(), gomock.Any()).
		Return("view-based-report-id", nil).Times(1)
	s.db.EXPECT().Exec(gomock.Any(), "SELECT pg_notify($1, $2)", pgNotify.ReportRequestSubmitted, "view-based-report-id").
		Return(pgconn.NewCommandTag("SELECT 1"), nil).Times(1)

	result, err := s.service.PostViewBasedNodeReport(ctx, req)
	s.NoError(err)
	s.Equal("view-based-report-id", result.GetReportID())
}

func (s *NodeReportServiceTestSuite) TestCancelNodeReportWithCentralWorker() {
	s.T().Setenv(env.CentralWorkerEnabled.EnvVar(), "true")

	reportID := "test-report-id"
	creator := &storage.SlimUser{Id: "uid", Name: "name"}
	ctx := s.getContextForUser(creator)
	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportID).
		Return(&storage.ReportSnapshot{
			ReportId:  reportID,
			Requester: creator,
			ReportStatus: &storage.ReportStatus{
				RunState: storage.ReportStatus_PREPARING,
			},
			Type: storage.ReportSnapshot_NODE_VULNERABILITY,
		}, true, nil).Times(1)
	s.expectNotify(ctx, pgNotify.ReportRequestCancelled, reportID)

	_, err := s.service.CancelNodeReport(ctx, &apiV2.ResourceByID{Id: reportID})
	s.NoError(err)
}
