package flavor

import (
	"path"
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
)

const testDir = "testdata/helmtest"

func customFlavor(t *testing.T) defaults.ImageFlavor {
	return defaults.ImageFlavor{
		MainRegistry:           "example.io",
		MainImageName:          "custom-main",
		MainImageTag:           "1.2.3",
		ScannerImageName:       "custom-scanner",
		ScannerSlimImageName:   "scanner-slim",
		ScannerImageTag:        "3.2.1",
		ScannerDBSlimImageName: "scanner-slim",
		ScannerDBImageName:     "custom-scanner-db",

		ScannerDBImageTag: "3.2.1",
		ChartRepo: defaults.ChartRepo{
			URL: "url",
		},
		ImagePullSecrets: defaults.ImagePullSecrets{
			AllowNone: false,
		},
		Versions: testutils.GetExampleVersion(t),
	}
}

func TestWithDifferentImageFlavors(t *testing.T) {
	testbuildinfo.SetForTest(t)
	// having a function as value allows to successfully run this test without dependency to GOTAGS='' and GOTAGS='release'
	imageFlavorCases := map[string]func() defaults.ImageFlavor{
		"development": func() defaults.ImageFlavor {
			testutils.SetVersion(t, testutils.GetExampleVersion(t))
			return defaults.DevelopmentBuildImageFlavor()
		},
		"stackrox": func() defaults.ImageFlavor {
			testutils.SetVersion(t, testutils.GetExampleVersionUnified(t))
			return defaults.StackRoxIOReleaseImageFlavor()
		},
		"rhacs": func() defaults.ImageFlavor {
			testutils.SetVersion(t, testutils.GetExampleVersionUnified(t))
			return defaults.RHACSReleaseImageFlavor()
		},
		"custom": func() defaults.ImageFlavor {
			return customFlavor(t)
		},
	}

	for name, f := range imageFlavorCases {
		imageFlavor := f()
		t.Run(name, func(t *testing.T) {
			helmChartTestUtils.RunHelmTestSuite(t, testDir, image.CentralServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
				Flavor:       &imageFlavor,
				HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(path.Join(testDir, name))},
			})
		})
	}
}
