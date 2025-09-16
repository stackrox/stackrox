package images

import "github.com/stackrox/rox/pkg/env"

// Environment variable settings for related image overrides.
var (
	Main          = env.RegisterSetting("RELATED_IMAGE_MAIN")
	CentralDB     = env.RegisterSetting("RELATED_IMAGE_CENTRAL_DB")
	Scanner       = env.RegisterSetting("RELATED_IMAGE_SCANNER")
	ScannerSlim   = env.RegisterSetting("RELATED_IMAGE_SCANNER_SLIM")
	ScannerDB     = env.RegisterSetting("RELATED_IMAGE_SCANNER_DB")
	ScannerSlimDB = env.RegisterSetting("RELATED_IMAGE_SCANNER_DB_SLIM")
	Collector     = env.RegisterSetting("RELATED_IMAGE_COLLECTOR")
	ScannerV4DB   = env.RegisterSetting("RELATED_IMAGE_SCANNER_V4_DB")
	ScannerV4     = env.RegisterSetting("RELATED_IMAGE_SCANNER_V4")
)
