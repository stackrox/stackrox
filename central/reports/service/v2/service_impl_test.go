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
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
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

	creator := &storage.SlimUser{}
	creator.SetId("uid")
	creator.SetName("name")

	accessScope := storage.SimpleAccessScope_builder{
		Rules: storage.SimpleAccessScope_Rules_builder{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				storage.SimpleAccessScope_Rules_Namespace_builder{ClusterName: "cluster-2", NamespaceName: "namespace-2"}.Build(),
			},
		}.Build(),
	}.Build()
	for _, tc := range s.upsertReportConfigTestCases(false) {
		s.T().Run(tc.desc, func(t *testing.T) {
			requestConfig := tc.setMocksAndGenReportConfig()
			mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
			ctx := authn.ContextWithIdentity(s.ctx, mockID, s.T())

			if !tc.isValidationError {
				mockID.EXPECT().UID().Return(creator.GetId()).AnyTimes()
				mockID.EXPECT().FullName().Return(creator.GetName()).AnyTimes()
				mockID.EXPECT().FriendlyName().Return(creator.GetName()).AnyTimes()

				mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
				mockRole.EXPECT().GetAccessScope().Return(accessScope).Times(1)
				mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).Times(1)

				protoReportConfig := tc.reportConfigGen()
				protoReportConfig.SetCreator(creator)
				protoReportConfig.GetVulnReportFilters().SetAccessScopeRules([]*storage.SimpleAccessScope_Rules{accessScope.GetRules()})
				s.reportConfigDataStore.EXPECT().AddReportConfiguration(ctx, protoReportConfig).Return(protoReportConfig.GetId(), nil).Times(1)
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(ctx, protoReportConfig.GetId()).Return(protoReportConfig, true, nil).Times(1)
			}
			result, err := s.service.PostReportConfiguration(ctx, requestConfig)
			if tc.isValidationError {
				s.Error(err)
			} else {
				s.NoError(err)
				protoassert.Equal(s.T(), requestConfig, result)
			}
		})
	}

	// Test error on context without user identity
	requestConfig := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
	_, err := s.service.PostReportConfiguration(allAccessContext, requestConfig)
	s.Error(err)
}

func (s *ReportServiceTestSuite) TestUpdateReportConfigurationError() {

	requester := &storage.SlimUser{}
	requester.SetId("uid")
	requester.SetName("name")

	status := &storage.ReportStatus{}
	status.SetRunState(storage.ReportStatus_WAITING)

	rs := &storage.ReportSnapshot{}
	rs.SetReportId("test_report")
	rs.SetName("test_report")
	rs.SetReportStatus(status)
	rs.SetRequester(requester)
	reportSnapshots := []*storage.ReportSnapshot{rs}
	user := reportSnapshots[0].GetRequester()
	userContext := s.getContextForUser(user)

	protoReportConfig := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
	s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), gomock.Any()).
		Return(protoReportConfig, true, nil).Times(1)
	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return(reportSnapshots, nil).Times(1)
	s.collectionDataStore.EXPECT().Exists(gomock.Any(), gomock.Any()).Return(true, nil).Times(1)
	requestConfig := apiV2.ReportConfiguration_builder{
		Id:   "test_rep",
		Name: "test_rep",
		ResourceScope: apiV2.ResourceScope_builder{
			CollectionScope: apiV2.CollectionReference_builder{
				CollectionId:   "collection-test",
				CollectionName: "collection-test",
			}.Build(),
		}.Build(),
		VulnReportFilters: apiV2.VulnerabilityReportFilters_builder{
			Fixability: apiV2.VulnerabilityReportFilters_FIXABLE,
			Severities: []apiV2.VulnerabilityReportFilters_VulnerabilitySeverity{apiV2.VulnerabilityReportFilters_CRITICAL_VULNERABILITY_SEVERITY},
			ImageTypes: []apiV2.VulnerabilityReportFilters_ImageType{
				apiV2.VulnerabilityReportFilters_DEPLOYED,
				apiV2.VulnerabilityReportFilters_WATCHED,
			},
			SinceLastSentScheduledReport: proto.Bool(true),
		}.Build(),
	}.Build()
	_, err := s.service.UpdateReportConfiguration(userContext, requestConfig)
	s.Error(err)

}

func (s *ReportServiceTestSuite) TestUpdateReportConfiguration() {
	s.scheduler.EXPECT().UpsertReportSchedule(gomock.Any()).Return(nil).AnyTimes()

	creator := &storage.SlimUser{}
	creator.SetId("uid")
	creator.SetName("name")
	userContext := s.getContextForUser(creator)

	accessScopeRules := []*storage.SimpleAccessScope_Rules{
		storage.SimpleAccessScope_Rules_builder{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				storage.SimpleAccessScope_Rules_Namespace_builder{ClusterName: "cluster-2", NamespaceName: "namespace-2"}.Build(),
			},
		}.Build(),
	}
	for _, tc := range s.upsertReportConfigTestCases(true) {
		s.T().Run(tc.desc, func(t *testing.T) {
			requestConfig := tc.setMocksAndGenReportConfig()
			if !tc.isValidationError {
				protoReportConfig := tc.reportConfigGen()
				protoReportConfig.SetCreator(creator)
				protoReportConfig.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
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
				protoassert.Equal(s.T(), &apiV2.Empty{}, result)
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
			query: apiV2.RawQuery_builder{Query: ""}.Build(),
			expectedQ: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.EmptyQuery(),
					withoutV1ConfigsQuery)
				query.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())
				return query
			}(),
		},
		{
			desc:  "Query with search field",
			query: apiV2.RawQuery_builder{Query: "Report Name:name"}.Build(),
			expectedQ: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.NewQueryBuilder().AddStrings(search.ReportName, "name").ProtoQuery(),
					withoutV1ConfigsQuery)
				query.SetPagination(v1.QueryPagination_builder{Limit: maxPaginationLimit}.Build())
				return query
			}(),
		},
		{
			desc: "Query with custom pagination",
			query: apiV2.RawQuery_builder{
				Query:      "",
				Pagination: apiV2.Pagination_builder{Limit: 25}.Build(),
			}.Build(),
			expectedQ: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.EmptyQuery(),
					withoutV1ConfigsQuery)
				query.SetPagination(v1.QueryPagination_builder{Limit: 25}.Build())
				return query
			}(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListReportConfigurationsResponse{}
			expectedResp.SetReportConfigs([]*apiV2.ReportConfiguration{fixtures.GetValidV2ReportConfigWithMultipleNotifiers()})

			s.reportConfigDataStore.EXPECT().GetReportConfigurations(allAccessContext, tc.expectedQ).
				Return([]*storage.ReportConfiguration{fixtures.GetValidReportConfigWithMultipleNotifiersV2()}, nil).Times(1)

			s.mockGetNotifierCall(expectedResp.GetReportConfigs()[0].GetNotifiers()[0])
			s.mockGetNotifierCall(expectedResp.GetReportConfigs()[0].GetNotifiers()[1])
			s.mockGetCollectionCall(expectedResp.GetReportConfigs()[0])

			configs, err := s.service.ListReportConfigurations(allAccessContext, tc.query)
			s.NoError(err)
			protoassert.Equal(s.T(), expectedResp, configs)
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

			rbid := &apiV2.ResourceByID{}
			rbid.SetId(tc.id)
			config, err := s.service.GetReportConfiguration(allAccessContext, rbid)
			if tc.isValidationError || tc.isDataNotFoundError {
				s.Error(err)
			} else {
				s.NoError(err)
				protoassert.Equal(s.T(), expectedResp, config)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestCountReportConfigurations() {
	allAccessContext := sac.WithAllAccess(context.Background())
	rawQuery := &apiV2.RawQuery{}
	rawQuery.SetQuery("")
	rawQuery2 := &apiV2.RawQuery{}
	rawQuery2.SetQuery("Report Name:name")
	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *v1.Query
	}{
		{
			desc:  "Empty query",
			query: rawQuery,
			expectedQ: search.ConjunctionQuery(
				search.NewQueryBuilder().ProtoQuery(),
				withoutV1ConfigsQuery),
		},
		{
			desc:  "Query with search field",
			query: rawQuery2,
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
		rbid := &apiV2.ResourceByID{}
		rbid.SetId(tc.id)
		_, err := s.service.DeleteReportConfiguration(allAccessContext, rbid)
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
				ret.SetNotifiers(nil)
				ret.ClearSchedule()

				s.mockCollectionStoreCalls(ret, true, false, isUpdate)
				return ret
			},
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.SetNotifiers(nil)
				ret.ClearSchedule()
				return ret
			},
			isValidationError: false,
		},
		{
			desc: "Report config with invalid schedule : invalid day of week",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.SetSchedule(apiV2.ReportSchedule_builder{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					DaysOfWeek: apiV2.ReportSchedule_DaysOfWeek_builder{
						Days: []int32{8},
					}.Build(),
				}.Build())
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid schedule : missing days of week",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.SetSchedule(apiV2.ReportSchedule_builder{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					DaysOfWeek: apiV2.ReportSchedule_DaysOfWeek_builder{
						Days: []int32{},
					}.Build(),
				}.Build())
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid schedule : invalid day of month",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.SetSchedule(apiV2.ReportSchedule_builder{
					IntervalType: apiV2.ReportSchedule_MONTHLY,
					DaysOfMonth: apiV2.ReportSchedule_DaysOfMonth_builder{
						Days: []int32{30},
					}.Build(),
				}.Build())
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid schedule : missing days of month",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.SetSchedule(apiV2.ReportSchedule_builder{
					IntervalType: apiV2.ReportSchedule_MONTHLY,
					DaysOfMonth: apiV2.ReportSchedule_DaysOfMonth_builder{
						Days: nil,
					}.Build(),
				}.Build())
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : missing email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig = nil
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : empty notifierID in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.SetNotifierId("")
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : empty mailing list in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.SetMailingLists(nil)
				s.mockNotifierStoreCalls(ret.GetNotifiers()[0], true, true, isUpdate)
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier: Custom email subject too long",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomSubject(strings.Repeat("a", env.ReportCustomEmailSubjectMaxLen.IntegerSetting()+1))
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier: Custom email body too long",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomBody(strings.Repeat("a", env.ReportCustomEmailBodyMaxLen.IntegerSetting()+1))
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : invalid email in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.SetMailingLists([]string{"sdfdksfjk"})
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
				ret.ClearResourceScope()
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
				ret.GetResourceScope().ScopeReference.(*apiV2.ResourceScope_CollectionScope).CollectionScope.SetCollectionId("")
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
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters.ClearCvesSince()
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
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters.SetImageTypes(nil)
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
				ret.SetId("")
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
	notifier := &storage.Notifier{}
	notifier.SetId(reqNotifier.GetEmailConfig().GetNotifierId())
	notifier.SetName(reqNotifier.GetNotifierName())
	s.notifierDataStore.EXPECT().GetNotifier(gomock.Any(), reqNotifier.GetEmailConfig().GetNotifierId()).
		Return(notifier, true, nil).Times(1)
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
	rc := &storage.ResourceCollection{}
	rc.SetId(reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId())
	rc.SetName(reqConfig.GetResourceScope().GetCollectionScope().GetCollectionName())
	s.collectionDataStore.EXPECT().Get(gomock.Any(), reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId()).
		Return(rc, true, nil).Times(1)
}

func (s *ReportServiceTestSuite) TestGetReportStatus() {
	status := &storage.ReportStatus{}
	status.SetErrorMsg("Error msg")

	snapshot := &storage.ReportSnapshot{}
	snapshot.SetReportId("test_report")
	snapshot.SetReportStatus(status)

	s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(snapshot, true, nil)
	id := &apiV2.ResourceByID{}
	id.SetId("test_report")
	repStatusResponse, err := s.service.GetReportStatus(s.ctx, &id)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), repStatusResponse.GetStatus().GetErrorMsg(), status.GetErrorMsg())
}

func (s *ReportServiceTestSuite) TestGetReportHistory() {
	rs := &storage.ReportStatus{}
	rs.SetErrorMsg("Error msg")
	rs.SetReportNotificationMethod(1)
	reportSnapshot := &storage.ReportSnapshot{}
	reportSnapshot.SetReportId("test_report")
	reportSnapshot.SetReportConfigurationId("test_report_config")
	reportSnapshot.SetName("Report")
	reportSnapshot.SetReportStatus(rs)

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).Return([]*storage.ReportSnapshot{reportSnapshot}, nil).AnyTimes()
	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
	emptyQuery := &apiV2.RawQuery{}
	emptyQuery.SetQuery("")
	req := &apiV2.GetReportHistoryRequest{}
	req.SetId("test_report_config")
	req.SetReportParamQuery(emptyQuery)

	res, err := s.service.GetReportHistory(s.ctx, req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.GetReportSnapshots()[0].GetReportJobId(), "test_report")
	assert.Equal(s.T(), res.GetReportSnapshots()[0].GetReportStatus().GetErrorMsg(), "Error msg")

	req = &apiV2.GetReportHistoryRequest{}
	req.SetId("")
	req.SetReportParamQuery(emptyQuery)

	_, err = s.service.GetReportHistory(s.ctx, req)
	assert.Error(s.T(), err)

	query := &apiV2.RawQuery{}
	query.SetQuery("Report Name:test_report")
	req = &apiV2.GetReportHistoryRequest{}
	req.SetId("test_report_config")
	req.SetReportParamQuery(query)

	res, err = s.service.GetReportHistory(s.ctx, req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.GetReportSnapshots()[0].GetReportJobId(), "test_report")
	assert.Equal(s.T(), res.GetReportSnapshots()[0].GetReportStatus().GetErrorMsg(), "Error msg")
}

func (s *ReportServiceTestSuite) TestGetMyReportHistory() {
	userA := &storage.SlimUser{}
	userA.SetId("user-a")
	userA.SetName("user-a")

	rs := &storage.ReportStatus{}
	rs.SetErrorMsg("Error msg")
	rs.SetReportNotificationMethod(1)
	slimUser := &storage.SlimUser{}
	slimUser.SetId("user-a")
	reportSnapshot := &storage.ReportSnapshot{}
	reportSnapshot.SetReportId("test_report")
	reportSnapshot.SetReportConfigurationId("test_report_config")
	reportSnapshot.SetName("Report")
	reportSnapshot.SetReportStatus(rs)
	reportSnapshot.SetRequester(slimUser)

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return([]*storage.ReportSnapshot{reportSnapshot}, nil).AnyTimes()
	s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
	emptyQuery := &apiV2.RawQuery{}
	emptyQuery.SetQuery("")
	req := &apiV2.GetReportHistoryRequest{}
	req.SetId("test_report_config")
	req.SetReportParamQuery(emptyQuery)

	res, err := s.service.GetMyReportHistory(s.getContextForUser(userA), req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), res.GetReportSnapshots()[0].GetReportJobId(), "test_report")
	assert.Equal(s.T(), res.GetReportSnapshots()[0].GetReportStatus().GetErrorMsg(), "Error msg")

	req = &apiV2.GetReportHistoryRequest{}
	req.SetId("")
	req.SetReportParamQuery(emptyQuery)
	_, err = s.service.GetMyReportHistory(s.getContextForUser(userA), req)
	assert.Error(s.T(), err)

	s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
		Return(nil, nil).AnyTimes()
	emptyQuery = &apiV2.RawQuery{}
	emptyQuery.SetQuery("")
	req = &apiV2.GetReportHistoryRequest{}
	req.SetId("test_report_config")
	req.SetReportParamQuery(emptyQuery)

	_, err = s.service.GetMyReportHistory(s.ctx, req)
	assert.Error(s.T(), err)
}

func (s *ReportServiceTestSuite) TestAuthz() {
	status := &storage.ReportStatus{}
	status.SetErrorMsg("Error msg")

	snapshot := &storage.ReportSnapshot{}
	snapshot.SetReportId("test_report")
	snapshot.SetReportStatus(status)
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
		notifier := &storage.Notifier{}
		notifier.SetId(nc.GetEmailConfig().GetNotifierId())
		notifier.SetName(nc.GetEmailConfig().GetNotifierId())
		notifiers = append(notifiers, notifier)
	}
	collection := &storage.ResourceCollection{}
	collection.SetId(reportConfig.GetResourceScope().GetCollectionId())

	user := &storage.SlimUser{}
	user.SetId("uid")
	user.SetName("name")
	accessScope := storage.SimpleAccessScope_builder{
		Rules: storage.SimpleAccessScope_Rules_builder{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				storage.SimpleAccessScope_Rules_Namespace_builder{ClusterName: "cluster-2", NamespaceName: "namespace-2"}.Build(),
			},
		}.Build(),
	}.Build()

	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return(user.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(user.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.GetName()).AnyTimes()

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
			req: apiV2.RunReportRequest_builder{
				ReportConfigId:           "",
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			}.Build(),
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: apiV2.RunReportRequest_builder{
				ReportConfigId:           reportConfig.GetId(),
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			}.Build(),
			ctx:     s.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report config not found",
			req: apiV2.RunReportRequest_builder{
				ReportConfigId:           reportConfig.GetId(),
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.GetId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Collection not found",
			req: apiV2.RunReportRequest_builder{
				ReportConfigId:           reportConfig.GetId(),
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.GetId()).
					Return(reportConfig, true, nil).Times(1)
				s.collectionDataStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(nil, false, nil)
			},
			isError: true,
		},
		{
			desc: "One of the notifiers not found",
			req: apiV2.RunReportRequest_builder{
				ReportConfigId:           reportConfig.GetId(),
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.GetId()).
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
			req: apiV2.RunReportRequest_builder{
				ReportConfigId:           reportConfig.GetId(),
				ReportNotificationMethod: apiV2.NotificationMethod_EMAIL,
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.GetId()).
					Return(reportConfig, true, nil).Times(1)
				s.collectionDataStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				s.notifierDataStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return(notifiers, nil).Times(1)
				s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
					Return("reportID", nil).Times(1)
			},
			isError: false,
			resp: apiV2.RunReportResponse_builder{
				ReportConfigId: reportConfig.GetId(),
				ReportId:       "reportID",
			}.Build(),
		},
		{
			desc: "Successful submission; Notification method download",
			req: apiV2.RunReportRequest_builder{
				ReportConfigId:           reportConfig.GetId(),
				ReportNotificationMethod: apiV2.NotificationMethod_DOWNLOAD,
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportConfigDataStore.EXPECT().GetReportConfiguration(gomock.Any(), reportConfig.GetId()).
					Return(reportConfig, true, nil).Times(1)
				s.collectionDataStore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(collection, true, nil).Times(1)
				s.notifierDataStore.EXPECT().GetManyNotifiers(gomock.Any(), notifierIDs).
					Return(notifiers, nil).Times(1)
				s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
					Return("reportID", nil).Times(1)
			},
			isError: false,
			resp: apiV2.RunReportResponse_builder{
				ReportConfigId: reportConfig.GetId(),
				ReportId:       "reportID",
			}.Build(),
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
				protoassert.Equal(s.T(), tc.resp, response)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestCancelReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.SetReportId(uuid.NewV4().String())
	reportSnapshot.GetReportStatus().SetRunState(storage.ReportStatus_WAITING)
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
			req: apiV2.ResourceByID_builder{
				Id: "",
			}.Build(),
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx:     s.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report requester id and cancelling user id mismatch",
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.SetRequester(storage.SlimUser_builder{
					Id:   reportSnapshot.GetRequester().GetId() + "-1",
					Name: reportSnapshot.GetRequester().GetName() + "-1",
				}.Build())
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report is already delivered",
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.GetReportStatus().SetRunState(storage.ReportStatus_DELIVERED)
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report is already generated",
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.GetReportStatus().SetRunState(storage.ReportStatus_GENERATED)
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report already in PREPARING state",
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.GetReportStatus().SetRunState(storage.ReportStatus_PREPARING)
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Scheduler error while cancelling request",
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
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
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
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
			req: apiV2.ResourceByID_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
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
				protoassert.Equal(s.T(), &apiV2.Empty{}, response)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestDeleteReport() {
	reportSnapshot := fixtures.GetReportSnapshot()
	reportSnapshot.SetReportId(uuid.NewV4().String())
	reportSnapshot.SetReportConfigurationId(uuid.NewV4().String())
	reportSnapshot.GetReportStatus().SetRunState(storage.ReportStatus_DELIVERED)
	reportSnapshot.GetReportStatus().SetReportNotificationMethod(storage.ReportStatus_DOWNLOAD)
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
			req: apiV2.DeleteReportRequest_builder{
				Id: "",
			}.Build(),
			ctx:     userContext,
			isError: true,
		},
		{
			desc: "User info not present in context",
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx:     s.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Report ID not found",
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(nil, false, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Delete requester user id and report requester user id mismatch",
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.SetRequester(storage.SlimUser_builder{
					Id:   reportSnapshot.GetRequester().GetId() + "-1",
					Name: reportSnapshot.GetRequester().GetName() + "-1",
				}.Build())
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report was not generated",
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.GetReportStatus().SetReportNotificationMethod(storage.ReportStatus_EMAIL)
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Report job has not completed",
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.GetReportStatus().SetRunState(storage.ReportStatus_PREPARING)
				s.reportSnapshotDataStore.EXPECT().Get(gomock.Any(), reportSnapshot.GetReportId()).
					Return(snap, true, nil).Times(1)
			},
			isError: true,
		},
		{
			desc: "Delete blob failed",
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
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
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
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
			req: apiV2.DeleteReportRequest_builder{
				Id: reportSnapshot.GetReportId(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				snap := reportSnapshot.CloneVT()
				snap.GetReportStatus().SetRunState(storage.ReportStatus_GENERATED)
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
				protoassert.Equal(s.T(), &apiV2.Empty{}, response)
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestPostViewBasedReport() {
	// Enable the view-based reports feature flag for this test
	s.T().Setenv(features.VulnerabilityViewBasedReports.EnvVar(), "true")

	user := &storage.SlimUser{}
	user.SetId("test-user-id")
	user.SetName("test-user-name")
	userContext := s.getContextForUser(user)

	vbvrf := &apiV2.ViewBasedVulnerabilityReportFilters{}
	vbvrf.SetQuery("CVE Severity:CRITICAL")
	validRequest := &apiV2.ReportRequestViewBased{}
	validRequest.SetType(apiV2.ReportRequestViewBased_VULNERABILITY)
	validRequest.SetViewBasedVulnReportFilters(proto.ValueOrDefault(vbvrf))
	validRequest.SetAreaOfConcern("User Workloads")

	testCases := []struct {
		desc    string
		req     *apiV2.ReportRequestViewBased
		ctx     context.Context
		mockGen func()
		isError bool
		resp    *apiV2.RunReportResponseViewBased
	}{
		{
			desc:    "Nil request",
			req:     nil,
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc:    "User info not present in context",
			req:     validRequest,
			ctx:     s.ctx,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Unsupported report type",
			req: apiV2.ReportRequestViewBased_builder{
				Type: 1,
				ViewBasedVulnReportFilters: apiV2.ViewBasedVulnerabilityReportFilters_builder{
					Query: "",
				}.Build(),
			}.Build(),
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Missing view-based vulnerability report filters",
			req: apiV2.ReportRequestViewBased_builder{
				Type:   apiV2.ReportRequestViewBased_VULNERABILITY,
				Filter: nil,
			}.Build(),
			ctx:     userContext,
			mockGen: func() {},
			isError: true,
		},
		{
			desc: "Scheduler error",
			req:  validRequest,
			ctx:  userContext,
			mockGen: func() {
				s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
					Return("", errors.New("scheduler error")).Times(1)
			},
			isError: true,
		},
		{
			desc: "Successful submission with all fields",
			req:  validRequest,
			ctx:  userContext,
			mockGen: func() {
				s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
					Return("reportID123", nil).Times(1)
			},
			isError: false,
			resp: apiV2.RunReportResponseViewBased_builder{
				ReportID: "reportID123",
			}.Build(),
		},
		{
			desc: "Successful submission with only deployed images",
			req: apiV2.ReportRequestViewBased_builder{
				Type: apiV2.ReportRequestViewBased_VULNERABILITY,
				ViewBasedVulnReportFilters: apiV2.ViewBasedVulnerabilityReportFilters_builder{
					Query: "CVE Severity:CRITICAL,IMPORTANT",
				}.Build(),
				AreaOfConcern: "High severity vulnerabilities",
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.scheduler.EXPECT().SubmitReportRequest(gomock.Any(), gomock.Any(), false).
					Return("reportID789", nil).Times(1)
			},
			isError: false,
			resp: apiV2.RunReportResponseViewBased_builder{
				ReportID: "reportID789",
			}.Build(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			tc.mockGen()
			response, err := s.service.PostViewBasedReport(tc.ctx, tc.req)
			if tc.isError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(response.GetReportID(), tc.resp.GetReportID())
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestGetViewBasedReportHistory() {
	// Enable the view-based reports feature flag for this test
	s.T().Setenv(features.VulnerabilityViewBasedReports.EnvVar(), "true")

	testCases := []struct {
		desc    string
		req     *apiV2.GetViewBasedReportHistoryRequest
		mockGen func()
		isError bool
	}{
		{
			desc: "Datastore error",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("datastore error")).Times(1)
			},
			isError: true,
		},
		{
			desc: "Successful request with empty query",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			mockGen: func() {
				reportSnapshot := storage.ReportSnapshot_builder{
					ReportId:              "test-report-id",
					ReportConfigurationId: "test-config-id",
					Name:                  "View Based Report",
					ReportStatus: storage.ReportStatus_builder{
						ErrorMsg:                 "",
						ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
						RunState:                 storage.ReportStatus_GENERATED,
						ReportRequestType:        storage.ReportStatus_VIEW_BASED,
					}.Build(),
				}.Build()
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
					Return([]*storage.ReportSnapshot{reportSnapshot}, nil).Times(1)
				s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
			},
			isError: false,
		},
		{
			desc: "Successful request with custom query",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{Query: "Report Name:test"}.Build(),
			}.Build(),
			mockGen: func() {
				reportSnapshot := storage.ReportSnapshot_builder{
					ReportId:              "test-report-id",
					ReportConfigurationId: "test-config-id",
					Name:                  "View Based Test Report",
					ReportStatus: storage.ReportStatus_builder{
						ErrorMsg:                 "",
						ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
						RunState:                 storage.ReportStatus_GENERATED,
						ReportRequestType:        storage.ReportStatus_VIEW_BASED,
					}.Build(),
				}.Build()
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
					Return([]*storage.ReportSnapshot{reportSnapshot}, nil).Times(1)
				s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
			},
			isError: false,
		},
		{
			desc: "Successful request with pagination",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{
					Query: "",
					Pagination: apiV2.Pagination_builder{
						Limit:  10,
						Offset: 0,
					}.Build(),
				}.Build(),
			}.Build(),
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
					Return([]*storage.ReportSnapshot{}, nil).Times(1)
				s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
			},
			isError: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			if tc.mockGen != nil {
				tc.mockGen()
			}
			response, err := s.service.GetViewBasedReportHistory(s.ctx, tc.req)
			if tc.isError {
				s.Error(err)
				s.Nil(response)
			} else {
				s.NoError(err)
				s.NotNil(response)
				s.NotNil(response.GetReportSnapshots())
			}
		})
	}
}

func (s *ReportServiceTestSuite) TestGetViewBasedMyReportHistory() {
	// Enable the view-based reports feature flag for this test
	s.T().Setenv(features.VulnerabilityViewBasedReports.EnvVar(), "true")

	userA := &storage.SlimUser{}
	userA.SetId("user-a")
	userA.SetName("user-a")
	userContext := s.getContextForUser(userA)

	testCases := []struct {
		desc    string
		req     *apiV2.GetViewBasedReportHistoryRequest
		ctx     context.Context
		mockGen func()
		isError bool
	}{
		{
			desc: "User info not present in context",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			ctx:     s.ctx,
			isError: true,
		},
		{
			desc: "Datastore error",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("datastore error")).Times(1)
			},
			isError: true,
		},
		{
			desc: "Successful request with empty query",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{Query: ""}.Build(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				reportSnapshot := storage.ReportSnapshot_builder{
					ReportId:              "test-report-id",
					ReportConfigurationId: "test-config-id",
					Name:                  "My View Based Report",
					ReportStatus: storage.ReportStatus_builder{
						ErrorMsg:                 "",
						ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
						RunState:                 storage.ReportStatus_GENERATED,
						ReportRequestType:        storage.ReportStatus_VIEW_BASED,
					}.Build(),
					Requester: userA,
				}.Build()
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
					Return([]*storage.ReportSnapshot{reportSnapshot}, nil).Times(1)
				s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
			},
			isError: false,
		},
		{
			desc: "Successful request with pagination",
			req: apiV2.GetViewBasedReportHistoryRequest_builder{
				ReportParamQuery: apiV2.RawQuery_builder{
					Query: "",
					Pagination: apiV2.Pagination_builder{
						Limit:  5,
						Offset: 10,
					}.Build(),
				}.Build(),
			}.Build(),
			ctx: userContext,
			mockGen: func() {
				s.reportSnapshotDataStore.EXPECT().SearchReportSnapshots(gomock.Any(), gomock.Any()).
					Return([]*storage.ReportSnapshot{}, nil).Times(1)
				s.blobStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{}, nil).AnyTimes()
			},
			isError: false,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			if tc.mockGen != nil {
				tc.mockGen()
			}
			response, err := s.service.(*serviceImpl).GetViewBasedMyReportHistory(tc.ctx, tc.req)
			if tc.isError {
				s.Error(err)
				s.Nil(response)
			} else {
				s.NoError(err)
				s.NotNil(response)
				s.NotNil(response.GetReportSnapshots())
			}
		})
	}
}

func (s *ReportServiceTestSuite) getContextForUser(user *storage.SlimUser) context.Context {
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	mockID.EXPECT().UID().Return(user.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(user.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.GetName()).AnyTimes()

	// Create a mock role with a default access scope for testing
	accessScope := storage.SimpleAccessScope_builder{
		Rules: storage.SimpleAccessScope_Rules_builder{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				storage.SimpleAccessScope_Rules_Namespace_builder{ClusterName: "cluster-2", NamespaceName: "namespace-2"}.Build(),
			},
		}.Build(),
	}.Build()
	mockRole := permissionsMocks.NewMockResolvedRole(s.mockCtrl)
	mockRole.EXPECT().GetAccessScope().Return(accessScope).AnyTimes()
	mockID.EXPECT().Roles().Return([]permissions.ResolvedRole{mockRole}).AnyTimes()

	return authn.ContextWithIdentity(s.ctx, mockID, s.T())
}
