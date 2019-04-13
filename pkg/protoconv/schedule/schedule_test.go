package schedule

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSchedule(tod string, tz string, weekday int32) *storage.Schedule {
	var sched storage.Schedule

	sched.TimeOfDay = tod
	sched.Timezone = tz
	if weekday == -1 {
		sched.Interval = &storage.Schedule_Daily{Daily: &storage.Schedule_DailyInterval{}}
	} else {
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
			schedule: newSchedule("12:00PM", "UTC", -1),
			result:   "0 12 * * *",
		},
		{
			testname: "Valid Time PST Daily",
			schedule: newSchedule("12:00PM", "PST", -1),
			result:   "0 4 * * *",
		},
		{
			testname: "Valid Time UTC Weekly",
			schedule: newSchedule("12:00PM", "UTC", 2),
			result:   "0 12 * * 2",
		},
		{
			testname: "Valid Time PST Weekly",
			schedule: newSchedule("12:00PM", "PST", 2),
			result:   "0 4 * * 2",
		},
		{
			testname:    "Invalid Time",
			schedule:    newSchedule("12:00", "UTC", 0),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Invalid Timezone",
			schedule:    newSchedule("12:00", "BLA", 0),
			result:      "",
			expectError: true,
		},
		{
			testname:    "Invalid weekday",
			schedule:    newSchedule("12:00", "BLA", 7),
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
