package version

import (
	"runtime"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/version/internal"
)

// GetMainVersion returns the tag of Rox.
func GetMainVersion() string {
	return internal.MainVersion
}

// GetCollectorVersion returns the current Collector tag.
func GetCollectorVersion() string {
	if env.CollectorVersion.Setting() != "" {
		return env.CollectorVersion.Setting()
	}
	return internal.CollectorVersion
}

// GetScannerVersion returns the current Scanner tag.
func GetScannerVersion() string {
	return internal.ScannerVersion
}

// Versions represents a collection of various pieces of version information.
type Versions struct {
	BuildDate        time.Time `json:"BuildDate"`
	CollectorVersion string    `json:"CollectorVersion"`
	GitCommit        string    `json:"GitCommit"`
	GoVersion        string    `json:"GoVersion"`
	MainVersion      string    `json:"MainVersion"`
	Platform         string    `json:"Platform"`
	ScannerVersion   string    `json:"ScannerVersion"`
}

// GetAllVersions returns all of the various pieces of version information.
func GetAllVersions() Versions {
	return Versions{
		BuildDate:        buildinfo.BuildTimestamp(),
		CollectorVersion: GetCollectorVersion(),
		GitCommit:        internal.GitShortSha,
		GoVersion:        runtime.Version(),
		MainVersion:      GetMainVersion(),
		Platform:         runtime.GOOS + "/" + runtime.GOARCH,
		ScannerVersion:   GetScannerVersion(),
	}
}
