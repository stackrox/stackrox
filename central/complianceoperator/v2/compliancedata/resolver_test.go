package compliancedata

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/robfig/cron.v2"
)

func parseCron(t *testing.T, expr string) cron.Schedule {
	t.Helper()
	// Prefix with TZ=UTC to match production behavior.
	sched, err := cron.Parse("TZ=UTC " + expr)
	require.NoError(t, err)
	return sched
}

func TestFindPreviousFireTime(t *testing.T) {
	cases := map[string]struct {
		cronExpr string
		before   time.Time
		wantZero bool
		validate func(t *testing.T, got time.Time)
	}{
		"daily at 02:00 UTC": {
			cronExpr: "0 2 * * *",
			before:   time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC),
			validate: func(t *testing.T, got time.Time) {
				assert.Equal(t, time.Date(2026, 9, 2, 2, 0, 0, 0, time.UTC), got)
			},
		},
		"weekly Monday at 03:00": {
			cronExpr: "0 3 * * 1",
			before:   time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC), // Wednesday
			validate: func(t *testing.T, got time.Time) {
				assert.Equal(t, time.Date(2026, 8, 31, 3, 0, 0, 0, time.UTC), got) // Monday
			},
		},
		"monthly on day 31": {
			cronExpr: "0 4 31 * *",
			before:   time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), // April 15; Feb has no 31, Mar 31 is the previous
			validate: func(t *testing.T, got time.Time) {
				assert.Equal(t, time.Date(2026, 3, 31, 4, 0, 0, 0, time.UTC), got)
			},
		},
		"monthly on day 31 - Feb gap (widened lookback covers it)": {
			cronExpr: "0 4 31 * *",
			before:   time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC), // Before Mar 31; previous was Jan 31
			validate: func(t *testing.T, got time.Time) {
				assert.Equal(t, time.Date(2026, 1, 31, 4, 0, 0, 0, time.UTC), got) // 58 days back
			},
		},
		"no fire in lookback window": {
			cronExpr: "0 0 29 2 *", // Feb 29 only
			before:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), // 2026 is not a leap year
			wantZero: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sched := parseCron(t, tc.cronExpr)
			got := FindPreviousFireTime(sched, tc.before)
			if tc.wantZero {
				assert.True(t, got.IsZero(), "expected zero time, got %v", got)
			} else {
				tc.validate(t, got)
			}
		})
	}
}

func TestResolveCheck(t *testing.T) {
	// Daily at 02:00 UTC. Now = 2026-09-02 15:00 UTC.
	// grace = 85min (45+40). now-grace = 14:35. referenceFire = 02:00 today.
	now := time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)
	dailyConfig := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "daily-scan",
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         2,
			Minute:       0,
		},
	}
	noSchedConfig := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "no-sched",
		Schedule:       &storage.Schedule{IntervalType: storage.Schedule_UNSET},
	}
	oneTimeConfig := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "one-time",
		OneTimeScan:    true,
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         2,
		},
	}

	resolver := NewConfigResolver(
		[]*storage.ComplianceOperatorScanConfigurationV2{dailyConfig, noSchedConfig, oneTimeConfig},
		now,
	)

	// Reference fire for daily-scan: FindPreviousFireTime("0 2 * * *", now - 85m = 13:35)
	// → 02:00 today (2026-09-02 02:00 UTC)

	cases := map[string]struct {
		configName     string
		assessmentTime *time.Time
		want           State
	}{
		"healthy: assessment after reference": {
			configName:     "daily-scan",
			assessmentTime: timePtr(time.Date(2026, 9, 2, 2, 5, 0, 0, time.UTC)),
			want:           Current,
		},
		"frozen: assessment before reference": {
			configName:     "daily-scan",
			assessmentTime: timePtr(time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)),
			want:           Outdated,
		},
		"long-broken: assessment months old": {
			configName:     "daily-scan",
			assessmentTime: timePtr(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)),
			want:           Outdated,
		},
		"nil assessment time": {
			configName:     "daily-scan",
			assessmentTime: nil,
			want:           Unknown,
		},
		"no schedule config": {
			configName:     "no-sched",
			assessmentTime: timePtr(time.Date(2026, 9, 2, 2, 5, 0, 0, time.UTC)),
			want:           Unknown,
		},
		"one-time scan config": {
			configName:     "one-time",
			assessmentTime: timePtr(time.Date(2026, 9, 2, 2, 5, 0, 0, time.UTC)),
			want:           Unknown,
		},
		"unknown config name": {
			configName:     "nonexistent",
			assessmentTime: timePtr(time.Date(2026, 9, 2, 2, 5, 0, 0, time.UTC)),
			want:           Unknown,
		},
		"skew boundary: within 2min of reference → CURRENT": {
			configName:     "daily-scan",
			assessmentTime: timePtr(time.Date(2026, 9, 2, 1, 58, 30, 0, time.UTC)),
			want:           Current,
		},
		"just outside skew: 3min before reference → OUTDATED": {
			configName:     "daily-scan",
			assessmentTime: timePtr(time.Date(2026, 9, 2, 1, 56, 0, 0, time.UTC)),
			want:           Outdated,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := resolver.ResolveCheck(tc.configName, tc.assessmentTime)
			assert.Equal(t, tc.want, got, "state mismatch")
		})
	}
}

func TestResolveCheck_JustAfterFire_BrokenCluster(t *testing.T) {
	// Tests that a long-broken cluster stays OUTDATED even right after a fire.
	// Daily at 02:00 UTC. now = 02:30 (30 min after fire, within grace of 85min).
	// Grace-adjusted: now-grace = 01:05. referenceFire = yesterday 02:00.
	// Assessment time = 3 days ago → OUTDATED (held accountable to yesterday's cycle).
	now := time.Date(2026, 9, 2, 2, 30, 0, 0, time.UTC)
	cfg := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "daily",
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_DAILY,
			Hour:         2,
			Minute:       0,
		},
	}
	resolver := NewConfigResolver([]*storage.ComplianceOperatorScanConfigurationV2{cfg}, now)

	oldTime := time.Date(2026, 8, 30, 2, 0, 0, 0, time.UTC)
	got := resolver.ResolveCheck("daily", &oldTime)
	assert.Equal(t, Outdated, got, "broken cluster should stay OUTDATED even just after fire")
}

func TestResolveCheck_MonthlySchedule(t *testing.T) {
	// Monthly on day 15 at 04:00. now = Sept 16. Ref = Sept 15 04:00.
	now := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	cfg := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "monthly",
		Schedule: &storage.Schedule{
			IntervalType: storage.Schedule_MONTHLY,
			Hour:         4,
			Minute:       0,
			Interval: &storage.Schedule_DaysOfMonth_{
				DaysOfMonth: &storage.Schedule_DaysOfMonth{Days: []int32{15}},
			},
		},
	}
	resolver := NewConfigResolver([]*storage.ComplianceOperatorScanConfigurationV2{cfg}, now)

	// Assessment from Sept 15 04:05 → CURRENT
	currentTime := time.Date(2026, 9, 15, 4, 5, 0, 0, time.UTC)
	assert.Equal(t, Current, resolver.ResolveCheck("monthly", &currentTime))

	// Assessment from Aug 15 → OUTDATED (ref is Sept 15)
	oldTime := time.Date(2026, 8, 15, 4, 0, 0, 0, time.UTC)
	assert.Equal(t, Outdated, resolver.ResolveCheck("monthly", &oldTime))
}

func TestRollupState(t *testing.T) {
	cases := map[string]struct {
		states []State
		want   State
	}{
		"all unknown":                {[]State{Unknown, Unknown}, Unknown},
		"all current":                {[]State{Current, Current}, Current},
		"all outdated":               {[]State{Outdated, Outdated}, Outdated},
		"outdated dominates current": {[]State{Current, Outdated}, Outdated},
		"outdated dominates unknown": {[]State{Unknown, Outdated}, Outdated},
		"current over unknown":       {[]State{Unknown, Current}, Current},
		"mixed all three":            {[]State{Unknown, Current, Outdated}, Outdated},
		"empty":                      {[]State{}, Unknown},
		"single current":             {[]State{Current}, Current},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, RollupState(tc.states...))
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
