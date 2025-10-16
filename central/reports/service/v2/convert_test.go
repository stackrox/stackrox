package v2

import (
	"testing"

	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	collectionDSMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
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
	creator := &storage.SlimUser{}
	creator.SetId("uid")
	creator.SetName("name")
	accessScopeRules := []*storage.SimpleAccessScope_Rules{
		storage.SimpleAccessScope_Rules_builder{
			IncludedClusters: []string{"cluster-1"},
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				storage.SimpleAccessScope_Rules_Namespace_builder{ClusterName: "cluster-2", NamespaceName: "namespace-2"}.Build(),
			},
		}.Build(),
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
				ret.SetCreator(creator)
				ret.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
				return ret
			},
		},
		{
			testname: "Report config without notifiers",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.SetNotifiers(nil)
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.SetNotifiers(nil)
				ret.SetCreator(creator)
				ret.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
				return ret
			},
		},
		{
			testname: "Report config with custom email subject and body",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomSubject("custom subject")
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomBody("custom body")
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomSubject("custom subject")
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomBody("custom body")
				ret.SetCreator(creator)
				ret.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
				return ret
			},
		},
		{
			testname: "Report config without schedule",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ClearSchedule()
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ClearSchedule()
				ret.SetCreator(creator)
				ret.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
				return ret
			},
		},
		{
			testname: "Report config without filter",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ClearFilter()
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ClearFilter()
				ret.SetCreator(creator)
				return ret
			},
		},
		{
			testname: "Report config without resource scope",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ClearResourceScope()
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ClearResourceScope()
				ret.SetCreator(creator)
				ret.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
				return ret
			},
		},
		{
			testname: "Report config without CvesSince in filter",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().ClearCvesSince()
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetVulnReportFilters().ClearCvesSince()
				ret.SetCreator(creator)
				ret.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
				return ret
			},
		},
		{
			testname: "Report config without scope reference in ResourceScope",
			reportConfigGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetResourceScope().ClearScopeReference()
				return ret
			},
			resultGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetResourceScope().ClearScopeReference()
				ret.SetCreator(creator)
				ret.GetVulnReportFilters().SetAccessScopeRules(accessScopeRules)
				return ret
			},
		},
	}

	for _, c := range cases {
		s.T().Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()
			expected := c.resultGen()
			converted := s.service.convertV2ReportConfigurationToProto(reportConfig, creator, accessScopeRules)
			protoassert.Equal(t, expected, converted)
		})
	}
}

func setAllNotifierNamesToFixedValue(reportConfig *apiV2.ReportConfiguration, name string) {
	for _, notifierConfig := range reportConfig.GetNotifiers() {
		notifierConfig.SetNotifierName(name)
	}
}

func setCollectionName(reportConfig *apiV2.ReportConfiguration, name string) {
	if reportConfig.GetResourceScope() != nil && reportConfig.GetResourceScope().GetCollectionScope() != nil {
		reportConfig.GetResourceScope().GetCollectionScope().SetCollectionName(name)
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
				ret.SetNotifiers(nil)
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.SetNotifiers(nil)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config with custom email subject and body",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomSubject("custom subject")
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomBody("custom body")
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomSubject("custom subject")
				ret.GetNotifiers()[0].GetEmailConfig().SetCustomBody("custom body")
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without schedule",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ClearSchedule()
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ClearSchedule()
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without filter",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ClearFilter()
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ClearFilter()
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without resource scope",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.ClearResourceScope()
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.ClearResourceScope()
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				return ret
			},
		},
		{
			testname: "Report config without CvesSince in filter",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetVulnReportFilters().ClearCvesSince()
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().ClearCvesSince()
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				setCollectionName(ret, mockCollectionName)
				return ret
			},
		},
		{
			testname: "Report config without scope reference in ResourceScope",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiersV2()
				ret.GetResourceScope().ClearScopeReference()
				return ret
			},
			resultGen: func() *apiV2.ReportConfiguration {
				ret := fixtures.GetValidV2ReportConfigWithMultipleNotifiers()
				ret.GetResourceScope().ClearScopeReference()
				setAllNotifierNamesToFixedValue(ret, mockNotifierName)
				return ret
			},
		},
	}

	for _, c := range cases {
		s.T().Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()

			for _, notifierConfig := range reportConfig.GetNotifiers() {
				notifier := &storage.Notifier{}
				notifier.SetId(notifierConfig.GetId())
				notifier.SetName(mockNotifierName)
				s.notifierDatastore.EXPECT().GetNotifier(gomock.Any(), notifierConfig.GetId()).
					Return(notifier, true, nil).Times(1)
			}
			if reportConfig.GetResourceScope() != nil && reportConfig.GetResourceScope().GetScopeReference() != nil {
				rc := &storage.ResourceCollection{}
				rc.SetId(reportConfig.GetResourceScope().GetCollectionId())
				rc.SetName(mockCollectionName)
				s.collectionDatastore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(rc, true, nil).Times(1)
			}

			expected := c.resultGen()
			converted, err := s.service.convertProtoReportConfigurationToV2(reportConfig)
			assert.NoError(t, err)
			protoassert.Equal(t, expected, converted)
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
				sched.SetIntervalType(apiV2.ReportSchedule_WEEKLY)
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
			protoassert.Equal(t, c.result, converted)
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
			protoassert.Equal(t, c.result, converted)
		})
	}
}

func newSchedule(minute int32, hour int32, weekdays []int32, isWeeklyIntervalType bool, daysOfMonth []int32) *storage.Schedule {
	var sched storage.Schedule

	sched.SetHour(hour)
	sched.SetMinute(minute)
	if len(daysOfMonth) != 0 {
		sched.SetIntervalType(storage.Schedule_MONTHLY)
		sd := &storage.Schedule_DaysOfMonth{}
		sd.SetDays(daysOfMonth)
		sched.SetDaysOfMonth(proto.ValueOrDefault(sd))
		return &sched
	}
	if len(weekdays) == 0 {
		sched.SetIntervalType(storage.Schedule_DAILY)
	} else {
		sched.SetIntervalType(storage.Schedule_WEEKLY)
		if isWeeklyIntervalType {
			sw := &storage.Schedule_WeeklyInterval{}
			sw.SetDay(weekdays[0])
			sched.SetWeekly(proto.ValueOrDefault(sw))
		} else {
			sd := &storage.Schedule_DaysOfWeek{}
			sd.SetDays(weekdays)
			sched.SetDaysOfWeek(proto.ValueOrDefault(sd))
		}
	}
	return &sched
}

func newScheduleV2(minute int32, hour int32, weekdays []int32, daysOfMonth []int32) *apiV2.ReportSchedule {
	var sched apiV2.ReportSchedule

	sched.SetHour(hour)
	sched.SetMinute(minute)
	if len(daysOfMonth) != 0 {
		sched.SetIntervalType(apiV2.ReportSchedule_MONTHLY)
		rd := &apiV2.ReportSchedule_DaysOfMonth{}
		rd.SetDays(daysOfMonth)
		sched.SetDaysOfMonth(proto.ValueOrDefault(rd))
		return &sched
	}
	if len(weekdays) == 0 {
		sched.SetIntervalType(apiV2.ReportSchedule_UNSET)
	} else {
		sched.SetIntervalType(apiV2.ReportSchedule_WEEKLY)
		rd := &apiV2.ReportSchedule_DaysOfWeek{}
		rd.SetDays(weekdays)
		sched.SetDaysOfWeek(proto.ValueOrDefault(rd))
	}
	return &sched
}
