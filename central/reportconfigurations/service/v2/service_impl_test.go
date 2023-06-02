package v2

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/reportconfigurations/datastore/mocks"
	managerMocks "github.com/stackrox/rox/central/reports/manager/mocks"
	collectionMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestReportConfigurationServiceV2(t *testing.T) {
	suite.Run(t, new(ReportConfigurationServiceTestSuite))
}

type ReportConfigurationServiceTestSuite struct {
	suite.Suite
	service               Service
	reportConfigDatastore *mocks.MockDataStore
	notifierDatastore     *notifierMocks.MockDataStore
	collectionDatastore   *collectionMocks.MockDataStore
	manager               *managerMocks.MockManager
	mockCtrl              *gomock.Controller
}

type upsertTestCase struct {
	desc                       string
	setMocksAndGenReportConfig func() *apiV2.ReportConfiguration
	reportConfigGen            func() *storage.ReportConfiguration
	isValidationError          bool
}

func (s *ReportConfigurationServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		s.T().Skip("Skip test when reporting enhancements are disabled")
		s.T().SkipNow()
	}
	s.reportConfigDatastore = mocks.NewMockDataStore(s.mockCtrl)
	s.notifierDatastore = notifierMocks.NewMockDataStore(s.mockCtrl)
	s.collectionDatastore = collectionMocks.NewMockDataStore(s.mockCtrl)
	s.manager = managerMocks.NewMockManager(s.mockCtrl)
	s.service = New(s.reportConfigDatastore, s.notifierDatastore, s.collectionDatastore, s.manager)
}

func (s *ReportConfigurationServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ReportConfigurationServiceTestSuite) TestCreateReportConfiguration() {
	ctx := context.Background()

	for _, tc := range s.upsertReportConfigTestCases(false) {
		s.T().Run(tc.desc, func(t *testing.T) {
			requestConfig := tc.setMocksAndGenReportConfig()
			if !tc.isValidationError {
				protoReportConfig := tc.reportConfigGen()
				s.reportConfigDatastore.EXPECT().AddReportConfiguration(ctx, protoReportConfig).Return(protoReportConfig.GetId(), nil).Times(1)
				s.reportConfigDatastore.EXPECT().GetReportConfiguration(ctx, protoReportConfig.GetId()).Return(protoReportConfig, true, nil).Times(1)
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
}

func (s *ReportConfigurationServiceTestSuite) TestUpdateReportConfiguration() {
	ctx := context.Background()

	for _, tc := range s.upsertReportConfigTestCases(true) {
		s.T().Run(tc.desc, func(t *testing.T) {
			requestConfig := tc.setMocksAndGenReportConfig()
			if !tc.isValidationError {
				protoReportConfig := tc.reportConfigGen()
				s.reportConfigDatastore.EXPECT().UpdateReportConfiguration(ctx, protoReportConfig).Return(nil).Times(1)
			}
			result, err := s.service.UpdateReportConfiguration(ctx, requestConfig)
			if tc.isValidationError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(&apiV2.Empty{}, result)
			}
		})
	}
}

func (s *ReportConfigurationServiceTestSuite) TestGetReportConfigurations() {
	ctx := context.Background()
	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *v1.Query
	}{
		{
			desc:      "Empty query",
			query:     &apiV2.RawQuery{Query: ""},
			expectedQ: search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
		},
		{
			desc:  "Query with search field",
			query: &apiV2.RawQuery{Query: "Report Name:name"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ReportName, "name").
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 25},
			},
			expectedQ: search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(25)).ProtoQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.GetReportConfigurationsResponse{
				ReportConfigs: []*apiV2.ReportConfiguration{fixtures.GetValidV2ReportConfigWithMultipleNotifiers()},
			}

			s.reportConfigDatastore.EXPECT().GetReportConfigurations(ctx, tc.expectedQ).
				Return([]*storage.ReportConfiguration{fixtures.GetValidReportConfigWithMultipleNotifiers()}, nil).Times(1)

			s.mockGetNotifierCall(expectedResp.ReportConfigs[0].GetNotifiers()[0], 1, true, "")
			s.mockGetNotifierCall(expectedResp.ReportConfigs[0].GetNotifiers()[1], 1, true, "")
			s.mockGetCollectionCall(expectedResp.ReportConfigs[0], 1, true, "")

			configs, err := s.service.GetReportConfigurations(ctx, tc.query)
			s.NoError(err)
			s.Equal(expectedResp, configs)
		})
	}
}

func (s *ReportConfigurationServiceTestSuite) TestGetReportConfigurationByID() {
	ctx := context.Background()
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
					s.reportConfigDatastore.EXPECT().GetReportConfiguration(ctx, tc.id).
						Return(fixtures.GetValidReportConfigWithMultipleNotifiers(), true, nil).Times(1)

					expectedResp = fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
					s.mockGetNotifierCall(expectedResp.GetNotifiers()[0], 1, true, "")
					s.mockGetNotifierCall(expectedResp.GetNotifiers()[1], 1, true, "")
					s.mockGetCollectionCall(expectedResp, 1, true, "")
				} else {
					s.reportConfigDatastore.EXPECT().GetReportConfiguration(ctx, tc.id).
						Return(nil, false, nil)
				}
			}

			config, err := s.service.GetReportConfiguration(ctx, &apiV2.ResourceByID{Id: tc.id})
			if tc.isValidationError || tc.isDataNotFoundError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(expectedResp, config)
			}
		})
	}
}

func (s *ReportConfigurationServiceTestSuite) TestCountReportConfigurations() {
	ctx := context.Background()
	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *v1.Query
	}{
		{
			desc:      "Empty query",
			query:     &apiV2.RawQuery{Query: ""},
			expectedQ: search.NewQueryBuilder().ProtoQuery(),
		},
		{
			desc:      "Query with search field",
			query:     &apiV2.RawQuery{Query: "Report Name:name"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ReportName, "name").ProtoQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			s.reportConfigDatastore.EXPECT().Count(ctx, tc.expectedQ).Return(1, nil).Times(1)
			_, err := s.service.CountReportConfigurations(ctx, tc.query)
			s.NoError(err)
		})
	}
}

func (s *ReportConfigurationServiceTestSuite) TestDeleteReportConfiguration() {
	ctx := context.Background()
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
			s.reportConfigDatastore.EXPECT().RemoveReportConfiguration(ctx, tc.id).Return(nil).Times(1)
		}
		_, err := s.service.DeleteReportConfiguration(ctx, &apiV2.ResourceByID{Id: tc.id})
		if tc.isError {
			s.Error(err)
		} else {
			s.NoError(err)
		}
	}
}

func (s *ReportConfigurationServiceTestSuite) upsertReportConfigTestCases(isUpdate bool) []upsertTestCase {
	cases := []upsertTestCase{
		{
			desc: "Valid report config with multiple notifiers",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				times := 1
				if !isUpdate {
					times = 2
				}
				s.mockGetNotifierCall(ret.GetNotifiers()[0], times, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], times, true, "")

				s.mockGetCollectionCall(ret, times, true, "")

				return ret
			},
			reportConfigGen: func() *storage.ReportConfiguration {
				return fixtures.GetValidReportConfigWithMultipleNotifiers()
			},
			isValidationError: false,
		},
		{
			desc: "Valid report config without notifiers",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				times := 1
				if !isUpdate {
					times = 2
				}
				s.mockGetCollectionCall(ret, times, true, "")
				return ret
			},
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
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
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : empty notifierID in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.NotifierId = ""
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : empty mailing list in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.MailingLists = nil
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : invalid email in email config",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.MailingLists = []string{"sdfdksfjk"}
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier : notifier not found",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, false, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid notifier: incorrect name",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, true, "different-name")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with missing resource scope",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, true, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid resource scope : empty collectionID",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference.(*apiV2.ResourceScope_CollectionScope).CollectionScope.CollectionId = ""
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, true, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid resource scope : collection not found",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, true, "")

				s.mockGetCollectionCall(ret, 1, false, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid resource scope : incorrect name",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, true, "")

				s.mockGetCollectionCall(ret, 1, true, "different-name")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with missing vuln report filters",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters = nil
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, true, "")

				s.mockGetCollectionCall(ret, 1, true, "")
				return ret
			},
			isValidationError: true,
		},
		{
			desc: "Report config with invalid vuln report filters : cvesSince unset",
			setMocksAndGenReportConfig: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters.CvesSince = nil
				s.mockGetNotifierCall(ret.GetNotifiers()[0], 1, true, "")
				s.mockGetNotifierCall(ret.GetNotifiers()[1], 1, true, "")

				s.mockGetCollectionCall(ret, 1, true, "")
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

func (s *ReportConfigurationServiceTestSuite) mockGetNotifierCall(reqNotifier *apiV2.NotifierConfiguration, times int,
	notifierIDExits bool, nameOverride string) {
	name := reqNotifier.GetNotifierName()
	if nameOverride != "" {
		name = nameOverride
	}
	if notifierIDExits {
		s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), reqNotifier.GetEmailConfig().GetNotifierId()).
			Return(&storage.Notifier{
				Id:   reqNotifier.GetEmailConfig().GetNotifierId(),
				Name: name,
			}, true, nil).Times(times)
	} else {
		s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), reqNotifier.GetEmailConfig().GetNotifierId()).
			Return(nil, false, nil).Times(times)
	}
}

func (s *ReportConfigurationServiceTestSuite) mockGetCollectionCall(reqConfig *apiV2.ReportConfiguration, times int,
	collectionIDExists bool, nameOverride string) {
	name := reqConfig.GetResourceScope().GetCollectionScope().GetCollectionName()
	if nameOverride != "" {
		name = nameOverride
	}
	if collectionIDExists {
		s.collectionDatastore.EXPECT().Get(gomock.Any(), reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId()).
			Return(&storage.ResourceCollection{
				Id:   reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId(),
				Name: name,
			}, true, nil).Times(times)
	} else {
		s.collectionDatastore.EXPECT().Get(gomock.Any(), reqConfig.GetResourceScope().GetCollectionScope().GetCollectionId()).
			Return(nil, false, nil).Times(times)
	}
}
