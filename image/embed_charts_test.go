package image

import (
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func TestManager(t *testing.T) {
	suite.Run(t, new(embedTestSuite))
}

type embedTestSuite struct {
	suite.Suite

	envIsolator *envisolator.EnvIsolator
}

func (s *embedTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	testutils.SetExampleVersion(s.T())
}

func (s *embedTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *embedTestSuite) TestEmbedAllFiles() {
	err := filepath.WalkDir("templates", func(p string, d fs.DirEntry, err error) error {
		s.Require().NoError(err)

		_, statErr := fs.Stat(AssetFS, p)
		s.Require().NoError(statErr, "Could not find file or directory %q, to fix this add %q to the go embed directive",
			p, path.Join(path.Dir(p), "*"))
		return nil
	})
	s.Require().NoError(err)
}

func (s *embedTestSuite) TestChartTemplatesAvailable() {
	_, err := GetCentralServicesChartTemplate()
	s.Require().NoError(err, "failed to load central services chart")
	_, err = GetSecuredClusterServicesChartTemplate()
	s.Require().NoError(err, "failed to load secured cluster services chart")
}
