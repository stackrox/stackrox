package v2

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/reportconfigurations/datastore/mocks"
	managerMocks "github.com/stackrox/rox/central/reports/manager/mocks"
	collectionMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
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
	desc              string
	v2ReprtConfigGen  func() *apiV2.ReportConfiguration
	reportConfigGen   func() *storage.ReportConfiguration
	setMocks          func()
	isValidationError bool
}

var noMocks = func() {}

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

func (s *ReportConfigurationServiceTestSuite) TestAddConfiguration() {
	ctx := context.Background()

	for _, tc := range s.upsertReportConfigTestCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			requestConfig := tc.v2ReprtConfigGen()
			tc.setMocks()
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

func (s *ReportConfigurationServiceTestSuite) upsertReportConfigTestCases() []upsertTestCase {
	cases := []upsertTestCase{
		{
			desc: "Valid report config with multiple notifiers",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				return fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
			},
			reportConfigGen: func() *storage.ReportConfiguration {
				return fixtures.GetValidReportConfigWithMultipleNotifiers()
			},
			isValidationError: false,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)
				s.collectionDatastore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
		{
			desc: "Valid report config without notifiers",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				return ret
			},
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				return ret
			},
			isValidationError: false,
			setMocks: func() {
				s.collectionDatastore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
		{
			desc: "Report config with invalid schedule : invalid day of week",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.Schedule{
					IntervalType: apiV2.Schedule_WEEKLY,
					Interval: &apiV2.Schedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.Schedule_DaysOfWeek{
							Days: []int32{8},
						},
					},
				}
				return ret
			},
			isValidationError: true,
			setMocks:          noMocks,
		},
		{
			desc: "Report config with invalid schedule : missing days of week",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.Schedule{
					IntervalType: apiV2.Schedule_WEEKLY,
					Interval: &apiV2.Schedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.Schedule_DaysOfWeek{
							Days: []int32{},
						},
					},
				}
				return ret
			},
			isValidationError: true,
			setMocks:          noMocks,
		},
		{
			desc: "Report config with invalid schedule : invalid day of month",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.Schedule{
					IntervalType: apiV2.Schedule_MONTHLY,
					Interval: &apiV2.Schedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.Schedule_DaysOfMonth{
							Days: []int32{30},
						},
					},
				}
				return ret
			},
			isValidationError: true,
			setMocks:          noMocks,
		},
		{
			desc: "Report config with invalid schedule : missing days of month",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = &apiV2.Schedule{
					IntervalType: apiV2.Schedule_MONTHLY,
					Interval: &apiV2.Schedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.Schedule_DaysOfMonth{
							Days: nil,
						},
					},
				}
				return ret
			},
			isValidationError: true,
			setMocks:          noMocks,
		},
		{
			desc: "Report config with invalid notifier : missing email config",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig = nil
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
		{
			desc: "Report config with invalid notifier : empty notifierID in email config",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.NotifierId = ""
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
		{
			desc: "Report config with invalid notifier : empty mailing list in email config",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.MailingLists = nil
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
		{
			desc: "Report config with invalid notifier : invalid email in email config",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[1].NotifierConfig.(*apiV2.NotifierConfiguration_EmailConfig).EmailConfig.MailingLists = []string{"sdfdksfjk"}
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
		{
			desc: "Report config with invalid notifier : notifier not found",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				return fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).
					Times(1).Return(nil, true, nil).
					Times(1).Return(nil, false, nil)
			},
		},
		{
			desc: "Report config with missing resource scope",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)
			},
		},
		{
			desc: "Report config with invalid resource scope : empty collectionID",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference.(*apiV2.ResourceScope_CollectionId).CollectionId = ""
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)
			},
		},
		{
			desc: "Report config with invalid resource scope : collection not found",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				return fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)
				s.collectionDatastore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)
			},
		},
		{
			desc: "Report config with missing vuln report filters",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters = nil
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)
				s.collectionDatastore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
		{
			desc: "Report config with invalid vuln report filters : cvesSince unset",
			v2ReprtConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter.(*apiV2.ReportConfiguration_VulnReportFilters).VulnReportFilters.CvesSince = nil
				return ret
			},
			isValidationError: true,
			setMocks: func() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(2)
				s.collectionDatastore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)
			},
		},
	}

	return cases
}
