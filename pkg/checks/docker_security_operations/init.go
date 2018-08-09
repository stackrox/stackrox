package dockersecurityoperations

import "github.com/stackrox/rox/pkg/checks"

func init() {
	checks.AddToRegistry(
		NewImageSprawlBenchmark(),
		NewContainerSprawlBenchmark(),
	)
}
