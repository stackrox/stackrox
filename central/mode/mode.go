package mode

import (
	"fmt"

	"github.com/stackrox/rox/pkg/env"
)

// CentralMode represents the operational mode for Central.
type CentralMode string

const (
	// Full mode runs all Central subsystems (default behavior).
	Full CentralMode = "full"
	// Reports mode runs only report schedulers, no API server or background workers.
	Reports CentralMode = "reports"
	// CronJob mode runs a single periodic task and exits.
	CronJob CentralMode = "cronjob"
)

// Get returns the current Central mode from the ROX_CENTRAL_MODE environment variable.
func Get() CentralMode {
	m := CentralMode(env.CentralMode.Setting())
	switch m {
	case Full, Reports, CronJob:
		return m
	default:
		panic(fmt.Sprintf("invalid ROX_CENTRAL_MODE: %q (must be full, reports, or cronjob)", m))
	}
}

// IsBackgroundWorkersEnabled returns true if background workers should start.
// Background workers include reprocessor, pruning, cloud sources, etc.
func IsBackgroundWorkersEnabled() bool {
	return Get() == Full
}

// IsAPIEnabled returns true if the API server and sensor connections should be active.
func IsAPIEnabled() bool {
	return Get() == Full
}

// IsReportsEnabled returns true if report generation should be active.
func IsReportsEnabled() bool {
	m := Get()
	return m == Full || m == Reports
}

// IsCronJob returns true if Central is running in cronjob mode.
func IsCronJob() bool {
	return Get() == CronJob
}
