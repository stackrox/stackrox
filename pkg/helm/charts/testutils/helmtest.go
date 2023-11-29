package testutils

import (
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

// RunHelmTestSuiteOpts defines options to configure the helm test suite.
type RunHelmTestSuiteOpts struct {
	Flavor                  *defaults.ImageFlavor
	HelmTestOpts            []helmTest.LoaderOpt
	MetaValuesOverridesFunc func(*charts.MetaValues)
}

// RunHelmTestSuite runs a helm test suite against the specified chart. The chart is taken from the
// Helm charts mounted based on the image.ChartPrefix.
// The opts can be used to configure the test environment.
func RunHelmTestSuite(t *testing.T, testDir string, chartPrefix image.ChartPrefix, opts RunHelmTestSuiteOpts) {
	if opts.Flavor == nil {
		flavor := flavorUtils.MakeImageFlavorForTest(t)
		opts.Flavor = &flavor
	}

	helmImage := image.GetDefaultImage()
	tpl, err := helmImage.GetChartTemplate(chartPrefix)
	require.NoErrorf(t, err, "error retrieving template for chart %s", chartPrefix)

	metaVals := charts.GetMetaValuesForFlavor(*opts.Flavor)
	if opts.MetaValuesOverridesFunc != nil {
		opts.MetaValuesOverridesFunc(metaVals)
	}

	ch, err := tpl.InstantiateAndLoad(metaVals)
	require.NoErrorf(t, err, "error instantiating chart %s", chartPrefix)

	suite, err := helmTest.NewLoader(testDir, opts.HelmTestOpts...).LoadSuite()
	require.NoError(t, err, "failed to load helmtest suite")

	target := &helmTest.Target{
		Chart: ch,
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      getReleaseName(t, chartPrefix),
			Namespace: "stackrox",
			IsInstall: true,
		},
	}

	suite.Run(t, target)
}

func getReleaseName(t *testing.T, chart image.ChartPrefix) string {
	switch chart {
	case image.SecuredClusterServicesChartPrefix:
		return "stackrox-secured-cluster-services"
	case image.CentralServicesChartPrefix:
		return "stackrox-central-services"
	}
	require.Fail(t, "Chart prefix %q unknown", chart)
	return ""
}
