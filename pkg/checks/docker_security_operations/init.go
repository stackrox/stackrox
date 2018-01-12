package dockersecurityoperations

import "bitbucket.org/stack-rox/apollo/pkg/checks"

func init() {
	checks.AddToRegistry(
		NewImageSprawlBenchmark(),
		NewContainerSprawlBenchmark(),
	)
}
