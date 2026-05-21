package mode

import "github.com/stackrox/rox/pkg/env"

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
	return CentralMode(env.CentralMode.Setting())
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
