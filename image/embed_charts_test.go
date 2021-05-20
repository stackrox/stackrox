package image

import (
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/charts"
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

func (s *embedTestSuite) TestLoadChart() {
	chart, err := s.image.LoadChart(CentralServicesChartPrefix, charts.RHACSMetaValues())
	s.Require().NoError(err)
	s.Equal("stackrox-central-services", chart.Name())

	chart, err = s.image.LoadChart(SecuredClusterServicesChartPrefix, charts.RHACSMetaValues())
	s.Require().NoError(err)
	s.Equal("stackrox-secured-cluster-services", chart.Name())
}
