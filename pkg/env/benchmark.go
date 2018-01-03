package env

import "os"

var (
	// ScanID is used to provide the benchmark services with the current scan
	ScanID = Setting(scanID{})

	// Checks is used to provide the benchmark services with the checks that need to be run as part of the benchmark
	Checks = Setting(checks{})
)

type scanID struct{}

func (s scanID) EnvVar() string {
	return "ROX_APOLLO_SCAN_ID"
}

func (s scanID) Setting() string {
	return os.Getenv(s.EnvVar())
}

type checks struct{}

func (c checks) EnvVar() string {
	return "ROX_APOLLO_CHECKS"
}

func (c checks) Setting() string {
	return os.Getenv(c.EnvVar())
}
