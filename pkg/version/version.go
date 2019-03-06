package version

import (
	"github.com/stackrox/rox/pkg/env"
)

var (
	mainVersion      string
	collectorVersion string
	scannerVersion   string
)

// GetMainVersion returns the tag of Prevent
func GetMainVersion() string {
	return mainVersion
}

// GetCollectorVersion returns the current collector tag
func GetCollectorVersion() string {
	if env.CollectorVersion.Setting() != "" {
		return env.CollectorVersion.Setting()
	}
	return collectorVersion
}

// GetScannerVersion returns the current scanner tag
func GetScannerVersion() string {
	return scannerVersion
}
