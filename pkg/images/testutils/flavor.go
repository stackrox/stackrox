package testutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/images"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version"
)

// MakeImageFlavorForTest is to be used in tests where flavor is passed as a parameter. This makes it easier to test and expect
// values in the tests without having to inject values and rely on flavor determination in the production code.
func MakeImageFlavorForTest(t *testing.T) images.ImageFlavor {
	testutils.MustBeInTest(t)
	return images.ImageFlavor{
		MainRegistry:           "test.registry",
		MainImageName:          "main",
		MainImageTag:           "1.2.3",
		CollectorRegistry:      "test.registry",
		CollectorImageName:     "collector",
		CollectorImageTag:      "3.2.1-full",
		CollectorSlimImageName: "collector",
		CollectorSlimImageTag:  "3.2.1-slim",
		ScannerImageName:       "scanner",
		ScannerImageTag:        "2.2.2",
		ScannerDBImageName:     "scanner-db",
		ScannerDBImageTag:      "2.2.2",
		ChartRepo: images.ChartRepo{
			URL: "some.url/path/to/chart",
		},
		ImagePullSecrets: images.ImagePullSecrets{
			AllowNone: false,
		},
		Versions: version.Versions{
			BuildDate:        time.Now(),
			CollectorVersion: "3.2.1",
			MainVersion:      "1.2.3",
			ScannerVersion:   "2.2.2",
			ChartVersion:     "1.23.4",
		},
	}
}
