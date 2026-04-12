package env

import "time"

// ProcessEnricherInterval controls how often the process signal enricher
// scans for unresolved container metadata. Lower values detect new
// processes faster; higher values reduce CPU on stable clusters.
// Default: 5s. Set to e.g. "30s" or "5m" for edge/stable environments.
var ProcessEnricherInterval = registerDurationSetting("ROX_SENSOR_PROCESS_ENRICHER_INTERVAL", 5*time.Second)
