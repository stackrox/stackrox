package version

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/version/internal"
)

// GetMainVersion returns the tag of Prevent
func GetMainVersion() string {
	return internal.MainVersion
}

// GetCollectorVersion returns the current collector tag
func GetCollectorVersion() string {
	if env.CollectorVersion.Setting() != "" {
		return env.CollectorVersion.Setting()
	}
	return internal.CollectorVersion
}

// GetScannerVersion returns the current scanner tag
func GetScannerVersion() string {
	return internal.ScannerVersion
}
