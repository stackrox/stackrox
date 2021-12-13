package testutils

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/images"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/version"
)

func TestFlavor(t *testing.T) images.Flavor {
	testutils.MustBeInTest(t)
	return images.Flavor{
		MainRegistry:           "test.registry",
		MainImageName:          "main-test",
		MainImageTag:           "1.2.3",
		CollectorRegistry:      "collector.test.registry",
		CollectorImageName:     "collector-test-full",
		CollectorImageTag:      "3.2.1-full",
		CollectorSlimImageName: "collector-test-slim",
		CollectorSlimImageTag:  "3.2.1-slim",
		ScannerImageName:       "scanner-test",
		ScannerImageTag:        "2.2.2",
		ScannerDBImageName:     "scanner-db-test",
		ScannerDBImageTag:      "2.2.2",
		ChartRepo:              images.ChartRepo{
			URL: "some.url/path/to/chart",
		},
		ImagePullSecrets:       images.ImagePullSecrets{
			AllowNone: false,
		},
		Versions:               version.Versions{
			BuildDate:        time.Now(),
			CollectorVersion: "3.2.1",
			MainVersion:      "1.2.3",
			ScannerVersion:   "2.2.2",
			ChartVersion:     "1.0",
		},
	}
}


