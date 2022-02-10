package centralservices

import (
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"
)

func TestWithHelmtest(t *testing.T) {
	additionalTestDirs := []string{"../shared/scanner-full"}
	// TODO(ROX-8793): The tests will be enabled in a follow-up ticket because the current implementation breaks helm chart rendering.

	if !buildinfo.ReleaseBuild {
		additionalTestDirs = append(additionalTestDirs, "../shared/scanner-slim")
	}
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest", image.CentralServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
		HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(additionalTestDirs...)},
		MetaValuesOverridesFunc: func(values *charts.MetaValues) {
			// TODO(ROX-8793): The feature flag is enabled in development builds only and should be removed on release.
			if !buildinfo.ReleaseBuild {
				values.FeatureFlags[features.LocalImageScanning.EnvVar()] = true
			}
		},
	})
}
