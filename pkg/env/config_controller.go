package env

import "time"

var (
	// ConfigControllerReconcileInterval sets the periodic reconciliation interval for the config-controller.
	ConfigControllerReconcileInterval = registerDurationSetting("ROX_CONFIG_CONTROLLER_RECONCILE_INTERVAL", 30*time.Minute)
)
