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

func TestOverriddenTagsAreRenderedInTheChart(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetVersion(t, testutils.GetExampleVersion(t))
	helmChartTestUtils.RunHelmTestSuite(t, testDir, image.SecuredClusterServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
		MetaValuesOverridesFunc: func(values *charts.MetaValues) {
			values.ClusterName = "test"
			values.ImageTag = "custom-main"
			values.CollectorFullImageTag = "custom-collector-full"
			values.CollectorSlimImageTag = "custom-collector-slim"
			values.ScannerImageTag = "custom-scanner"
		},
		HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(path.Join(testDir, "override"))},
	})
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
	}
	opensourceDir := "opensource-development"
	if buildinfo.ReleaseBuild {
		opensourceDir = "opensource-release"
	}
	imageFlavorCases[opensourceDir] = func() defaults.ImageFlavor {
		testutils.SetVersion(t, testutils.GetExampleVersion(t))
		return defaults.OpenSourceImageFlavor()
	}

	for name, f := range imageFlavorCases {
		imageFlavor := f()
		t.Run(name, func(t *testing.T) {
			helmChartTestUtils.RunHelmTestSuite(t, testDir, image.SecuredClusterServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
				Flavor: &imageFlavor,
				MetaValuesOverridesFunc: func(values *charts.MetaValues) {
					values.ClusterName = "test"
				},
				HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(path.Join(testDir, name))},
			})
		})
	}
}
