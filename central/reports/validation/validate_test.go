package validation

import (
	"testing"

	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stretchr/testify/assert"
)

func TestValidateSchedule(t *testing.T) {
	v := &Validator{}

	var cases = []struct {
		testname    string
		config      *apiV2.ReportConfiguration
		expectError bool
		errContains string
	}{
		{
			testname: "Nil schedule is valid",
			config: &apiV2.ReportConfiguration{
				Schedule: nil,
			},
			expectError: false,
		},
		{
			testname: "UNSET interval type is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_UNSET,
				},
			},
			expectError: true,
			errContains: "DAILY, WEEKLY, or MONTHLY",
		},
		{
			testname: "Valid daily schedule",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_DAILY,
					Hour:         14,
					Minute:       30,
				},
			},
			expectError: false,
		},
		{
			testname: "Daily schedule with days_of_week is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_DAILY,
					Hour:         14,
					Minute:       30,
					Interval: &apiV2.ReportSchedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{Days: []int32{1}},
					},
				},
			},
			expectError: true,
			errContains: "Daily schedule must not specify",
		},
		{
			testname: "Daily schedule with days_of_month is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_DAILY,
					Hour:         14,
					Minute:       30,
					Interval: &apiV2.ReportSchedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{Days: []int32{1}},
					},
				},
			},
			expectError: true,
			errContains: "Daily schedule must not specify",
		},
		{
			testname: "Valid weekly schedule",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					Hour:         10,
					Minute:       0,
					Interval: &apiV2.ReportSchedule_DaysOfWeek_{
						DaysOfWeek: &apiV2.ReportSchedule_DaysOfWeek{Days: []int32{1, 3}},
					},
				},
			},
			expectError: false,
		},
		{
			testname: "Weekly schedule without days is invalid",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_WEEKLY,
					Hour:         10,
					Minute:       0,
				},
			},
			expectError: true,
			errContains: "days of week",
		},
		{
			testname: "Valid monthly schedule",
			config: &apiV2.ReportConfiguration{
				Schedule: &apiV2.ReportSchedule{
					IntervalType: apiV2.ReportSchedule_MONTHLY,
					Hour:         8,
					Minute:       45,
					Interval: &apiV2.ReportSchedule_DaysOfMonth_{
						DaysOfMonth: &apiV2.ReportSchedule_DaysOfMonth{Days: []int32{1}},
					},
				},
			},
			expectError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			err := v.validateSchedule(c.config)
			if c.expectError {
				assert.Error(t, err)
				if c.errContains != "" {
					assert.Contains(t, err.Error(), c.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
