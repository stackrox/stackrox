package v2

import (
	"testing"

	notifierMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	collectionMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestConvertV2ReportConfigurationToProto(t *testing.T) {
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
				return fixtures.GetValidReportConfigWithMultipleNotifiers()
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Notifiers = nil
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Schedule = nil
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.Filter = nil
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.ResourceScope = nil
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.GetVulnReportFilters().CvesSince = nil
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
				ret.ResourceScope.ScopeReference = nil
				return ret
			},
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()
			expected := c.resultGen()
			converted := convertV2ReportConfigurationToProto(reportConfig)
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

func TestConvertProtoReportConfigurationToV2(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	notifierDatastore := notifierMocks.NewMockDataStore(mockCtrl)
	collectionDatastore := collectionMocks.NewMockDataStore(mockCtrl)
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
				return fixtures.GetValidReportConfigWithMultipleNotifiers()
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
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
			testname: "Report config without schedule",
			reportConfigGen: func() *storage.ReportConfiguration {
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
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
				ret := fixtures.GetValidReportConfigWithMultipleNotifiers()
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
		t.Run(c.testname, func(t *testing.T) {
			reportConfig := c.reportConfigGen()

			for _, notifierConfig := range reportConfig.GetNotifiers() {
				notifierDatastore.EXPECT().GetNotifier(gomock.Any(), notifierConfig.GetEmailConfig().GetNotifierId()).
					Return(&storage.Notifier{
						Id:   notifierConfig.GetEmailConfig().GetNotifierId(),
						Name: mockNotifierName,
					}, true, nil).Times(1)
			}
			if reportConfig.GetResourceScope() != nil && reportConfig.GetResourceScope().GetScopeReference() != nil {
				collectionDatastore.EXPECT().Get(gomock.Any(), reportConfig.GetResourceScope().GetCollectionId()).
					Return(&storage.ResourceCollection{
						Id:   reportConfig.GetResourceScope().GetCollectionId(),
						Name: mockCollectionName,
					}, true, nil).Times(1)
			}

			expected := c.resultGen()
			converted := convertProtoReportConfigurationToV2(reportConfig, collectionDatastore, notifierDatastore)
			assert.Equal(t, expected, converted)
		})
	}
}

func TestConvertProtoScheduleToV2(t *testing.T) {
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
			result:   newScheduleV2(34, 12, []int32{3}, []int32{}),
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
			result:   newScheduleV2(34, 12, []int32{3, 5}, []int32{}),
		},
		{
			testname: "Schedule with monthly interval",
			schedule: newSchedule(34, 12, []int32{}, false, []int32{1}),
			result:   newScheduleV2(34, 12, []int32{}, []int32{1}),
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			converted := ConvertProtoScheduleToV2(c.schedule)
			assert.Equal(t, c.result, converted)
		})
	}
}

func TestConvertV2ScheduleToProto(t *testing.T) {
	var cases = []struct {
		testname string
		schedule *apiV2.ReportSchedule
		result   *storage.Schedule
	}{
		{
			testname: "Report Schedule with Weekly interval",
			schedule: newScheduleV2(34, 12, []int32{2}, []int32{}),
			result:   newSchedule(34, 12, []int32{1}, false, []int32{}),
		},
		{
			testname: "Report Schedule with Weekly interval, Multiple days",
			schedule: newScheduleV2(34, 12, []int32{2, 4}, []int32{}),
			result:   newSchedule(34, 12, []int32{1, 3}, false, []int32{}),
		},
		{
			testname: "Report Schedule with Monthly interval",
			schedule: newScheduleV2(34, 12, []int32{}, []int32{1}),
			result:   newSchedule(34, 12, []int32{}, false, []int32{1}),
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			converted := convertV2ScheduleToProto(c.schedule)
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
