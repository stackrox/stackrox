package translation

import "github.com/stackrox/rox/operator/pkg/images"

var (
	imageOverrides = images.Overrides{
		images.Main:      "central.image.fullRef",
		images.CentralDB: "central.db.image.fullRef",
		images.Scanner:   "scanner.image.fullRef",
		images.ScannerDB: "scanner.dbImage.fullRef",
	}
)
