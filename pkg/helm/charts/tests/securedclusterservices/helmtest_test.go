package securedclusterservices

import (
	"testing"

	"github.com/stackrox/rox/image"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"
)

func TestWithHelmtest(t *testing.T) {
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest", image.SecuredClusterServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{})
}
