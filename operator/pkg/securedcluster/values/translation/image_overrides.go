package translation

import "github.com/stackrox/rox/operator/pkg/images"

var (
	imageOverrides = images.Overrides{
		images.Main:             "image.main.fullRef",
		images.CollectorSlim:    "image.collector.slim.fullRef",
		images.CollectorFull:    "image.collector.full.fullRef",
		images.ScannerSlim:      "image.scanner.fullRef",
		images.ScannerSlimDB:    "image.scannerDb.fullRef",
		images.ScannerV4DB:      "image.scannerV4.db.fullRef",
		images.ScannerV4Indexer: "scannerV4.indexer.image.fullRef",
		images.ScannerV4Matcher: "scannerV4.matcher.image.fullRef",
	}
)
