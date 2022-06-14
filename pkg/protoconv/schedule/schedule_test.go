package schedule

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSchedule(minute int32, hour int32, weekdays []int32, daysOfMonth []int32) *storage.Schedule {
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
	} else {
		if len(weekdays) == 0 {
			sched.IntervalType = storage.Schedule_DAILY
		} else {
			if len(weekdays) == 1 {
				sched.IntervalType = storage.Schedule_WEEKLY
				sched.Interval = &storage.Schedule_Weekly{
					Weekly: &storage.Schedule_WeeklyInterval{
						Day: weekdays[0],
					},
				}
			} else {
				sched.IntervalType = storage.Schedule_WEEKLY
				sched.Interval = &storage.Schedule_DaysOfWeek_{
					DaysOfWeek: &storage.Schedule_DaysOfWeek{
						Days: weekdays,
					},
				}
			}
		}
	}
	return &sched
}

func TestSchedule(t *testing.T) {
	var cases = []struct {
		testname    string
		schedule    *storage.Schedule
		result      string
		expectError bool
	}{
		{
			testname: "Valid Time UTC Daily",
			schedule: newSchedule(12, 12, []int32{}, []int32{}),
			result:   "12 12 * * *",
		},
		{
			testname: "Valid Time UTC Weekly",
			schedule: newSchedule(34, 12, []int32{2}, []int32{}),
			result:   "34 12 * * 2",
		},
		{
			testname: "Valid Time UTC Weekly Multiple days",
			schedule: newSchedule(34, 12, []int32{2, 4}, []int32{}),
			result:   "34 12 * * 2,4",
		},
		{
			testname: "Valid Time UTC Monthly",
			schedule: newSchedule(34, 12, []int32{}, []int32{1}),
			result:   "34 12 1 * *",
		},
		{
			testname:    "Invalid Hour",
			schedule:    newSchedule(0, -1, []int32{0}, []int32{}),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Invalid weekday",
			schedule:    newSchedule(0, 0, []int32{7}, []int32{}),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Negative minute",
			schedule:    newSchedule(-5, 6, []int32{}, []int32{}),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Large minute",
			schedule:    newSchedule(66, 6, []int32{}, []int32{}),
			result:      "",
			expectError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			cron, err := ConvertToCronTab(c.schedule)
			require.Equal(t, c.expectError, err != nil)
			assert.Equal(t, c.result, cron)
		})
	}
}
