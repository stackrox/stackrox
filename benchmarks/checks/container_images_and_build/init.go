package containerimagesandbuild

import "github.com/stackrox/rox/benchmarks/checks"

func init() {
	checks.AddToRegistry( // Part 4
		NewContainerUserBenchmark(),
		NewTrustedBaseImagesBenchmark(),
		NewUnnecessaryPackagesBenchmark(),
		NewScannedImagesBenchmark(),
		NewContentTrustBenchmark(),
		NewImageHealthcheckBenchmark(),
		NewImageUpdateInstructionsBenchmark(),
		NewSetuidSetGidPermissionsBenchmark(),
		NewImageCopyBenchmark(),
		NewImageSecretsBenchmark(),
		NewVerifiedPackagesBenchmark(),
	)
}
