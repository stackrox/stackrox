package index

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestNodeIndexerSuite(t *testing.T) {
	suite.Run(t, new(nodeIndexerSuite))
}

type nodeIndexerSuite struct {
	suite.Suite
}

func createConfig(hostPath string, client *http.Client, mappingURL string) NodeIndexerConfig {
	return NodeIndexerConfig{
		HostPath:           hostPath,
		Client:             client,
		DisableAPI:         true,
		Repo2CPEMappingURL: mappingURL,
		Timeout:            10 * time.Second,
	}
}

func createTestServer(t *testing.T, tlsEnabled bool) *httptest.Server {
	mappingData := `{
	"data": {
		"rhocp-4.16-for-rhel-9-x86_64-rpms": {
			"cpes": ["cpe:/a:redhat:openshift:4.16::el9"]
		},
		"rhel-9-for-x86_64-baseos-eus-rpms__9_DOT_4": {
			"cpes": ["cpe:/o:redhat:rhel_eus:9.4::baseos"]
		}
	}
}`

	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "") {
			w.WriteHeader(http.StatusNotFound)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("last-modified", "Mon, 02 Jan 2006 15:04:05 MST")
		_, err := w.Write([]byte(mappingData))
		assert.NoError(t, err)
	}))
	if !tlsEnabled {
		s.Start()
	} else {
		serverCert, err := tls.LoadX509KeyPair(filepath.Join("testdata", "certs", "server-cert.pem"), filepath.Join("testdata", "certs", "server-key.pem"))
		caCert, err := os.ReadFile(filepath.Join("testdata", "certs", "ca-cert.pem"))
		require.NoError(t, err)
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		require.NoError(t, err)
		s.TLS = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs: caCertPool,
		}
		s.StartTLS()
	}

	return s
}

func createLayer(hostPath string) (*claircore.Layer, error) {
	layer, err := layer(context.Background(), layerDigest, hostPath)
	if err != nil {
		return nil, err
	}
	return layer, nil
}

func (s *nodeIndexerSuite) TestLayer() {
	testdir, err := filepath.Abs("testdata")
	s.NoError(err)

	layer, err := layer(context.Background(), layerDigest, testdir)
	s.NoError(err)

	s.NotNil(layer)
	s.NoError(layer.Close())
}

func (s *nodeIndexerSuite) TestLayerNoURI() {
	_, err := layer(context.Background(), layerDigest, "")
	s.ErrorContains(err, "no URI provided")
}

func (s *nodeIndexerSuite) TestLayerIllegalDigest() {
	_, err := layer(context.Background(), "sha256:nodigest", s.T().TempDir())
	s.ErrorContains(err, "unable to decode digest as hex")
}

func (s *nodeIndexerSuite) TestRunRepositoryScanner() {
	layer, err := createLayer("testdata")
	s.NoError(err)
	s.T().Cleanup(func() {
		s.NoError(layer.Close())
	})
	server := createTestServer(s.T(), false)
	s.T().Cleanup(server.Close)
	c := createConfig("testdata", server.Client(), server.URL)

	repositories, err := runRepositoryScanner(context.Background(), c, layer)
	s.NoError(err)

	s.Len(repositories, 2)
}

func (s *nodeIndexerSuite) TestRunRepositoryScannerAnyPath() {
	layer, err := createLayer(s.T().TempDir())
	s.NoError(err)
	s.T().Cleanup(func() {
		s.NoError(layer.Close())
	})
	server := createTestServer(s.T(), false)
	s.T().Cleanup(server.Close)
	c := createConfig("testdata", server.Client(), server.URL)

	repositories, err := runRepositoryScanner(context.Background(), c, layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(repositories, 0)
}

func (s *nodeIndexerSuite) TestRunPackageScanner() {
	layer, err := createLayer("testdata")
	s.NoError(err)
	s.T().Cleanup(func() {
		s.NoError(layer.Close())
	})

	packages, err := runPackageScanner(context.Background(), layer)
	s.NoError(err)

	s.Len(packages, 106)
}

func (s *nodeIndexerSuite) TestRunPackageScannerAnyPath() {
	layer, err := createLayer(s.T().TempDir())
	s.NoError(err)
	s.T().Cleanup(func() {
		s.NoError(layer.Close())
	})

	packages, err := runPackageScanner(context.Background(), layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(packages, 0)
}

func (s *nodeIndexerSuite) TestIndexerE2E() {
	s.T().Setenv(env.NodeIndexHostPath.EnvVar(), "testdata")
	s.T().Setenv(mtls.CertFilePathEnvName, filepath.Join("testdata", "certs", "client-cert.pem"))
	s.T().Setenv(mtls.KeyFileEnvName, filepath.Join("testdata", "certs", "client-key.pem"))
	server := createTestServer(s.T(), true)
	s.T().Cleanup(server.Close)
	cfg := DefaultNodeIndexerConfig
	cfg.Repo2CPEMappingURL = server.URL
	indexer := NewNodeIndexer(cfg)

	report, err := indexer.IndexNode(context.Background())
	s.NoError(err)

	s.NotNil(report)
	s.True(report.Success)
	s.Len(report.GetContents().GetPackages(), 106, "Expected number of installed packages differs")
	s.Len(report.GetContents().GetRepositories(), 2, "Expected number of discovered repositories differs")
}

func (s *nodeIndexerSuite) TestIndexerE2ENoPath() {
	s.T().Setenv(env.NodeIndexHostPath.EnvVar(), "doesnotexist")
	server := createTestServer(s.T(), false)
	s.T().Cleanup(server.Close)
	cfg := DefaultNodeIndexerConfig
	cfg.Client = server.Client()
	cfg.Repo2CPEMappingURL = server.URL
	ni := NewNodeIndexer(cfg)

	report, err := ni.IndexNode(context.Background())

	s.ErrorContains(err, "no such file or directory")
	s.Nil(report)
}
