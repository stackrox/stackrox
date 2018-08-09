package env

import (
	"os"

	"github.com/stackrox/rox/generated/api/v1"
)

var (
	// ScanID is used to provide the benchmark services with the current scan
	ScanID = Setting(scanID{})

	// Checks is used to provide the benchmark services with the checks that need to be run as part of the benchmark
	Checks = Setting(checks{})

	// BenchmarkID is used to provide the benchmark service with the benchmark name
	BenchmarkID = Setting(benchmarkID{})

	// BenchmarkCompletion is used to provide the benchmark service with whether or not the benchmark container should exit
	BenchmarkCompletion = Setting(benchmarkCompletion{})

	// BenchmarkReason is used to provide the benchmark service with why the benchmark was run (e.g. SCHEDULED or TRIGGERED)
	BenchmarkReason = Setting(benchmarkReason{})
)

type scanID struct{}

func (s scanID) EnvVar() string {
	return "ROX_PREVENT_SCAN_ID"
}

func (s scanID) Setting() string {
	return os.Getenv(s.EnvVar())
}

type checks struct{}

func (c checks) EnvVar() string {
	return "ROX_PREVENT_CHECKS"
}

func (c checks) Setting() string {
	return os.Getenv(c.EnvVar())
}

type benchmarkID struct{}

func (c benchmarkID) EnvVar() string {
	return "ROX_PREVENT_BENCHMARK_ID"
}

func (c benchmarkID) Setting() string {
	return os.Getenv(c.EnvVar())
}

type benchmarkCompletion struct{}

func (c benchmarkCompletion) EnvVar() string {
	return "ROX_PREVENT_BENCHMARK_COMPLETION"
}

func (c benchmarkCompletion) Setting() string {
	return os.Getenv(c.EnvVar())
}

type benchmarkReason struct{}

func (c benchmarkReason) EnvVar() string {
	return "ROX_PREVENT_BENCHMARK_REASON"
}

func (c benchmarkReason) Setting() string {
	if val, ok := os.LookupEnv(c.EnvVar()); ok {
		return val
	}
	return v1.BenchmarkReason_SCHEDULED.String()
}
