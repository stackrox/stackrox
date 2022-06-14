package centralservices

import (
	"testing"

	"github.com/stackrox/stackrox/image"
	helmChartTestUtils "github.com/stackrox/stackrox/pkg/helm/charts/testutils"
)

func TestWithHelmtest(t *testing.T) {
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest", image.CentralServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{})
}
