package testutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version"
)

// MakeImageFlavorForTest is to be used in tests where flavor is passed as a parameter. This makes it easier to test and expect
// values in the tests without having to inject values and rely on flavor determination in the production code.
func MakeImageFlavorForTest(t *testing.T) defaults.ImageFlavor {
	testutils.MustBeInTest(t)
	return defaults.ImageFlavor{
		MainRegistry:           "test.registry",
		MainImageName:          "main",
		MainImageTag:           "1.2.3",
		CollectorRegistry:      "test.registry",
		CollectorImageName:     "collector",
		CollectorImageTag:      "3.2.1-latest",
		CollectorSlimImageName: "collector",
		CollectorSlimImageTag:  "3.2.1-slim",
		ScannerImageName:       "scanner",
		ScannerSlimImageName:   "scanner-slim",
		ScannerImageTag:        "2.2.2",
		ScannerDBImageName:     "scanner-db",
		ScannerDBSlimImageName: "scanner-db-slim",
		ChartRepo: defaults.ChartRepo{
			URL: "some.url/path/to/chart",
		},
		ImagePullSecrets: defaults.ImagePullSecrets{
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
