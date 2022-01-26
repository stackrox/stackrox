package centralservices

import (
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	helmChartTestUtils "github.com/stackrox/rox/pkg/helm/charts/testutils"

)

func TestWithHelmtest(t *testing.T) {
	additionalTestDirs := []string{"../shared/scanner-full"}
	if !buildinfo.ReleaseBuild {
		additionalTestDirs = append(additionalTestDirs, "../shared/scanner-slim")
	}
	helmChartTestUtils.RunHelmTestSuite(t, "testdata/helmtest", image.CentralServicesChartPrefix, helmChartTestUtils.RunHelmTestSuiteOpts{
		HelmTestOpts: []helmTest.LoaderOpt{helmTest.WithAdditionalTestDirs(additionalTestDirs...)},
	})
}
