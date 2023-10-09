package v2

import (
	"testing"

	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestTypeConversions(t *testing.T) {
	suite.Run(t, new(typeConversionTestSuite))
}

type typeConversionTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	collectionDatastore *collectionDSMocks.MockDataStore
	notifierDatastore   *notifierDSMocks.MockDataStore
	service             *serviceImpl
}

func (s *typeConversionTestSuite) SetupSuite() {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")
	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip test when reporting enhancements are disabled")
		s.T().SkipNow()
	}

	s.mockCtrl = gomock.NewController(s.T())
	s.collectionDatastore = collectionDSMocks.NewMockDataStore(s.mockCtrl)
	s.notifierDatastore = notifierDSMocks.NewMockDataStore(s.mockCtrl)
	s.service = &serviceImpl{
		collectionDatastore: s.collectionDatastore,
		notifierDatastore:   s.notifierDatastore,
	}
}

func (s *typeConversionTestSuite) TearDownSuite() {
	s.mockCtrl.Finish()
}

func (s *typeConversionTestSuite) TestConvertV2ReportConfigurationToProto() {
	creator := &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
	accessScopeRules := []*storage.SimpleAccessScope_Rules{
		{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{ClusterName: "cluster-2", NamespaceName: "namespace-2"},
			},
		},
	}

	var cases = []struct {
		testname        string
		reportConfigGen func() *apiV2.ReportConfiguration
		resultGen       func() *storage.ReportConfiguration
	}{
		{
			testname: "Report config with notifiers",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				return fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Creator = creator
				ret.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				return ret
			},
		},
		{
			testname: "Report config without notifiers",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Notifiers = nil
				ret.Creator = creator
				ret.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				return ret
			},
		},
		{
			testname: "Report config with custom email subject and body",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[0].GetEmailConfig().CustomSubject = "custom subject"
				ret.Notifiers[0].GetEmailConfig().CustomBody = "custom body"
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Notifiers[0].GetEmailConfig().CustomSubject = "custom subject"
				ret.Notifiers[0].GetEmailConfig().CustomBody = "custom body"
				ret.Creator = creator
				ret.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				return ret
			},
		},
		{
			testname: "Report config without schedule",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Schedule = nil
				ret.Creator = creator
				ret.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				return ret
			},
		},
		{
			testname: "Report config without filter",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Filter = nil
				ret.Creator = creator
				return ret
			},
		},
		{
			testname: "Report config without resource scope",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ResourceScope = nil
				ret.Creator = creator
				ret.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				return ret
			},
		},
		{
			testname: "Report config without CvesSince in filter",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().CvesSince = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetVulnReportFilters().CvesSince = nil
				ret.Creator = creator
				ret.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				return ret
			},
		},
		{
			testname: "Report config without scope reference in ResourceScope",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference = nil
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ResourceScope.ScopeReference = nil
				ret.Creator = creator
				ret.GetVulnReportFilters().AccessScopeRules = accessScopeRules
				return ret
			},
		},
	}

	for _, c := range cases {
		s.T().Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()
			expected := c.resultGen()
			converted := s.service.convertV2ReportConfigurationToProto(reportConfig, creator, accessScopeRules)
			assert.Equal(t, expected, converted)
		})
	}
}

func setAllNotifierNamesToFixedValue(reportConfig *apiV2.ReportConfiguration, name string) {
	for _, notifierConfig := range reportConfig.GetNotifiers() {
		notifierConfig.NotifierName = name
	}
}

func setCollectionName(reportConfig *apiV2.ReportConfiguration, name string) {
	if reportConfig.ResourceScope != nil && reportConfig.ResourceScope.GetCollectionScope() != nil {
		reportConfig.ResourceScope.GetCollectionScope().CollectionName = name
	}
}

func (s *typeConversionTestSuite) TestConvertProtoReportConfigurationToV2() {
	mockNotifierName := "mock-notifier"
	mockCollectionName := "mock-collection"

	var cases = []struct {
		testname        string
		reportConfigGen func() *storage.ReportConfiguration
		resultGen       func() *apiV2.ReportConfiguration
	}{
		{
			testname: "Report config with notifiers",
			reportConfigGen: func() *storage.ReportConfiguration {
				return fixtures.GetValidReportConfigWithMultipleNotifiersV2()
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without notifiers",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Notifiers = nil
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config with custom email subject and body",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Notifiers[0].GetEmailConfig().CustomSubject = "custom subject"
				ret.Notifiers[0].GetEmailConfig().CustomBody = "custom body"
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Notifiers[0].GetEmailConfig().CustomSubject = "custom subject"
				ret.Notifiers[0].GetEmailConfig().CustomBody = "custom body"
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without schedule",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Schedule = nil
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Schedule = nil
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without filter",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.Filter = nil
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.Filter = nil
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without resource scope",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ResourceScope = nil
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				return ret
			},
		},
		{
			testname: "Report config without CvesSince in filter",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetVulnReportFilters().CvesSince = nil
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().CvesSince = nil
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without scope reference in ResourceScope",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ResourceScope.ScopeReference = nil
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference = nil
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				return ret
			},
		},
	}

	for _, c := range cases {
		s.T().Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()

			for _, notifierConfig := range reportConfig.GetNotifiers() {
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), notifierConfig.GetId()).
					Return(&storage.Notifier{
						Id:   notifierConfig.GetId(),
						Name: mockNotifierName,
					}, true, nil).Times(1)
			}
			if reportConfig.GetResourceScope() != nil && reportConfig.GetResourceScope().GetScopeReference() != nil {
				s.collectionDatastore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(&storage.ResourceCollection{
						Id:   reportConfig.GetResourceScope().GetCollectionId(),
						Name: mockCollectionName,
					}, true, nil).Times(1)
			}

			expected := c.resultGen()
			converted, err := s.service.convertProtoReportConfigurationToV2(reportConfig)
			assert.NoError(t, err)
			assert.Equal(t, expected, converted)
		})
	}
}

func (s *typeConversionTestSuite) TestConvertProtoScheduleToV2() {
	var cases = []struct {
		testname string
		schedule *storage.Schedule
		result   *apiV2.ReportSchedule
	}{
		{
			testname: "Schedule with Daily interval",
			schedule: newSchedule(12, 12, []int32{}, false, []int32{}),
			result:   newScheduleV2(12, 12, []int32{}, []int32{}),
		},
		{
			testname: "Schedule with Weekly interval",
			schedule: newSchedule(34, 12, []int32{2}, false, []int32{}),
			result:   newScheduleV2(34, 12, []int32{2}, []int32{}),
		},
		{
			testname: "Schedule with Weekly interval, oneOf interval is of type WeeklyInterval which allows just one day of week to be set",
			schedule: newSchedule(34, 12, []int32{2}, true, []int32{}),
			result: func() *apiV2.ReportSchedule {
				sched := newScheduleV2(34, 12, []int32{}, []int32{})
				sched.IntervalType = apiV2.ReportSchedule_WEEKLY
				return sched
			}(),
		},
		{
			testname: "Schedule with Weekly interval, Multiple days",
			schedule: newSchedule(34, 12, []int32{2, 4}, false, []int32{}),
			result:   newScheduleV2(34, 12, []int32{2, 4}, []int32{}),
		},
		{
			testname: "Schedule with monthly interval",
			schedule: newSchedule(34, 12, []int32{}, false, []int32{1}),
			result:   newScheduleV2(34, 12, []int32{}, []int32{1}),
		},
	}

	for _, c := range cases {
		s.T().Run(c.testname, func(t *testing.T) {
			converted := s.service.convertProtoScheduleToV2(c.schedule)
			assert.Equal(t, c.result, converted)
		})
	}
}

func (s *typeConversionTestSuite) TestConvertV2ScheduleToProto() {
	var cases = []struct {
		testname string
		schedule *apiV2.ReportSchedule
		result   *storage.Schedule
	}{
		{
			testname: "Report Schedule with Weekly interval",
			schedule: newScheduleV2(34, 12, []int32{2}, []int32{}),
			result:   newSchedule(34, 12, []int32{2}, false, []int32{}),
		},
		{
			testname: "Report Schedule with Weekly interval, Multiple days",
			schedule: newScheduleV2(34, 12, []int32{2, 4}, []int32{}),
			result:   newSchedule(34, 12, []int32{2, 4}, false, []int32{}),
		},
		{
			testname: "Report Schedule with Monthly interval",
			schedule: newScheduleV2(34, 12, []int32{}, []int32{1}),
			result:   newSchedule(34, 12, []int32{}, false, []int32{1}),
		},
	}

	for _, c := range cases {
		s.T().Run(c.testname, func(t *testing.T) {
			converted := s.service.convertV2ScheduleToProto(c.schedule)
			assert.Equal(t, c.result, converted)
		})
	}
}

func newSchedule(minute int32, hour int32, weekdays []int32, isWeeklyIntervalType bool, daysOfMonth []int32) *storage.Schedule {
	var sched storage.Schedule

	sched.Hour = hour
	sched.Minute = minute
	if len(daysOfMonth) != 0 {
		sched.IntervalType = storage.Schedule_MONTHLY
		sched.Interval = &storage.Schedule_DaysOfMonth_{
			DaysOfMonth: &storage.Schedule_DaysOfMonth{
				Days: daysOfMonth,
			},
		}
		return &sched
	}
	if len(weekdays) == 0 {
		sched.IntervalType = storage.Schedule_DAILY
	} else {
		sched.IntervalType = storage.Schedule_WEEKLY
		if isWeeklyIntervalType {
			sched.Interval = &storage.Schedule_Weekly{
				Weekly: &storage.Schedule_WeeklyInterval{
					Day: weekdays[0],
				},
			}
		} else {
			sched.Interval = &storage.Schedule_DaysOfWeek_{
				DaysOfWeek: &storage.Schedule_DaysOfWeek{
					Days: weekdays,
				},
			}
		}
	}
	return &sched
}

func newScheduleV2(minute int32, hour int32, weekdays []int32, daysOfMonth []int32) *apiV2.ReportSchedule {
	var sched apiV2.ReportSchedule

	sched.Hour = hour
	sched.Minute = minute
	if len(daysOfMonth) != 0 {
		sched.IntervalType = apiV2.ReportSchedule_MONTHLY
		sched.Interval = &apiV2.ReportSchedule_DaysOfMonth_{
			DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{
				Days: daysOfMonth,
			},
		}
		return &sched
	}
	if len(weekdays) == 0 {
		sched.IntervalType = apiV2.ReportSchedule_UNSET
	} else {
		sched.IntervalType = apiV2.ReportSchedule_WEEKLY
		sched.Interval = &apiV2.ReportSchedule_DaysOfWeek_{
			DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{
				Days: weekdays,
			},
		}
	}
	return &sched
}
