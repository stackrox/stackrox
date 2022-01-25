package flavor

import (
	"path"
	"testing"

	helmTest "github.com/stackrox/helmtest/pkg/framework"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

const testDir = "testdata/helmtest"

func TestWithDifferentImageFlavors(t *testing.T) {
	testbuildinfo.SetForTest(t)
	// having a function as value allows to successfully run this test without dependency to GOTAGS='' and GOTAGS='release'
	imageFlavorCases := map[string]func() defaults.ImageFlavor{
		"development": func() defaults.ImageFlavor {
			testutils.SetVersion(t, testutils.GetExampleVersion(t))
			return defaults.DevelopmentBuildImageFlavor()
		},
		"stackrox": func() defaults.ImageFlavor {
			testutils.SetVersion(t, testutils.GetExampleVersionUnified(t))
			return defaults.StackRoxIOReleaseImageFlavor()
		},
	}

	for name, f := range imageFlavorCases {
		imageFlavor := f()
		t.Run(name, func(t *testing.T) {
			helmImage := image.GetDefaultImage()
			tpl, err := helmImage.GetSecuredClusterServicesChartTemplate()
			require.NoError(t, err, "error retrieving chart template")
			metaVals := charts.GetMetaValuesForFlavor(imageFlavor)
			metaVals.ClusterName = "Test"
			ch, err := tpl.InstantiateAndLoad(metaVals)
			require.NoError(t, err, "error instantiating chart")

			suite, err := helmTest.NewLoader(testDir, helmTest.WithAdditionalTestDirs(path.Join(), name)).LoadSuite()
			require.NoError(t, err, "failed to load helmtest suite")

			target := &helmTest.Target{
				Chart: ch,
				ReleaseOptions: chartutil.ReleaseOptions{
					Name:      "stackrox-secured-cluster-services",
					Namespace: "stackrox",
					IsInstall: true,
				},
			}

			suite.Run(t, target)
		})
	}
}
