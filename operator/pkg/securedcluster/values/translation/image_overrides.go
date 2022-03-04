package translation

import "github.com/stackrox/rox/operator/pkg/images"

var (
	imageOverrides = images.Overrides{
		images.Main:          "image.main.fullRef",
		images.CollectorSlim: "image.collector.slim.fullRef",
		images.CollectorFull: "image.collector.full.fullRef",
		images.Scanner:   "scanner.image.fullRef",
		images.ScannerDB: "scanner.dbImage.fullRef",
		images.ScannerSlim:   "scanner.slimImage.fullRef",
		images.ScannerSlimDB: "scanner.slimDBImage.fullRef",
	}
)
