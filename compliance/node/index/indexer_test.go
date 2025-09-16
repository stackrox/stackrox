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
	"github.com/stackrox/rox/pkg/mtls"
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
		Repo2CPEMappingURL: mappingURL,
		Timeout:            10 * time.Second,
	}
}

func (s *nodeIndexerSuite) createTestServer(tlsEnabled bool) *httptest.Server {
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

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "") {
			w.WriteHeader(http.StatusNotFound)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("last-modified", "Mon, 02 Jan 2006 15:04:05 MST")
		_, err := w.Write([]byte(mappingData))
		s.NoError(err)
	}))
	if !tlsEnabled {
		server.Start()
	} else {
		serverCert, err := tls.LoadX509KeyPair(
			filepath.Join("testdata", "certs", "server-cert.pem"),
			filepath.Join("testdata", "certs", "server-key.pem"),
		)
		s.Require().NoError(err)
		caCert, err := os.ReadFile(filepath.Join("testdata", "certs", "ca-cert.pem"))
		s.Require().NoError(err)
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		server.TLS = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caCertPool,
		}
		server.StartTLS()
	}

	s.T().Cleanup(server.Close)

	return server
}

func (s *nodeIndexerSuite) mustCreateLayer(hostPath string) *claircore.Layer {
	layer, err := layer(context.Background(), layerDigest, hostPath)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(layer.Close())
	})
	return layer
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
	layer := s.mustCreateLayer("testdata")
	server := s.createTestServer(false)
	c := createConfig("testdata", server.Client(), server.URL)

	repositories, err := runRepositoryScanner(context.Background(), c, layer)
	s.NoError(err)

	s.Len(repositories, 2)
}

func (s *nodeIndexerSuite) TestRunRepositoryScannerAnyPath() {
	layer := s.mustCreateLayer(s.T().TempDir())
	server := s.createTestServer(false)
	c := createConfig("testdata", server.Client(), server.URL)

	repositories, err := runRepositoryScanner(context.Background(), c, layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(repositories, 0)
}

func (s *nodeIndexerSuite) TestRunPackageScanner() {
	layer := s.mustCreateLayer("testdata")

	packages, err := runPackageScanner(context.Background(), layer)
	s.NoError(err)

	s.Len(packages, 106)
}

func (s *nodeIndexerSuite) TestRunPackageScannerAnyPath() {
	layer := s.mustCreateLayer(s.T().TempDir())

	packages, err := runPackageScanner(context.Background(), layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(packages, 0)
}

func (s *nodeIndexerSuite) TestBuildMappingURL() {
	tcs := map[string]struct {
		advertisedEndpointSetting string
		mappingURLSetting         string
		expectedURL               string
	}{
		"Empty": {
			advertisedEndpointSetting: "",
			mappingURLSetting:         "",
			expectedURL:               "https://sensor.stackrox.svc:443/scanner/definitions?file=repo2cpe",
		},
		"Host with port": {
			advertisedEndpointSetting: "example.com:8080",
			mappingURLSetting:         "",
			expectedURL:               "https://example.com:8080/scanner/definitions?file=repo2cpe",
		},
		"Host without port": {
			advertisedEndpointSetting: "sensor.rhacs.svc",
			mappingURLSetting:         "",
			expectedURL:               "https://sensor.rhacs.svc/scanner/definitions?file=repo2cpe",
		},
		"HTTP scheme": {
			advertisedEndpointSetting: "http://example.com",
			mappingURLSetting:         "",
			expectedURL:               "https://example.com/scanner/definitions?file=repo2cpe",
		},
		"Mapping setting provided": {
			advertisedEndpointSetting: "sensor.namespace.svc:443",
			mappingURLSetting:         "https://example.com/download",
			expectedURL:               "https://example.com/download",
		},
		"Mapping setting provided with no scheme and trailing slash": {
			advertisedEndpointSetting: "sensor.namespace.svc:443",
			mappingURLSetting:         "example.com/download/",
			expectedURL:               "https://example.com/download",
		},
	}
	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			s.T().Setenv("ROX_ADVERTISED_ENDPOINT", tc.advertisedEndpointSetting)
			s.T().Setenv("ROX_NODE_INDEX_MAPPING_URL", tc.mappingURLSetting)
			s.Equal(tc.expectedURL, buildMappingURL())
		})
	}
}

func (s *nodeIndexerSuite) TestIndexerE2E() {
	s.T().Setenv(mtls.CertFilePathEnvName, filepath.Join("testdata", "certs", "client-cert.pem"))
	s.T().Setenv(mtls.KeyFileEnvName, filepath.Join("testdata", "certs", "client-key.pem"))
	server := s.createTestServer(true)
	cfg := DefaultNodeIndexerConfig()
	cfg.HostPath = "testdata"
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
	server := s.createTestServer(false)
	cfg := DefaultNodeIndexerConfig()
	cfg.Client = server.Client()
	cfg.HostPath = "doesnotexist"
	cfg.Repo2CPEMappingURL = server.URL
	indexer := NewNodeIndexer(cfg)

	report, err := indexer.IndexNode(context.Background())

	s.ErrorContains(err, "no such file or directory")
	s.Nil(report)
}
