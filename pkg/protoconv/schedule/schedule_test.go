package schedule

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSchedule(minute int32, hour int32, weekday int32) *storage.Schedule {
	var sched storage.Schedule

	sched.Hour = hour
	sched.Minute = minute
	if weekday == -1 {
		sched.IntervalType = storage.Schedule_DAILY
	} else {
		sched.IntervalType = storage.Schedule_WEEKLY
		sched.Interval = &storage.Schedule_Weekly{Weekly: &storage.Schedule_WeeklyInterval{Day: weekday}}
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
			schedule: newSchedule(12, 12, -1),
			result:   "12 12 * * *",
		},
		{
			testname: "Valid Time UTC Weekly",
			schedule: newSchedule(34, 12, 2),
			result:   "34 12 * * 2",
		},
		{
			testname:    "Invalid Hour",
			schedule:    newSchedule(0, -1, 0),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Invalid weekday",
			schedule:    newSchedule(0, 0, 7),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Negative minute",
			schedule:    newSchedule(-5, 6, -1),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Large minute",
			schedule:    newSchedule(66, 6, -1),
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
