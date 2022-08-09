package flavor

import (
	"path"
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/helm/charts"
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
		CentralDBImageName:     "custom-central-db",
		CentralDBImageTag:      "1.2.4",
		ScannerImageName:       "custom-scanner",
		ScannerSlimImageName:   "scanner-slim",
		ScannerImageTag:        "3.2.1",
		ScannerDBSlimImageName: "scanner-slim",
		ScannerDBImageName:     "custom-scanner-db",

		ChartRepo: defaults.ChartRepo{
			URL:     "url",
			IconURL: "url",
		},
		ImagePullSecrets: defaults.ImagePullSecrets{
			AllowNone: false,
		},
		Versions: testutils.GetExampleVersion(t),
	}
}

func TestOverriddenTagsAreRenderedInTheChart(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetVersion(t, testutils.GetExampleVersion(t))
	helmChartTestUtils.RunHelmTestSuite(t, testDir, image.CentralServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
		MetaValuesOverridesFunc: func(values *charts.MetaValues) {
			values.ImageTag = "custom-main"
			values.ScannerImageTag = "custom-scanner"
		},
		HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(path.Join(testDir, "override"))},
	})
}

func TestWithDifferentImageFlavors(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetVersion(t, testutils.GetExampleVersion(t))
	imageFlavorCases := map[string]defaults.ImageFlavor{
		"development": defaults.DevelopmentBuildImageFlavor(),
		"stackrox":    defaults.StackRoxIOReleaseImageFlavor(),
		"rhacs":       defaults.RHACSReleaseImageFlavor(),
		"custom":      customFlavor(t),
	}
	if buildinfo.ReleaseBuild {
		imageFlavorCases["opensource-release"] = defaults.OpenSourceImageFlavor()
	} else {
		imageFlavorCases["opensource-development"] = defaults.OpenSourceImageFlavor()
	}

	for name, imageFlavor := range imageFlavorCases {
		t.Run(name, func(t *testing.T) {
			imageFlavor := imageFlavor
			helmChartTestUtils.RunHelmTestSuite(t, testDir, image.CentralServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
				Flavor:       &imageFlavor,
				HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(path.Join(testDir, name))},
			})
		})
	}
}
