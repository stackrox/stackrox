package compliancedata

import (
	"time"

	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"gopkg.in/robfig/cron.v2"
)

var log = logging.LoggerForModule()

// State represents whether compliance data is current.
type State int

const (
	Unknown  State = 0
	Current  State = 1
	Outdated State = 2

	// lookbackDuration covers MONTHLY worst case: day-31 → next day-31 can be
	// ~62 days apart (Jan 31 → Mar 31 with Feb skip). 66 days gives margin.
	lookbackDuration = 66 * 24 * time.Hour

	// skew tolerance for CronJob trigger delay / clock drift.
	skew = 2 * time.Minute
)

// grace returns the configured or default grace period.
// Default = ComplianceScanScheduleWatcherTimeout + ComplianceScanWatcherTimeout.
func grace() time.Duration {
	if g := env.ComplianceOutdatedGrace.DurationSetting(); g > 0 {
		return g
	}
	return env.ComplianceScanScheduleWatcherTimeout.DurationSetting() +
		env.ComplianceScanWatcherTimeout.DurationSetting()
}

// ToProto converts a State to the proto ComplianceDataState enum.
func (s State) ToProto() v2.ComplianceDataState {
	switch s {
	case Current:
		return v2.ComplianceDataState_COMPLIANCE_DATA_STATE_CURRENT
	case Outdated:
		return v2.ComplianceDataState_COMPLIANCE_DATA_STATE_OUTDATED
	default:
		return v2.ComplianceDataState_COMPLIANCE_DATA_STATE_UNKNOWN
	}
}

// RollupState merges multiple states with OUTDATED > CURRENT > UNKNOWN precedence.
func RollupState(states ...State) State {
	result := Unknown
	for _, s := range states {
		if s == Outdated {
			return Outdated
		}
		if s == Current && result == Unknown {
			result = Current
		}
	}
	return result
}

// ConfigResolver pre-computes the reference fire time for a scan config's schedule.
// Use ResolveCheck / ResolveGroupedMin to evaluate individual checks or grouped aggregates.
type ConfigResolver struct {
	referenceFires map[string]time.Time // scan_config_name → reference fire time (UTC)
}

// NewConfigResolver builds a resolver from a set of scan configurations.
// now should be time.Now().UTC(). Configs with no schedule or unparseable
// crontab degrade to UNKNOWN (zero reference fire).
func NewConfigResolver(configs []*storage.ComplianceOperatorScanConfigurationV2, now time.Time) *ConfigResolver {
	cr := &ConfigResolver{
		referenceFires: make(map[string]time.Time, len(configs)),
	}
	g := grace()
	for _, cfg := range configs {
		sched := cfg.GetSchedule()
		if sched == nil || sched.GetIntervalType() == storage.Schedule_UNSET {
			continue
		}
		if cfg.GetOneTimeScan() {
			continue
		}
		cronTab, err := schedule.ConvertToCronTab(sched)
		if err != nil {
			log.Warnf("compliance outdated: cannot convert schedule for config %q: %v", cfg.GetScanConfigName(), err)
			continue
		}
		// Prefix with TZ=UTC: CO's CronJob fires in UTC (no spec.timeZone),
		// and robfig/cron.v2 defaults to time.Local which would misalign if
		// Central's container tz is not UTC.
		cronSched, err := cron.Parse("TZ=UTC " + cronTab)
		if err != nil {
			log.Warnf("compliance outdated: cannot parse crontab %q for config %q: %v", cronTab, cfg.GetScanConfigName(), err)
			continue
		}
		ref := FindPreviousFireTime(cronSched, now.Add(-g))
		cr.referenceFires[cfg.GetScanConfigName()] = ref
	}
	return cr
}

// ResolveCheck returns the data state for a single check result.
// scanConfigName is the check's scan_config_name; assessmentTime is its
// last_started_time (nil → UNKNOWN).
func (cr *ConfigResolver) ResolveCheck(scanConfigName string, assessmentTime *time.Time) State {
	ref, ok := cr.referenceFires[scanConfigName]
	if !ok || ref.IsZero() {
		return Unknown
	}
	if assessmentTime == nil {
		return Unknown
	}
	if assessmentTime.Before(ref.Add(-skew)) {
		return Outdated
	}
	return Current
}

// ResolveGroupedMin returns the data state for a (config, cluster) group
// given the MIN(last_started_time) of all checks in that group.
// A nil minTime (all NULLs) → UNKNOWN.
func (cr *ConfigResolver) ResolveGroupedMin(scanConfigName string, minTime *time.Time) State {
	return cr.ResolveCheck(scanConfigName, minTime)
}

// FindPreviousFireTime finds the most recent time before `before` that the
// cron schedule would have fired. Uses a widened lookback (66 days) to cover
// monthly schedules. Returns zero time if no fire found.
//
// IMPORTANT: `before` should be in UTC so the result matches CO's UTC CronJob.
func FindPreviousFireTime(cronSchedule cron.Schedule, before time.Time) time.Time {
	candidate := before.Add(-lookbackDuration)
	var previousFire time.Time
	for {
		next := cronSchedule.Next(candidate)
		if next.After(before) {
			break
		}
		previousFire = next
		candidate = next
	}
	return previousFire
}
