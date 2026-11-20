package settle

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// expectationCutoff returns the oldest LastStartedTime that still counts as
// fresh, and whether the scan configuration carries a cron-based expectation
// at all. There is no expectation when the configuration has no schedule
// (one-time scans), when the schedule is not one the window can compute, or
// when no scheduled fire has happened since the configuration was created --
// nothing can be overdue before the first expectation elapses.
//
// The Compliance Operator evaluates the scan configuration's cron in the
// cluster's local timezone; Central computes the window in UTC. The grace
// period (ROX_COMPLIANCE_REPORT_SCHEDULE_GRACE) absorbs both the operator's
// and sensor's scheduling delays and this timezone approximation.
func (s *Settler) expectationCutoff(scanConfig *storage.ComplianceOperatorScanConfigurationV2, now time.Time) (time.Time, bool) {
	lastFire, prevFire, ok := lastTwoFires(scanConfig.GetSchedule(), now)
	if !ok {
		return time.Time{}, false
	}
	if created := scanConfig.GetCreatedTime(); created != nil && lastFire.Before(created.AsTime()) {
		// No scheduled fire has happened since the configuration was created.
		return time.Time{}, false
	}
	// Within the grace period of the latest fire, the Compliance Operator may
	// not have gotten to the scans yet: expect the previous fire instead, so a
	// report run shortly after a fire does not flag every scan as overdue.
	if now.Before(lastFire.Add(s.grace)) {
		return prevFire, true
	}
	return lastFire, true
}

// ReportDue returns the most recent schedule fire of the scan configuration at
// or before now, and the time at which the scheduled report for that fire is
// due (fire + delay). ok is false when there is no report to run: the
// configuration has no schedule (one-time scans are handled by the scheduler
// separately, keyed on the creation time), the schedule cannot be computed, or
// no fire has happened since the configuration was created -- a report for a
// fire that predates the configuration is pointless, since the audit would
// classify every scan as fresh and the report would carry no data.
func ReportDue(scanConfig *storage.ComplianceOperatorScanConfigurationV2, now time.Time, delay time.Duration) (fire, due time.Time, ok bool) {
	lastFire, _, ok := lastTwoFires(scanConfig.GetSchedule(), now)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	if created := scanConfig.GetCreatedTime(); created != nil && lastFire.Before(created.AsTime()) {
		return time.Time{}, time.Time{}, false
	}
	return lastFire, lastFire.Add(delay), true
}

// lastTwoFires returns the two most recent times at or before now at which the
// schedule fires, most recent first. ok is false when the schedule is nil, of
// an unsupported interval type, or carries no days. When only one fire is
// found in the search horizon (e.g. a young configuration), the second return
// value is the zero time, which the cutoff treats as "everything is fresh".
//
// Days are searched backward from today, so the result is exact for the daily,
// weekly and monthly schedules the scan configuration API allows.
func lastTwoFires(schedule *storage.Schedule, now time.Time) (time.Time, time.Time, bool) {
	if schedule == nil {
		return time.Time{}, time.Time{}, false
	}
	hour, minute := int(schedule.GetHour()), int(schedule.GetMinute())
	dayAllowed := func(time.Time) bool { return true }
	switch schedule.GetIntervalType() {
	case storage.Schedule_DAILY:
		// Every day is allowed.
	case storage.Schedule_WEEKLY:
		days := weekdaySet(schedule)
		if len(days) == 0 {
			return time.Time{}, time.Time{}, false
		}
		dayAllowed = func(d time.Time) bool { return days.Contains(int(d.Weekday())) }
	case storage.Schedule_MONTHLY:
		days := set.NewIntSet()
		for _, dom := range schedule.GetDaysOfMonth().GetDays() {
			days.Add(int(dom))
		}
		if len(days) == 0 {
			return time.Time{}, time.Time{}, false
		}
		dayAllowed = func(d time.Time) bool { return days.Contains(d.Day()) }
	default:
		return time.Time{}, time.Time{}, false
	}

	var fires []time.Time
	// Searching 93 days back covers two occurrences of any schedule the scan
	// configuration API allows, including a monthly day-of-31 across short
	// months (e.g. from mid-October back to July 31st).
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for back := 0; back < 93 && len(fires) < 2; back++ {
		if dayAllowed(day) {
			fire := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, time.UTC)
			if !fire.After(now) {
				fires = append(fires, fire)
			}
		}
		day = day.AddDate(0, 0, -1)
	}
	if len(fires) == 0 {
		return time.Time{}, time.Time{}, false
	}
	if len(fires) == 1 {
		return fires[0], time.Time{}, true
	}
	return fires[0], fires[1], true
}

// weekdaySet returns the days of week a weekly schedule fires on, in Go's
// numbering (0 = Sunday), which matches the cron day-of-week numbering the
// schedule converts to (see protoconv.ConvertToCronTab).
func weekdaySet(schedule *storage.Schedule) set.IntSet {
	days := set.NewIntSet()
	if interval, ok := schedule.GetInterval().(*storage.Schedule_Weekly); ok && interval != nil {
		days.Add(int(interval.Weekly.GetDay()))
	}
	for _, day := range schedule.GetDaysOfWeek().GetDays() {
		days.Add(int(day))
	}
	return days
}
