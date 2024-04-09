package centralservices

import (
	"testing"

	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"
)

func configureMetaValues(metaValues *charts.MetaValues) {
	// Activate certain feature flags.
	// This allows us to execute tests for features which are currently disabled by default.
	metaValues.FeatureFlags["ROX_SCANNER_V4_SUPPORT"] = "true"
}

func TestWithHelmtest(t *testing.T) {
	testSuiteOpts := helmChartTestUtils.RunHelmTestSuiteOpts{
		MetaValuesOverridesFunc: configureMetaValues,
	}
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest", image.CentralServicesChartPrefix, testSuiteOpts)
}
