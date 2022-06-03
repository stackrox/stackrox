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
	testutils.SetVersion(t, testutils.GetExampleVersion(t))
	imageFlavorCases := map[string]defaults.ImageFlavor{
		"development": defaults.DevelopmentBuildImageFlavor(),
		"stackrox":    defaults.StackRoxIOReleaseImageFlavor(),
		"rhacs":       defaults.RHACSReleaseImageFlavor(),
	}
	if buildinfo.ReleaseBuild {
		imageFlavorCases["opensource-release"] = defaults.OpenSourceImageFlavor()
	} else {
		imageFlavorCases["opensource-development"] = defaults.OpenSourceImageFlavor()
	}

	for name, f := range imageFlavorCases {
		t.Run(name, func(t *testing.T) {
			f := f
			helmChartTestUtils.RunHelmTestSuite(t, testDir, image.SecuredClusterServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
				Flavor: &f,
				MetaValuesOverridesFunc: func(values *charts.MetaValues) {
					values.ClusterName = "test"
				},
				HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(path.Join(testDir, name))},
			})
		})
	}
}
