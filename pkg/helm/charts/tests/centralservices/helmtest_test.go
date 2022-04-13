package centralservices

import (
	"testing"

	"github.com/stackrox/stackrox/image"
	"github.com/stackrox/stackrox/pkg/buildinfo"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/helm/charts"
	helmChartTestUtils "github.com/stackrox/stackrox/pkg/helm/charts/testutils"
)

func TestWithHelmtest(t *testing.T) {
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest", image.CentralServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
		MetaValuesOverridesFunc: func(values *charts.MetaValues) {
			// TODO(ROX-8793): The feature flag is enabled in development builds only and should be removed on release.
			if !buildinfo.ReleaseBuild {
				values.FeatureFlags[features.LocalImageScanning.EnvVar()] = true
			}
		},
	})
}
