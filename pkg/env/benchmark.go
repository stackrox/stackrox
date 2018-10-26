package env

import (
	"github.com/stackrox/rox/generated/api/v1"
)

var (
	// ScanID is used to provide the benchmark services with the current scan
	ScanID = NewSetting("ROX_SCAN_ID")

	// Checks is used to provide the benchmark services with the checks that need to be run as part of the benchmark
	Checks = NewSetting("ROX_CHECKS")

	// BenchmarkID is used to provide the benchmark service with the benchmark name
	BenchmarkID = NewSetting("ROX_BENCHMARK_ID")

	// BenchmarkCompletion is used to provide the benchmark service with whether or not the benchmark container should exit
	BenchmarkCompletion = NewSetting("ROX_BENCHMARK_COMPLETION")

	// BenchmarkReason is used to provide the benchmark service with why the benchmark was run (e.g. SCHEDULED or TRIGGERED)
	BenchmarkReason = NewSetting("ROX_BENCHMARK_REASON", WithDefault(v1.BenchmarkReason_SCHEDULED.String()), AllowEmpty())
)
