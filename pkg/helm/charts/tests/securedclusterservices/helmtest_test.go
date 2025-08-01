package securedclusterservices

import (
	"testing"

	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"
)

func enableDesiredFeatureFlags(metaVals *charts.MetaValues) {
	featureFlagsToEnable := []string{
		"ROX_ADMISSION_CONTROLLER_CONFIG",
	}
	if metaVals.FeatureFlags == nil {
		metaVals.FeatureFlags = make(map[string]interface{})
	}
	for _, featureFlag := range featureFlagsToEnable {
		metaVals.FeatureFlags[featureFlag] = true
	}
}

func TestWithHelmtest(t *testing.T) {
	testSuiteOpts := helmChartTestUtils.RunHelmTestSuiteOpts{
		MetaValuesOverridesFunc: enableDesiredFeatureFlags,
	}
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest", image.SecuredClusterServicesChartPrefix, testSuiteOpts)
}
