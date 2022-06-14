package images

import "github.com/stackrox/stackrox/pkg/env"

// Environment variable settings for related image overrides.
var (
	Main          = env.RegisterSetting("RELATED_IMAGE_MAIN")
	Scanner       = env.RegisterSetting("RELATED_IMAGE_SCANNER")
	ScannerSlim   = env.RegisterSetting("RELATED_IMAGE_SCANNER_SLIM")
	ScannerDB     = env.RegisterSetting("RELATED_IMAGE_SCANNER_DB")
	ScannerSlimDB = env.RegisterSetting("RELATED_IMAGE_SCANNER_DB_SLIM")
	CollectorSlim = env.RegisterSetting("RELATED_IMAGE_COLLECTOR_SLIM")
	CollectorFull = env.RegisterSetting("RELATED_IMAGE_COLLECTOR_FULL")
)
