package translation

import "github.com/stackrox/stackrox/operator/pkg/images"

var (
	imageOverrides = images.Overrides{
		images.Main:      "central.image.fullRef",
		images.Scanner:   "scanner.image.fullRef",
		images.ScannerDB: "scanner.dbImage.fullRef",
	}
)
