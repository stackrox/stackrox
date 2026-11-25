package images

import "github.com/stackrox/rox/pkg/env"

// Environment variable settings for related image overrides.
// These define the set of images that are used for operands.
//
// Keep this list in sync with the `all-images` target in the top-level Makefile.
// That target shall produce the set of all stackrox images, which are built
// from this monorepo.
var (
	// These images are built from stackrox/stackrox.
	Main        = env.RegisterSetting("RELATED_IMAGE_MAIN")
	CentralDB   = env.RegisterSetting("RELATED_IMAGE_CENTRAL_DB")
	ScannerV4DB = env.RegisterSetting("RELATED_IMAGE_SCANNER_V4_DB")
	ScannerV4   = env.RegisterSetting("RELATED_IMAGE_SCANNER_V4")

	// These are built from their own repositories at stackrox/collector
	Collector = env.RegisterSetting("RELATED_IMAGE_COLLECTOR")
	// ... and stackrox/fact.
	Fact = env.RegisterSetting("RELATED_IMAGE_FACT")
)
