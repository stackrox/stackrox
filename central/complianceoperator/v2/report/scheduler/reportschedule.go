package scheduler

import (
	"time"

	"github.com/adhocore/gronx"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"gopkg.in/robfig/cron.v2"
)

// compile-time check that reportSchedule is a cron.Schedule, so the manager's shared cron
// instance computes each entry's fire times through it.
var _ cron.Schedule = (*reportSchedule)(nil)

// reportSchedule is the report schedule of a single scan configuration. It is immutable.
//
// Reporting is decoupled from scan completion (see compliance-report-partial-cycles.md,
// proposal 1): the report job of the scan cycle starting at a schedule tick T fires at T plus a
// fixed delay (ROX_COMPLIANCE_REPORT_SCHEDULE_DELAY), not when the cycle's scans complete. The
// delay is only a heuristic; the settler and the expectation audit that later iterations of the
// rollout add absorb its imprecision.
type reportSchedule struct {
	scanConfigID   string
	scanConfigName string
	cronTab        string
}

// newReportSchedule returns the report schedule of the given scan configuration, or an error if
// the configuration has no evaluable schedule: one-time scans and unscheduled configurations
// get no scheduled reports and are reported on demand only.
func newReportSchedule(scanConfig *storage.ComplianceOperatorScanConfigurationV2) (*reportSchedule, error) {
	if scanConfig.GetId() == "" {
		return nil, errors.New("scan configuration ID is required")
	}
	if scanConfig.GetOneTimeScan() {
		return nil, errors.Errorf("scan configuration %q is a one-time scan", scanConfig.GetScanConfigName())
	}
	cronTab, err := schedule.ConvertToCronTab(scanConfig.GetSchedule())
	if err != nil {
		return nil, errors.Wrapf(err, "scan configuration %q has no evaluable schedule", scanConfig.GetScanConfigName())
	}
	return &reportSchedule{
		scanConfigID:   scanConfig.GetId(),
		scanConfigName: scanConfig.GetScanConfigName(),
		cronTab:        cronTab,
	}, nil
}

// NextFireAfter returns the earliest report due time strictly after ref.
//
// The report of the cycle starting at tick T is due at T+delay, so the earliest due time after
// ref is the first schedule tick strictly after ref-delay -- a cycle that just started still
// gets its report -- plus the delay. A cycle whose report time has already passed is skipped:
// fire times are only ever in the future.
func (s *reportSchedule) NextFireAfter(ref time.Time) (time.Time, error) {
	delay := env.ComplianceReportScheduleDelay.DurationSetting()
	// Compliance schedules encode their hour/minute in UTC and gronx evaluates an expression in
	// the reference's location, so pin the reference to UTC (as the freshness sweeper does);
	// evaluating it in Central's local zone would shift the fire time. This also sidesteps
	// robfig's Parse, which evaluates cron specs in the process's local zone.
	tick, err := gronx.NextTickAfter(s.cronTab, ref.UTC().Add(-delay), false)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "unable to compute the next report time of scan configuration %q", s.scanConfigName)
	}
	return tick.Add(delay), nil
}

// Next implements cron.Schedule: it returns the next report due time strictly after the given
// time, or the zero time when it cannot be computed. robfig treats a zero next time as
// unsatisfiable and never runs the entry, so a transient computation failure disables the
// configuration's scheduled reports until the entry is replaced by an upsert.
func (s *reportSchedule) Next(after time.Time) time.Time {
	fire, err := s.NextFireAfter(after)
	if err != nil {
		log.Warnf("Compliance report scheduler: %v", err)
		return time.Time{}
	}
	return fire
}
