package centralservices

import (
	"testing"

	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"
)

func TestChartLabelTruncation(t *testing.T) {
	testSuiteOpts := helmChartTestUtils.RunHelmTestSuiteOpts{
		MetaValuesOverridesFunc: func(vals *charts.MetaValues) {
			// Create a very long chart version that will exceed 63 chars (combined with the chart name)
			// which ends with a non alphanumeric character to test the truncation and sanitization logic.
			vals.Versions.ChartVersion = "400.0.0-extremely-long-prerelease-version-string-for-testing-"
		},
	}
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest-truncation", image.CentralServicesChartPrefix, testSuiteOpts)
}
