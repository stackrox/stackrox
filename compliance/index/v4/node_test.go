package v4

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/suite"
)

func TestNodeIndexerSuite(t *testing.T) {
	suite.Run(t, new(nodeIndexerSuite))
}

type nodeIndexerSuite struct {
	suite.Suite
}

func createLayer(path string) (layer *claircore.Layer, e error) {
	testdir, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	layer, err = constructLayer(context.TODO(), layerDigest, testdir)
	if err != nil {
		return nil, err
	}
	return layer, nil
}

func (s *nodeIndexerSuite) TestConstructLayer() {
	testdir, err := filepath.Abs("testdata")
	s.NoError(err)

	layer, err := constructLayer(context.TODO(), layerDigest, testdir)
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()

	s.NoError(err)
	s.NotNil(layer)
}

func (s *nodeIndexerSuite) TestConstructLayerNoURI() {
	layer, err := constructLayer(context.TODO(), layerDigest, "")
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()
	s.ErrorContains(err, "no URI provided")
}

func (s *nodeIndexerSuite) TestConstructLayerIllegalDigest() {
	layer, err := constructLayer(context.TODO(), "sha256:nodigest", s.T().TempDir())
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()
	s.ErrorContains(err, "unable to decode digest as hex")
}

func (s *nodeIndexerSuite) TestRunRespositoryScanner() {
	layer, err := createLayer("testdata")
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()
	s.NoError(err)

	repositories, err := runRepositoryScanner(context.TODO(), layer)
	s.NoError(err)

	s.Len(repositories, 2)
}

func (s *nodeIndexerSuite) TestRunRespositoryScannerAnyPath() {
	layer, err := createLayer(s.T().TempDir())
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()
	s.NoError(err)

	repositories, err := runRepositoryScanner(context.TODO(), layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(repositories, 0)
}

func (s *nodeIndexerSuite) TestRunPackageScanner() {
	layer, err := createLayer("testdata")
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()
	s.NoError(err)

	packages, err := runPackageScanner(context.TODO(), layer)
	s.NoError(err)

	s.Len(packages, 106)
}

func (s *nodeIndexerSuite) TestRunPackageScannerAnyPath() {
	layer, err := createLayer(s.T().TempDir())
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()
	s.NoError(err)

	packages, err := runPackageScanner(context.TODO(), layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(packages, 0)
}

func (s *nodeIndexerSuite) TestIndexerE2E() {
	testdir, err := filepath.Abs("testdata")
	s.NoError(err)
	err = os.Setenv("ROX_NODE_SCANNING_V4_HOST_PATH", testdir)
	s.NoError(err)
	ni := NewNodeIndexer()

	report, err := ni.IndexNode(context.TODO())
	s.NoError(err)

	s.NotNil(report)
	s.True(report.Success)
	s.Len(report.GetContents().GetPackages(), 106, "Expected number of installed packages differs")
	s.Len(report.GetContents().GetRepositories(), 2, "Expected number of discovered repositories differs")
}

func (s *nodeIndexerSuite) TestIndexerE2ENoPath() {
	err := os.Setenv("ROX_NODE_SCANNING_V4_HOST_PATH", "/notexisting")
	s.NoError(err)
	ni := NewNodeIndexer()

	report, err := ni.IndexNode(context.TODO())

	s.ErrorContains(err, "no such file or directory")
	s.Nil(report)
}
