package dockersecurityoperations

import "github.com/stackrox/rox/benchmarks/checks"

func init() {
	checks.AddToRegistry(
		NewImageSprawlBenchmark(),
		NewContainerSprawlBenchmark(),
	)
}
