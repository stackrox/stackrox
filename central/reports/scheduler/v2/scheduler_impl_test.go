package v2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/robfig/cron.v2"
)

func TestFindPreviousFireTime(t *testing.T) {
	// Note: robfig/cron.v2 interprets cron specs in the system's local timezone.
	// In production, Central runs in UTC. Tests use time.Local to be portable.
	loc := time.Now().Location()

	var cases = []struct {
		testname     string
		cronSpec     string
		now          time.Time
		expectedTime time.Time
	}{
		{
			testname:     "Daily at 14:30, now is 16:00 same day",
			cronSpec:     "30 14 * * *",
			now:          time.Date(2026, 2, 16, 16, 0, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 14, 30, 0, 0, loc),
		},
		{
			testname:     "Daily at 14:30, now is 10:00 next day - missed yesterday's run",
			cronSpec:     "30 14 * * *",
			now:          time.Date(2026, 2, 17, 10, 0, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 14, 30, 0, 0, loc),
		},
		{
			testname:     "Weekly Monday at 09:00, now is Wednesday",
			cronSpec:     "0 9 * * 1",
			now:          time.Date(2026, 2, 18, 12, 0, 0, 0, loc), // Wednesday
			expectedTime: time.Date(2026, 2, 16, 9, 0, 0, 0, loc),  // Previous Monday
		},
		{
			testname:     "Daily at 00:00, now is 23:59 same day",
			cronSpec:     "0 0 * * *",
			now:          time.Date(2026, 2, 16, 23, 59, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 0, 0, 0, 0, loc),
		},
		{
			testname:     "Daily at 23:59, now is 00:01 next day",
			cronSpec:     "59 23 * * *",
			now:          time.Date(2026, 2, 17, 0, 1, 0, 0, loc),
			expectedTime: time.Date(2026, 2, 16, 23, 59, 0, 0, loc),
		},
	}

	for _, c := range cases {
		t.Run(c.testname, func(t *testing.T) {
			schedule, err := cron.Parse(c.cronSpec)
			assert.NoError(t, err)

			previousFire := findPreviousFireTime(schedule, c.now)
			assert.Equal(t, c.expectedTime, previousFire)
		})
	}
}

func TestFindPreviousFireTimeReturnsZeroWhenNoFireInWindow(t *testing.T) {
	// Monthly on the 15th at 10:00, now is Jan 16.
	// The lookback window is 32 days, starting from Dec 15.
	// Dec 15 at 10:00 is the only fire in that window, so it should be found.
	// But if the schedule fires only on Feb 29 (leap year), and now is Jan 1,
	// there may be no fire in the 32-day window. Use a far-future date to simulate.
	loc := time.Now().Location()
	schedule, err := cron.Parse("0 10 29 2 *") // Only Feb 29
	assert.NoError(t, err)

	// Now is March 1, 2027 (non-leap year). Feb 29 doesn't exist, so no fire in window.
	previousFire := findPreviousFireTime(schedule, time.Date(2027, 3, 1, 0, 0, 0, 0, loc))
	assert.True(t, previousFire.IsZero(), "Expected zero time when no fire exists in lookback window")
}
