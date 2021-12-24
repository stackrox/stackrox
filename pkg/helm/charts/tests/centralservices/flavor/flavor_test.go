package flavor

import (
	"fmt"
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

func TestWithDifferentImageFlavors(t *testing.T) {
	testbuildinfo.SetForTest(t)
	testutils.SetExampleVersion(t)
	imageFlavorCases := map[string]defaults.ImageFlavor{
		"development": defaults.DevelopmentBuildImageFlavor(),
		"stackrox":    defaults.StackRoxIOReleaseImageFlavor(),
	}

	for name, imageFlavor := range imageFlavorCases {
		t.Run(name, func(t *testing.T) {
			helmImage := image.GetDefaultImage()
			tpl, err := helmImage.GetCentralServicesChartTemplate()
			require.NoError(t, err, "error retrieving chart template")
			metaVals := charts.GetMetaValuesForFlavor(imageFlavor)
			ch, err := tpl.InstantiateAndLoad(metaVals)
			require.NoError(t, err, "error instantiating chart")

			suite, err := helmTest.NewLoader("testdata/helmtest", helmTest.WithCustomFilePattern(fmt.Sprintf("%s.test.yaml", name))).LoadSuite()
			require.NoError(t, err, "failed to load helmtest suite")

			target := &helmTest.Target{
				Chart: ch,
				ReleaseOptions: chartutil.ReleaseOptions{
					Name:      "stackrox-central-services",
					Namespace: "stackrox",
					IsInstall: true,
				},
			}

			suite.Run(t, target)
		})
	}
}
