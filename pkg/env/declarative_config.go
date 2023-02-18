package env

import "time"

var (
	// DeclarativeConfigWatchInterval will set the duration for when to check for changes in declarative configuration
	// watch handlers.
	DeclarativeConfigWatchInterval = registerDurationSetting("ROX_DECLARATIVE_CONFIG_WATCH_INTERVAL", 5*time.Second)
	// DeclarativeConfigReconcileInterval will set the duration for when to reconcile declarative configurations.
	DeclarativeConfigReconcileInterval = registerDurationSetting("ROX_DECLARATIVE_CONFIG_RECONCILE_INTERVAL", time.Minute)
)
