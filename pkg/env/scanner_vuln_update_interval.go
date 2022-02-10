package env

import "time"

// ScannerVulnUpdateInterval specifies the frequency at which Central should query the vulnerability endpoint.
var ScannerVulnUpdateInterval = registerDurationSetting("ROX_SCANNER_VULN_UPDATE_INTERVAL", 5*time.Minute)
