package image

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	flavorUtils "github.com/stackrox/rox/pkg/images/defaults/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func init() {
	testutils.SetMainVersion(&testing.T{}, "3.0.55.0")
	testbuildinfo.SetForTest(&testing.T{})
}

func TestManager(t *testing.T) {
	suite.Run(t, new(embedTestSuite))
}

type embedTestSuite struct {
	suite.Suite

	envIsolator *envisolator.EnvIsolator
	image       *Image
}

func (s *embedTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	testutils.SetExampleVersion(s.T())
	s.image = GetDefaultImage()
}

func (s *embedTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
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
	testCases := map[string]defaults.ImageFlavor{
		"testFlavor": flavorUtils.MakeImageFlavorForTest(s.T()),
		"development": defaults.DevelopmentBuildImageFlavor(),
		"stackrox_io": defaults.StackRoxIOReleaseImageFlavor(),
		"rhacs": defaults.RHACSReleaseImageFlavor(),
	}

	for name, flavor := range testCases {
		s.Run(name, func() {
			chart, err := s.image.LoadChart(CentralServicesChartPrefix, charts.GetMetaValuesForFlavor(flavor))
			s.Require().NoError(err)
			s.Equal("stackrox-central-services", chart.Name())

			chart, err = s.image.LoadChart(SecuredClusterServicesChartPrefix, charts.GetMetaValuesForFlavor(flavor))
			s.Require().NoError(err)
			s.Equal("stackrox-secured-cluster-services", chart.Name())
		})
	}
}

func (s *embedTestSuite) TestSecuredClusterChartShouldIgnoreFeatureFlags() {
	metaVals := charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(s.T()))
	delete(metaVals, "FeatureFlags")

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
func (s *embedTestSuite) TestLoadSecuredClusterDoesNotContainScannerManifests() {
	s.envIsolator.Setenv(features.LocalImageScanning.Name(), "false")

	metaVals := charts.GetMetaValuesForFlavor(flavorUtils.MakeImageFlavorForTest(s.T()))
	chart, err := s.image.LoadChart(SecuredClusterServicesChartPrefix, metaVals)
	s.Require().NoError(err)
	s.Equal("stackrox-secured-cluster-services", chart.Name())
	s.NotEmpty(chart.Templates)

	var foundScannerTpls []string
	for _, tpl := range chart.Templates {
		if strings.Contains(tpl.Name, "scanner") {
			foundScannerTpls = append(foundScannerTpls, tpl.Name)
		}
	}

	s.Empty(foundScannerTpls, "Found unexpected scanner manifests %q in SecuredCluster chart", foundScannerTpls)
}
