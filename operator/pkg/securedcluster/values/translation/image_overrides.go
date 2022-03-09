package translation

import "github.com/stackrox/rox/operator/pkg/images"

var (
	imageOverrides = images.Overrides{
		images.Main:          "image.main.fullRef",
		images.CollectorSlim: "image.collector.slim.fullRef",
		images.CollectorFull: "image.collector.full.fullRef",
		images.ScannerSlim:   "image.scanner.fullRef",
		images.ScannerSlimDB: "image.scannerDb.fullRef",
	}
)
