package image

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chart"
)

func init() {
	testutils.SetMainVersion(&testing.T{}, "3.0.55.0")
}

func TestManager(t *testing.T) {
	suite.Run(t, new(embedTestSuite))
}

type embedTestSuite struct {
	suite.Suite

	image *Image
}

func (s *embedTestSuite) SetupTest() {
	testutils.SetExampleVersion(s.T())
	s.image = GetDefaultImage()
}

func (s *embedTestSuite) TestEmbedAllFiles() {
	err := filepath.WalkDir("templates", func(p string, d fs.DirEntry, err error) error {
		s.Require().NoError(err)

		_, statErr := fs.Stat(s.image.fs, p)
		s.Require().NoError(statErr, "Could not find file or directory %q, to fix this add %q to the go embed directive",
			p, path.Join(path.Dir(p), "*"))
		return nil
	})
	s.Require().NoError(err)
}

func (s *embedTestSuite) TestChartTemplatesAvailable() {
	_, err := s.image.GetCentralServicesChartTemplate()
	s.Require().NoError(err, "failed to load central services chart")
	_, err = s.image.GetSecuredClusterServicesChartTemplate()
	s.Require().NoError(err, "failed to load secured cluster services chart")
}

func (s *embedTestSuite) TestLoadChartForFlavor() {
	testCases := []defaults.ImageFlavor{
		flavorUtils.MakeImageFlavorForTest(s.T()),
		defaults.DevelopmentBuildImageFlavor(),
		defaults.StackRoxIOReleaseImageFlavor(),
		defaults.RHACSReleaseImageFlavor(),
		defaults.OpenSourceImageFlavor(),
	}

	for _, flavor := range testCases {
		testName := fmt.Sprintf("Image Flavor %s", flavor.MainRegistry)
		s.Run(testName, func() {
			chart, err := s.image.LoadChart(CentralServicesChartPrefix, charts.GetMetaValuesForFlavor(flavor))
			s.Require().NoError(err)
			s.Equal("stackrox-central-services", chart.Name())

			chart, err = s.image.LoadChart(SecuredClusterServicesChartPrefix, charts.GetMetaValuesForFlavor(flavor))
			s.Require().NoError(err)
			s.Equal("stackrox-secured-cluster-services", chart.Name())
		})
	}
}

func (s *embedTestSuite) TestSecuredClusterChartShouldIgnoreFeatureFlagValuesOnReleaseBuilds() {
	metaVals := charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(s.T()))
	metaVals.ReleaseBuild = true

	chart, err := s.image.LoadChart(SecuredClusterServicesChartPrefix, metaVals)
	s.Require().NoError(err)
	s.NotEmpty(chart.Files)

	for _, f := range chart.Files {
		if f.Name == "feature-flag-values.yaml" {
			s.Fail("Found feature-flag-values.yaml in release build but should be ignored.")
		}
	}
}

// This test will be removed after the scanner integration is finished. It is critical to check that no scanner manifests are contained within
// secured cluster.
func (s *embedTestSuite) TestLoadSecuredClusterScanner() {
	testCases := map[string]struct {
		kubectlOutput           bool
		expectScannerFilesExist bool
	}{

		"contains scanner manifests": {
			kubectlOutput:           false,
			expectScannerFilesExist: true,
		},
		"in kubectl output does not contain scanner manifests": {
			kubectlOutput:           true,
			expectScannerFilesExist: false,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			metaVals := charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(s.T()))
			metaVals.KubectlOutput = testCase.kubectlOutput

			loadedChart, err := s.image.LoadChart(SecuredClusterServicesChartPrefix, metaVals)
			s.Require().NoError(err)
			s.NotEmpty(loadedChart.Templates)

			var chartFiles []*chart.File
			chartFiles = append(chartFiles, loadedChart.Files...)
			chartFiles = append(chartFiles, loadedChart.Templates...)

			var foundScannerTpls []string
			for _, tpl := range chartFiles {
				if strings.Contains(tpl.Name, "scanner") {
					foundScannerTpls = append(foundScannerTpls, tpl.Name)
				}
			}

			// Release builds should not contain scanner files currently
			if testCase.expectScannerFilesExist {
				s.NotEmpty(foundScannerTpls, "Did not found any scanner manifests but expected them.")
			} else {
				s.Empty(foundScannerTpls, "Found unexpected scanner manifests %q in SecuredCluster loadedChart", foundScannerTpls)
			}
		})
	}

}
