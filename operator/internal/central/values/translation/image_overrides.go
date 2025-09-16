package translation

import "github.com/stackrox/rox/operator/internal/images"

var (
	imageOverrides = images.Overrides{
		images.Main:        "central.image.fullRef",
		images.CentralDB:   "central.db.image.fullRef",
		images.Scanner:     "scanner.image.fullRef",
		images.ScannerDB:   "scanner.dbImage.fullRef",
		images.ScannerV4DB: "scannerV4.db.image.fullRef",
		images.ScannerV4:   "scannerV4.image.fullRef",
	}
)
