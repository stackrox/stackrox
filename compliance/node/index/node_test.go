package index

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestNodeIndexerSuite(t *testing.T) {
	suite.Run(t, new(nodeIndexerSuite))
}

type nodeIndexerSuite struct {
	suite.Suite
}

func createConfig(server string) *NodeIndexerConfig {
	return &NodeIndexerConfig{
		DisableAPI:         true,
		Repo2CPEMappingURL: server,
		Timeout:            10 * time.Second,
	}
}

func createTestServer(t *testing.T) *httptest.Server {
	mappingData := `{"data":{"rhocp-4.16-for-rhel-9-x86_64-rpms":{"cpes":["cpe:/a:redhat:openshift:4.16::el9"]},"rhel-9-for-x86_64-baseos-eus-rpms__9_DOT_4":{"cpes":["cpe:/o:redhat:rhel_eus:9.4::baseos"]}}})`

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "") {
			w.WriteHeader(http.StatusNotFound)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("last-modified", "Mon, 02 Jan 2006 15:04:05 MST")
		_, err := w.Write([]byte(mappingData))
		assert.NoError(t, err)
	}))
	return s
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
	s.NoError(err)

	s.NotNil(layer)
	s.NoError(layer.Close())
}

func (s *nodeIndexerSuite) TestConstructLayerNoURI() {
	_, err := constructLayer(context.TODO(), layerDigest, "")
	s.ErrorContains(err, "no URI provided")
}

func (s *nodeIndexerSuite) TestConstructLayerIllegalDigest() {
	_, err := constructLayer(context.TODO(), "sha256:nodigest", s.T().TempDir())
	s.ErrorContains(err, "unable to decode digest as hex")
}

func (s *nodeIndexerSuite) TestRunRespositoryScanner() {
	cwd, err := os.Getwd()
	s.NoError(err)
	s.T().Setenv(mtls.CertFilePathEnvName, path.Join(cwd, "testdata", "certs", "cert.pem"))
	s.T().Setenv(mtls.KeyFileEnvName, path.Join(cwd, "testdata", "certs", "key.pem"))
	layer, err := createLayer("testdata")
	s.NoError(err)
	server := createTestServer(s.T())
	defer server.Close()
	c := createConfig(server.URL)

	repositories, err := runRepositoryScanner(context.TODO(), c, layer)
	s.NoError(err)

	s.Len(repositories, 2)
	s.NoError(layer.Close())
}

func (s *nodeIndexerSuite) TestRunRespositoryScannerAnyPath() {
	cwd, err := os.Getwd()
	s.NoError(err)
	s.T().Setenv(mtls.CertFilePathEnvName, path.Join(cwd, "testdata", "certs", "cert.pem"))
	s.T().Setenv(mtls.KeyFileEnvName, path.Join(cwd, "testdata", "certs", "key.pem"))
	layer, err := createLayer(s.T().TempDir())
	s.NoError(err)
	server := createTestServer(s.T())
	defer server.Close()
	c := createConfig(server.URL)

	repositories, err := runRepositoryScanner(context.TODO(), c, layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(repositories, 0)
	s.NoError(layer.Close())
}

func (s *nodeIndexerSuite) TestRunPackageScanner() {
	layer, err := createLayer("testdata")
	s.NoError(err)

	packages, err := runPackageScanner(context.TODO(), layer)
	s.NoError(err)

	s.Len(packages, 106)
	s.NoError(layer.Close())
}

func (s *nodeIndexerSuite) TestRunPackageScannerAnyPath() {
	layer, err := createLayer(s.T().TempDir())
	s.NoError(err)

	packages, err := runPackageScanner(context.TODO(), layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(packages, 0)
	s.NoError(layer.Close())
}

func (s *nodeIndexerSuite) TestIndexerE2E() {
	cwd, err := os.Getwd()
	s.NoError(err)
	s.T().Setenv(mtls.CertFilePathEnvName, path.Join(cwd, "testdata", "certs", "cert.pem"))
	s.T().Setenv(mtls.KeyFileEnvName, path.Join(cwd, "testdata", "certs", "key.pem"))
	testdir, err := filepath.Abs("testdata")
	s.NoError(err)
	s.T().Setenv("ROX_NODE_INDEX_HOST_PATH", testdir)
	srv := createTestServer(s.T())
	defer srv.Close()
	ni := NewNodeIndexer(createConfig(srv.URL))

	report, err := ni.IndexNode(context.TODO())
	s.NoError(err)

	s.NotNil(report)
	s.True(report.Success)
	s.Len(report.GetContents().GetPackages(), 106, "Expected number of installed packages differs")
	s.Len(report.GetContents().GetRepositories(), 2, "Expected number of discovered repositories differs")
}

func (s *nodeIndexerSuite) TestIndexerE2ENoPath() {
	err := os.Setenv("ROX_NODE_INDEX_HOST_PATH", "/notexisting")
	s.NoError(err)
	srv := createTestServer(s.T())
	defer srv.Close()
	ni := NewNodeIndexer(createConfig(srv.URL))

	report, err := ni.IndexNode(context.TODO())

	s.ErrorContains(err, "no such file or directory")
	s.Nil(report)
}
