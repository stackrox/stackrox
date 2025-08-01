package flavor

import (
	"path"
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
)

const testDir = "testdata/helmtest"

func TestWithDifferentFeatureFlags(t *testing.T) {
	testutils.SetVersion(t, testutils.GetExampleVersion(t))

	testCases := map[string]struct {
		featureFlags []string
		flavor       defaults.ImageFlavor
	}{
		"admission-controller-config": {
			featureFlags: []string{"ROX_ADMISSION_CONTROLLER_CONFIG"},
			flavor:       defaults.RHACSReleaseImageFlavor(),
		},
	}

	for testCaseName, testCaseSpec := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			imageFlavor := testCaseSpec.flavor
			helmChartTestUtils.RunHelmTestSuite(t, testDir, image.SecuredClusterServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
				Flavor: &imageFlavor,
				MetaValuesOverridesFunc: func(values *charts.MetaValues) {
					if values.FeatureFlags == nil {
						values.FeatureFlags = make(map[string]interface{})
					}
					for _, featureFlag := range testCaseSpec.featureFlags {
						values.FeatureFlags[featureFlag] = true
					}
				},
				HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(path.Join(testDir, testCaseName))},
			})
		})
	}
}
