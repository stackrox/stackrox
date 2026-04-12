package env

import "time"

// ProcessEnricherInterval controls how often the process signal enricher
// scans the LRU cache for unresolved container metadata as a fallback.
//
// Most containers are resolved event-driven via the cluster entity store
// callback when the pod informer processes the container. This ticker
// only catches the rare race where a process signal arrives before the
// pod event — typically a 1-2s window.
// Default: 30s. The previous 5s default was unnecessarily aggressive.
var ProcessEnricherInterval = registerDurationSetting("ROX_SENSOR_PROCESS_ENRICHER_INTERVAL", 30*time.Second)
