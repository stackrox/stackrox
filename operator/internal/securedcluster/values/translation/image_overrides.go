package translation

import "github.com/stackrox/rox/operator/internal/images"

var (
	imageOverrides = images.Overrides{
		images.Main:        "image.main.fullRef",
		images.Collector:   "image.collector.fullRef",
		images.ScannerV4DB: "image.scannerV4DB.fullRef",
		images.ScannerV4:   "image.scannerV4.fullRef",
		images.Fact:        "image.fact.fullRef",
	}
)
