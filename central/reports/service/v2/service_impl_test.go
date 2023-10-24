package v2

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	blobDSMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDSMocks "github.com/stackrox/rox/central/reports/config/datastore/mocks"
	schedulerMocks "github.com/stackrox/rox/central/reports/scheduler/v2/mocks"
	reportSnapshotDSMocks "github.com/stackrox/rox/central/reports/snapshot/datastore/mocks"
	"github.com/stackrox/rox/central/reports/validation"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	withoutV1ConfigsQuery = search.NewQueryBuilder().AddExactMatches(search.EmbeddedCollectionID, "").ProtoQuery()
)

type upsertTestCase struct {
	desc                       string
	setMocksAndGenReportConfig func() *apiV2.ReportConfiguration
	reportConfigGen            func() *storage.ReportConfiguration
	isValidationError          bool
}

func TestReportService(t *testing.T) {
	suite.Run(t, new(ReportServiceTestSuite))
}

type ReportServiceTestSuite struct {
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

func (s *ReportServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")
	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip test when reporting enhancements are disabled")
		s.T().SkipNow()
	}
}

func (s *ReportServiceTestSuite) SetupTest() {
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

func (s *ReportServiceTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *ReportServiceTestSuite) TeardownTest() {
	s.mockCtrl.Finish()
}

func (s *ReportServiceTestSuite) TestCreateReportConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())
	s.scheduler.EXPECT().UpsertReportSchedule(gomock.Any()).Return(nil).AnyTimes()

	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}

	accessScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{ClusterName: "cluster-2", NamespaceName: "namespace-2"},
			},
		},
	}
	for _, tc := range s.upsertReportConfigTestCases(false) {
		s.T().Run(tc.desc, func(t *testing.T) {
			requestConfig := tc.setMocksAndGenReportConfig()
			mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
			ctx := authn.ContextWithIdentity(s.ctx, mockID, s.T())

			if !tc.isValidationError {
				mockID.EXPECT().UID().Return(creator.Id).AnyTimes()
				mockID.EXPECT().FullName().Return(creator.Name).AnyTimes()
				mockID.EXPECT().FriendlyName().Return(creator.Name).AnyTimes()

				mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(accessScope).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)

				protoReportConfig := tc.reportConfigGen()
				protoReportConfig.Creator = creator
				protoReportConfig.GetVulnReportFilters().AccessScopeRules = []*storage.SimpleAccessScope_Rules{accessScope.Rules}
				s.reportConfigDataStore.EXPECT().AddReportConfiguration(ctx, protoReportConfig).Return(protoReportConfig.GetId(), nil).Times(1)
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(ctx, protoReportConfig.GetId()).Return(protoReportConfig, true, nil).Times(1)
			}
			result, err := s.service.PostReportConfiguration(ctx, requestConfig)
			if tc.isValidationError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(requestConfig, result)
			}
		})
	}

	// Test error on context without user identity
	requestConfig := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
	_, err := s.service.PostReportConfiguration(allAccessContext, requestConfig)
	s.Error(err)
}

func (s *ReportServiceTestSuite) TestUpdateReportConfigurationError() {

	requester := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}

	status := &storage.ReportStatus{
		RunState: storage.ReportStatus_WAITING,
	}

	reportSnapshots := []*storage.ReportSnapshot{{
		ReportId:     "test_report",
		Name:         "test_report",
		ReportStatus: status,
		Requester:    requester,
	},
	}
	user := reportSnapshots[0].GetRequester()
	userContext := s.getContextForUser(user)

	protoReportConfig := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), gomock.Any()).
		Return(protoReportConfig, true, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return(reportSnapshots, nil).Times(1)
	s.collectionDataStore.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil).Times(1)
	requestConfig := &apiV2.ReportConfiguration{
		Id:   "test_rep",
		Name: "test_rep",
		ResourceScope: &apiV2.ResourceScope{
			ScopeReference: &apiV2.ResourceScope_CollectionScope{
				CollectionScope: &apiV2.CollectionReference{
					CollectionId:   "collection-test",
					CollectionName: "collection-test",
				},
			},
		},
		Filter: &apiV2.ReportConfiguration_VulnReportFilters{
			VulnReportFilters: &apiV2.VulnerabilityReportFilters{
				Fixability: apiV2.VulnerabilityReportFilters_FIXABLE,
				Severities: []apiV2.VulnerabilityReportFilters_VulnerabilitySeverity{apiV2.VulnerabilityReportFilters_CRITICAL_VULNERABILITY_SEVERITY},
				ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{
					apiV2.VulnerabilityReportFilters_DEPLOYED,
					apiV2.VulnerabilityReportFilters_WATCHED,
				},
				CvesSince: &apiV2.VulnerabilityReportFilters_SinceLastSentScheduledReport{SinceLastSentScheduledReport: true},
			},
		},
	}
	_, err := s.service.UpdateReportConfiguration(userContext, requestConfig)
	s.Error(err)

}

func (s *ReportServiceTestSuite) TestUpdateReportConfiguration() {
	s.scheduler.EXPECT().UpsertReportSchedule(gomock.Any()).Return(nil).AnyTimes()

	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	userContext := s.getContextForUser(creator)

	accessScopeRules := []*storage.SimpleAccessScope_Rules{
		{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{ClusterName: "cluster-2", NamespaceName: "namespace-2"},
			},
		},
	}
	for _, tc := range s.upsertReportConfigTestCases(true) {
		s.T().Run(tc.desc, func(t *testing.T) {
			requestConfig := tc.setMocksAndGenReportConfig()
			if !tc.isValidationError {
				protoReportConfig := tc.reportConfigGen()
				protoReportConfig.Creator = creator
				protoReportConfig.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(userContext, protoReportConfig.GetId()).
					Return(protoReportConfig, true, nil).Times(1)
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(userContext, gomock.Any()).Return([]*storage.ReportSnapshot{}, nil).Times(1)
				s.reportConfigDataStore.EXPECT().UpdateReportConfiguration(userContext, protoReportConfig).Return(nil).Times(1)
			}
			result, err := s.service.UpdateReportConfiguration(userContext, requestConfig)
			if tc.isValidationError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(&apiV2.Empty{}, result)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestListReportConfigurations() {
	allAccessContext := sac.WithAllAccess(context.Background())
	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *v1.Query
	}{
		{
			desc:  "Empty query",
			query: &apiV2.RawQuery{Query: ""},
			expectedQ: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.EmptyQuery(),
					withoutV1ConfigsQuery)
				query.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}
				return query
			}(),
		},
		{
			desc:  "Query with search field",
			query: &apiV2.RawQuery{Query: "Report Name:name"},
			expectedQ: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.NewQueryBuilder().AddStrings(search.ReportName, "name").ProtoQuery(),
					withoutV1ConfigsQuery)
				query.Pagination = &v1.QueryPagination{Limit: maxPaginationLimit}
				return query
			}(),
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 25},
			},
			expectedQ: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.EmptyQuery(),
					withoutV1ConfigsQuery)
				query.Pagination = &v1.QueryPagination{Limit: 25}
				return query
			}(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListReportConfigurationsResponse{
				ReportConfigs: []*apiV2.ReportConfiguration{fixtures.GetValidV2ReportConfigWithMultipleNotifiers()},
			}

			s.reportConfigDataStore.EXPECT().GetReportConfigurations(allAccessContext, tc.expectedQ).
				Return([]*storage.ReportConfiguration{fixtures.GetValidReportConfigWithMultipleNotifiersV2()}, nil).Times(1)

			s.mockGetNotifierCall(expectedResp.ReportConfigs[0].GetNotifiers()[0])
			s.mockGetNotifierCall(expectedResp.ReportConfigs[0].GetNotifiers()[1])
			s.mockGetCollectionCall(expectedResp.ReportConfigs[0])

			configs, err := s.service.ListReportConfigurations(allAccessContext, tc.query)
			s.NoError(err)
			s.Equal(expectedResp, configs)
		})
	}
}

func (s *ReportServiceTestSuite) TestGetReportConfigurationByID() {
	allAccessContext := sac.WithAllAccess(context.Background())
	testCases := []struct {
		desc                string
		id                  string
		isValidationError   bool
		isDataNotFoundError bool
	}{
		{
			desc:                "Empty ID",
			id:                  "",
			isValidationError:   true,
			isDataNotFoundError: false,
		},
		{
			desc:                "Config not found in datastore",
			id:                  "absent-id",
			isValidationError:   false,
			isDataNotFoundError: true,
		},
		{
			desc:                "valid ID",
			id:                  "present-id",
			isValidationError:   false,
			isDataNotFoundError: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			var expectedResp *apiV2.ReportConfiguration
			if !tc.isValidationError {
				if !tc.isDataNotFoundError {
					s.reportConfigDataStore.EXPECT().GetReportConfiguration(allAccessContext, tc.id).
						Return(fixtures.GetValidReportConfigWithMultipleNotifiersV2(), true, nil).Times(1)

					expectedResp = fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
					s.mockGetNotifierCall(expectedResp.GetNotifiers()[0])
					s.mockGetNotifierCall(expectedResp.GetNotifiers()[1])
					s.mockGetCollectionCall(expectedResp)
				} else {
					s.reportConfigDataStore.EXPECT().GetReportConfiguration(allAccessContext, tc.id).
						Return(nil, false, nil)
				}
			}

			config, err := s.service.GetReportConfiguration(allAccessContext, &apiV2.ResourceByID{Id: tc.id})
			if tc.isValidationError || tc.isDataNotFoundError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(expectedResp, config)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestCountReportConfigurations() {
	allAccessContext := sac.WithAllAccess(context.Background())
	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *v1.Query
	}{
		{
			desc:  "Empty query",
			query: &apiV2.RawQuery{Query: ""},
			expectedQ: search.ConjunctionQuery(
				search.NewQueryBuilder().ProtoQuery(),
				withoutV1ConfigsQuery),
		},
		{
			desc:  "Query with search field",
			query: &apiV2.RawQuery{Query: "Report Name:name"},
			expectedQ: search.ConjunctionQuery(
				search.NewQueryBuilder().AddStrings(search.ReportName, "name").ProtoQuery(),
				withoutV1ConfigsQuery),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			s.reportConfigDataStore.EXPECT().Count(allAccessContext, tc.expectedQ).Return(1, nil).Times(1)
			_, err := s.service.CountReportConfigurations(allAccessContext, tc.query)
			s.NoError(err)
		})
	}
}

func (s *ReportServiceTestSuite) TestDeleteReportConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())
	s.scheduler.EXPECT().RemoveReportSchedule(gomock.Any()).Return().AnyTimes()
	testCases := []struct {
		desc    string
		id      string
		isError bool
	}{
		{
			desc:    "Empty ID",
			id:      "",
			isError: true,
		},
		{
			desc:    "valid ID",
			id:      "config-id",
			isError: false,
		},
	}

	for _, tc := range testCases {
		if !tc.isError {
			s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), gomock.Any()).
				Return(fixtures.GetValidReportConfigWithMultipleNotifiersV2(), true, nil).Times(1)
			s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return([]*storage.ReportSnapshot{}, nil).Times(1)
			s.reportConfigDataStore.EXPECT().RemoveReportConfiguration(allAccessContext, tc.id).Return(nil).Times(1)
		}
		_, err := s.service.DeleteReportConfiguration(allAccessContext, &apiV2.ResourceByID{Id: tc.id})
		if tc.isError {
			s.Error(err)
		} else {
			s.NoError(err)
		}
	}
}

func (s *ReportServiceTestSuite) upsertReportConfigTestCases(isUpdate bool) []upsertTestCase {
	cases := []upsertTestCase{
		{
			desc: "Valid report config with multiple notifiers",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()

				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, false, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], true, false, isUpdate)

				s.mockCollectionStoreCalls(ret, true, false, isUpdate)

				return ret
			},
			reportConfigGen: func() *storage.ReportConfiguration {
				return fixtures.GetValidReportConfigWithMultipleNotifiersV2()
			},
			isValidationError: false,
		},
		{
			desc: "Valid report config without notifiers",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				ret.Schedule = nil

				s.mockCollectionStoreCalls(ret, true, false, isUpdate)
				return ret
			},
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Notifiers = nil
				ret.Schedule = nil
				return ret
			},
			isValidationError: false,
		},
		{
			desc: "Report config with invalid schedule : invalid day of week",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					Interval: &apiV2.ReportSchedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{
							Days: []int32{8},
						},
					},
				}
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid schedule : missing days of week",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					Interval: &apiV2.ReportSchedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{
							Days: []int32{},
						},
					},
				}
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid schedule : invalid day of month",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_MONTHLY,
					Interval: &apiV2.ReportSchedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{
							Days: []int32{30},
						},
					},
				}
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid schedule : missing days of month",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_MONTHLY,
					Interval: &apiV2.ReportSchedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{
							Days: nil,
						},
					},
				}
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : missing email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig = nil
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : empty notifierID in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.NotifierId = ""
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : empty mailing list in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.MailingLists = nil
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier: Custom email subject too long",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[0].GetEmailConfig().CustomSubject = strings.Repeat("a", env.ReportCustomEmailSubjectMaxLen.IntegerSetting()+1)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier: Custom email body too long",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[0].GetEmailConfig().CustomBody = strings.Repeat("a", env.ReportCustomEmailBodyMaxLen.IntegerSetting()+1)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : invalid email in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.MailingLists = []string{"sdfdksfjk"}
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : notifier not found",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], false, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with missing resource scope",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid resource scope : empty collectionID",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference.(*apiV2.ResourceScope_CollectionScope).CollectionScope.CollectionId = ""
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid resource scope : collection not found",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], true, true, isUpdate)

				s.mockCollectionStoreCalls(ret, false, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with missing vuln report filters",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters = nil
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], true, true, isUpdate)

				s.mockCollectionStoreCalls(ret, true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid vuln report filters : cvesSince unset",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters.CvesSince = nil
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], true, true, isUpdate)

				s.mockCollectionStoreCalls(ret, true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid vuln report filters : image types not set",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters.ImageTypes = nil
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[1], true, true, isUpdate)

				s.mockCollectionStoreCalls(ret, true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
	}

	if isUpdate {
		cases = append(cases, upsertTestCase{
			desc: "Report config with empty id",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Id = ""
				return ret
			},
			isValidationError: true,
		})
	}

	return cases
}

func (s *ReportServiceTestSuite) mockNotifierStoreCalls(reqNotifier *apiV2.NotifierConfiguration,
	notifierIDExits, isValidationError, isUpdate bool) {
	if notifierIDExits {
		s.notifierDataStore.EXPECT().Exists(gomock.Any(), reqNotifier.GetEmailConfig().GetNotifierId()).
			Return(true, nil).Times(1)
	} else {
		s.notifierDataStore.EXPECT().Exists(gomock.Any(), reqNotifier.GetEmailConfig().GetNotifierId()).
			Return(false, nil).Times(1)
	}

	if !isValidationError && !isUpdate {
		s.mockGetNotifierCall(reqNotifier)
	}
}

func (s *ReportServiceTestSuite) mockGetNotifierCall(reqNotifier *apiV2.NotifierConfiguration) {
	s.notifierDataStore.EXPECT().GetNotifier(gomock.Any(), reqNotifier.GetEmailConfig().GetNotifierId()).
		Return(&storage.Notifier{
			Id:   reqNotifier.GetEmailConfig().GetNotifierId(),
			Name: reqNotifier.GetNotifierName(),
		}, true, nil).Times(1)
}

func (s *ReportServiceTestSuite) mockCollectionStoreCalls(reqConfig *apiV2.ReportConfiguration,
	collectionIDExists, isValidationError, isUpdate bool) {
	if collectionIDExists {
		s.collectionDataStore.EXPECT().Exists(gomock.Any(), reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId()).
			Return(true, nil).Times(1)
	} else {
		s.collectionDataStore.EXPECT().Exists(gomock.Any(), reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId()).
			Return(false, nil).Times(1)
	}

	if !isValidationError && !isUpdate {
		s.mockGetCollectionCall(reqConfig)
	}
}

func (s *ReportServiceTestSuite) mockGetCollectionCall(reqConfig *apiV2.ReportConfiguration) {
	s.collectionDataStore.EXPECT().Get(gomock.Any(), reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId()).
		Return(&storage.ResourceCollection{
			Id:   reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId(),
			Name: reqConfig.GetResourceScope().GetCollectionScope().GetCollectionName(),
		}, true, nil).Times(1)
}

func (s *ReportServiceTestSuite) TestGetReportStatus() {
	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	snapshot := &storage.ReportSnapshot{
		ReportId:     "test_report",
		ReportStatus: status,
	}

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(snapshot, true, nil)
	id := apiV2.ResourceByID{
		Id: "test_report",
	}
	repStatusResponse, err := s.service.GetReportStatus(s.ctx, &id)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), repStatusResponse.Status.GetErrorMsg(), status.GetErrorMsg())
}

func (s *ReportServiceTestSuite) TestGetReportHistory() {
	reportSnapshot := &storage.ReportSnapshot{
		ReportId:              "test_report",
		ReportConfigurationId: "test_report_config",
		Name:                  "Report",
		ReportStatus: &storage.ReportStatus{
			ErrorMsg:                 "Error msg",
			ReportNotificationMethod: 1,
		},
	}

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return([]*storage.ReportSnapshot{reportSnapshot}, nil).AnyTimes()
	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
	emptyQuery := &apiV2.RawQuery{Query: ""}
	req := &apiV2.GetReportHistoryRequest{
		Id:               "test_report_config",
		ReportParamQuery: emptyQuery,
	}

	res, err := s.service.GetReportHistory(s.ctx, req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.ReportSnapshots[0].GetReportJobId(), "test_report")
	assert.Equal(s.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")

	req = &apiV2.GetReportHistoryRequest{
		Id:               "",
		ReportParamQuery: emptyQuery,
	}

	_, err = s.service.GetReportHistory(s.ctx, req)
	assert.Error(s.T(), err)

	query := &apiV2.RawQuery{Query: "Report Name:test_report"}
	req = &apiV2.GetReportHistoryRequest{
		Id:               "test_report_config",
		ReportParamQuery: query,
	}

	res, err = s.service.GetReportHistory(s.ctx, req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.ReportSnapshots[0].GetReportJobId(), "test_report")
	assert.Equal(s.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")
}

func (s *ReportServiceTestSuite) TestGetMyReportHistory() {
	userA := &storage.SlimUser{
		Id:   "user-a",
		Name: "user-a",
	}

	reportSnapshot := &storage.ReportSnapshot{
		ReportId:              "test_report",
		ReportConfigurationId: "test_report_config",
		Name:                  "Report",
		ReportStatus: &storage.ReportStatus{
			ErrorMsg:                 "Error msg",
			ReportNotificationMethod: 1,
		},
		Requester: &storage.SlimUser{
			Id: "user-a",
		},
	}

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{reportSnapshot}, nil).AnyTimes()
	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
	emptyQuery := &apiV2.RawQuery{Query: ""}
	req := &apiV2.GetReportHistoryRequest{
		Id:               "test_report_config",
		ReportParamQuery: emptyQuery,
	}

	res, err := s.service.GetMyReportHistory(s.getContextForUser(userA), req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.ReportSnapshots[0].GetReportJobId(), "test_report")
	assert.Equal(s.T(), res.ReportSnapshots[0].GetReportStatus().GetErrorMsg(), "Error msg")

	req = &apiV2.GetReportHistoryRequest{
		Id:               "",
		ReportParamQuery: emptyQuery,
	}
	_, err = s.service.GetMyReportHistory(s.getContextForUser(userA), req)
	assert.Error(s.T(), err)

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return(nil, nil).AnyTimes()
	emptyQuery = &apiV2.RawQuery{Query: ""}
	req = &apiV2.GetReportHistoryRequest{
		Id:               "test_report_config",
		ReportParamQuery: emptyQuery,
	}

	_, err = s.service.GetMyReportHistory(s.ctx, req)
	assert.Error(s.T(), err)
}

func (s *ReportServiceTestSuite) TestAuthz() {
	status := &storage.ReportStatus{
		ErrorMsg: "Error msg",
	}

	snapshot := &storage.ReportSnapshot{
		ReportId:     "test_report",
		ReportStatus: status,
	}
	snapshotDS := reportSnapshotDSMocks.NewMockDataStore(s.mockCtrl)
	snapshotDS.EXPECT().Get(gomock.Any(), gomock.Any()).Return(snapshot, true, nil).AnyTimes()
	metadataSlice := []*storage.ReportSnapshot{snapshot}
	snapshotDS.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return(metadataSlice, nil).AnyTimes()
	svc := serviceImpl{snapshotDatastore: snapshotDS}
	testutils.AssertAuthzWorks(s.T(), &svc)
}

func (s *ReportServiceTestSuite) TestRunReport() {
	reportConfig := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	notifierIDs := make([]string, 0, len(reportConfig.GetNotifiers()))
	notifiers := make([]*storage.Notifier, 0, len(reportConfig.GetNotifiers()))
	for _, nc := range reportConfig.GetNotifiers() {
		notifierIDs = append(notifierIDs, nc.GetId())
		notifiers = append(notifiers, &storage.Notifier{
			Id:   nc.GetEmailConfig().GetNotifierId(),
			Name: nc.GetEmailConfig().GetNotifierId(),
		})
	}
	collection := &storage.ResourceCollection{
		Id: reportConfig.GetResourceScope().GetCollectionId(),
	}

	user := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	accessScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{ClusterName: "cluster-2", NamespaceName: "namespace-2"},
			},
		},
	}

	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()

	mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	mockRole.EXPECT().GetAccessScope().Return(accessScope).AnyTimes()
	mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).AnyTimes()
	userContext := authn.ContextWithIdentity(s.ctx, mockID, s.T())

	testCases := []struct {
		desc    string
		req     *apiV2.RunReportRequest
		ctx     context.Context
		mockGen func()
		isError bool
		resp    *apiV2.RunReportResponse
	}{
		{
			desc: "Report config ID empty",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           "",
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx:     s.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report config not found",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Collection not found",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				s.collectionDataStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(nil, false, nil)
			},
			isError: true,
		},
		{
			desc: "One of the notifiers not found",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				s.collectionDataStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				s.notifierDataStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return([]*storage.Notifier{notifiers[0]}, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Successful submission; Notification method email",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			},
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				s.collectionDataStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				s.notifierDataStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return(notifiers, nil).Times(1)
				s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
					Return("reportID", nil).Times(1)
			},
			isError: false,
			resp: &apiV2.RunReportResponse{
				ReportConfigId: reportConfig.Id,
				ReportId:       "reportID",
			},
		},
		{
			desc: "Successful submission; Notification method download",
			req: &apiV2.RunReportRequest{
				ReportConfigId:           reportConfig.Id,
				ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
			},
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.Id).
					Return(reportConfig, true, nil).Times(1)
				s.collectionDataStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				s.notifierDataStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return(notifiers, nil).Times(1)
				s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
					Return("reportID", nil).Times(1)
			},
			isError: false,
			resp: &apiV2.RunReportResponse{
				ReportConfigId: reportConfig.Id,
				ReportId:       "reportID",
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.mockGen()
			response, err := s.service.RunReport(tc.ctx, tc.req)
			if tc.isError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(tc.resp, response)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestCancelReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.ReportId = uuid.NewV4().String()
	reportSnapshot.ReportStatus.RunState = storage.ReportStatus_WAITING
	user := reportSnapshot.GetRequester()
	userContext := s.getContextForUser(user)

	testCases := []struct {
		desc    string
		req     *apiV2.ResourceByID
		ctx     context.Context
		mockGen func()
		isError bool
	}{
		{
			desc: "Empty Report ID",
			req: &apiV2.ResourceByID{
				Id: "",
			},
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx:     s.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report requester id and cancelling user id mismatch",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.Requester = &storage.SlimUser{
					Id:   reportSnapshot.Requester.Id + "-1",
					Name: reportSnapshot.Requester.Name + "-1",
				}
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report is already delivered",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_DELIVERED
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report is already generated",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_GENERATED
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report already in PREPARING state",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_PREPARING
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Scheduler error while cancelling request",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.scheduler.EXPECT().CancelReportRequest(gomock.Any(), gomock.Any()).
					Return(false, errors.New("Datastore error")).Times(1)
			},
			isError: true,
		},
		{
			desc: "Scheduler couldn't find report ID in queue",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.scheduler.EXPECT().CancelReportRequest(gomock.Any(), gomock.Any()).
					Return(false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Request cancelled",
			req: &apiV2.ResourceByID{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.scheduler.EXPECT().CancelReportRequest(gomock.Any(), gomock.Any()).
					Return(true, nil).Times(1)
			},
			isError: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.mockGen()
			response, err := s.service.CancelReport(tc.ctx, tc.req)
			if tc.isError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(&apiV2.Empty{}, response)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestDeleteReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.ReportId = uuid.NewV4().String()
	reportSnapshot.ReportConfigurationId = uuid.NewV4().String()
	reportSnapshot.ReportStatus.RunState = storage.ReportStatus_DELIVERED
	reportSnapshot.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
	user := reportSnapshot.GetRequester()
	userContext := s.getContextForUser(user)
	blobName := common.GetReportBlobPath(reportSnapshot.GetReportConfigurationId(), reportSnapshot.GetReportId())

	testCases := []struct {
		desc    string
		req     *apiV2.DeleteReportRequest
		ctx     context.Context
		mockGen func()
		isError bool
	}{
		{
			desc: "Empty Report ID",
			req: &apiV2.DeleteReportRequest{
				Id: "",
			},
			ctx:     userContext,
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx:     s.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Delete requester user id and report requester user id mismatch",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.Requester = &storage.SlimUser{
					Id:   reportSnapshot.Requester.Id + "-1",
					Name: reportSnapshot.Requester.Name + "-1",
				}
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report was not generated",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_EMAIL
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report job has not completed",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_PREPARING
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Delete blob failed",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.blobStore.EXPECT().Delete(gomock.Any(), blobName).Times(1).Return(errors.New(""))
			},
			isError: true,
		},
		{
			desc: "Report deleted",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(reportSnapshot, true, nil).Times(1)
				s.blobStore.EXPECT().Delete(gomock.Any(), blobName).Times(1).Return(nil)
			},
			isError: false,
		},
		{
			desc: "Generated but not downloaded report deleted",
			req: &apiV2.DeleteReportRequest{
				Id: reportSnapshot.GetReportId(),
			},
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.Clone()
				snap.ReportStatus.RunState = storage.ReportStatus_GENERATED
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), snap.GetReportId()).
					Return(snap, true, nil).Times(1)
				s.blobStore.EXPECT().Delete(gomock.Any(), blobName).Times(1).Return(nil)
			},
			isError: false,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			if tc.mockGen != nil {
				tc.mockGen()
			}
			response, err := s.service.DeleteReport(tc.ctx, tc.req)
			if tc.isError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(&apiV2.Empty{}, response)
			}
		})
	}
}

func (s *ReportServiceTestSuite) getContextForUser(user *storage.SlimUser) context.Context {
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()
	return authn.ContextWithIdentity(s.ctx, mockID, s.T())
}
