package index

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types"
	"github.com/stackrox/rox/pkg/certgen"
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
		ca, err := certgen.GenerateCA()
		s.Require().NoError(err)

		serverCertIssued, err := ca.IssueCertForSubject(mtls.SensorSubject)
		s.Require().NoError(err)
		serverCert, err := tls.X509KeyPair(serverCertIssued.CertPEM, serverCertIssued.KeyPEM)
		s.Require().NoError(err)

		clientCertIssued, err := ca.IssueCertForSubject(mtls.SensorSubject)
		s.Require().NoError(err)

		certDir := s.T().TempDir()
		s.Require().NoError(os.WriteFile(filepath.Join(certDir, "ca.pem"), ca.CertPEM(), 0644))
		s.Require().NoError(os.WriteFile(filepath.Join(certDir, "client-cert.pem"), clientCertIssued.CertPEM, 0600))
		s.Require().NoError(os.WriteFile(filepath.Join(certDir, "client-key.pem"), clientCertIssued.KeyPEM, 0600))
		s.T().Setenv(mtls.CAFileEnvName, filepath.Join(certDir, "ca.pem"))
		s.T().Setenv(mtls.CertFilePathEnvName, filepath.Join(certDir, "client-cert.pem"))
		s.T().Setenv(mtls.KeyFileEnvName, filepath.Join(certDir, "client-key.pem"))

		server.TLS = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    ca.CertPool(),
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
	s.ErrorContains(err, "host path is empty")
}

func (s *nodeIndexerSuite) TestLayerUsesFileURI() {
	layer, err := layer(context.Background(), layerDigest, "testdata")
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(layer.Close())
	})

	parsedURI, err := url.Parse(layer.URI)
	s.Require().NoError(err)
	s.Equal("file", parsedURI.Scheme)

	expectedPath, err := filepath.Abs("testdata")
	s.Require().NoError(err)
	s.Equal(filepath.ToSlash(expectedPath), parsedURI.Path)
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

	packages, err := runPackageScanner(context.Background(), rhcosPackageDB, layer)
	s.NoError(err)

	s.Len(packages, 106)
}

func (s *nodeIndexerSuite) TestRunPackageScannerWithUnmatchedFilter() {
	layer := s.mustCreateLayer("testdata")

	packages, err := runPackageScanner(context.Background(), "invalidPackageDB", layer)
	s.NoError(err)

	// All packages are filtered out.
	s.Len(packages, 0)
}

func (s *nodeIndexerSuite) TestRunPackageScannerAnyPath() {
	layer := s.mustCreateLayer(s.T().TempDir())

	packages, err := runPackageScanner(context.Background(), rhcosPackageDB, layer)
	s.NoError(err)

	// The scanner must not error out, but produce 0 results
	s.Len(packages, 0)
}

func (s *nodeIndexerSuite) TestBuildMappingURL() {
	tcs := map[string]struct {
		sensorEndpointSetting     string
		advertisedEndpointSetting string
		useLegacyAdvertised       bool
		mappingURLSetting         string
		expectedURL               string
	}{
		"Empty": {
			sensorEndpointSetting: "",
			mappingURLSetting:     "",
			expectedURL:           "https://sensor.stackrox.svc:443/scanner/definitions?file=repo2cpe",
		},
		"Host with port": {
			sensorEndpointSetting: "example.com:8080",
			mappingURLSetting:     "",
			expectedURL:           "https://example.com:8080/scanner/definitions?file=repo2cpe",
		},
		"Host without port": {
			sensorEndpointSetting: "sensor.rhacs.svc",
			mappingURLSetting:     "",
			expectedURL:           "https://sensor.rhacs.svc/scanner/definitions?file=repo2cpe",
		},
		"HTTP scheme": {
			sensorEndpointSetting: "http://example.com",
			mappingURLSetting:     "",
			expectedURL:           "https://example.com/scanner/definitions?file=repo2cpe",
		},
		"Legacy advertised endpoint only": {
			advertisedEndpointSetting: "sensor.legacy.svc:443",
			useLegacyAdvertised:       true,
			mappingURLSetting:         "",
			expectedURL:               "https://sensor.legacy.svc:443/scanner/definitions?file=repo2cpe",
		},
		"Mapping setting provided": {
			sensorEndpointSetting: "sensor.namespace.svc:443",
			mappingURLSetting:     "https://example.com/download",
			expectedURL:           "https://example.com/download",
		},
		"Mapping setting provided with no scheme and trailing slash": {
			sensorEndpointSetting: "sensor.namespace.svc:443",
			mappingURLSetting:     "example.com/download/",
			expectedURL:           "https://example.com/download",
		},
	}
	for name, tc := range tcs {
		s.T().Run(name, func(t *testing.T) {
			s.T().Setenv("ROX_SENSOR_ENDPOINT", tc.sensorEndpointSetting)
			if tc.useLegacyAdvertised {
				s.T().Setenv("ROX_ADVERTISED_ENDPOINT", tc.advertisedEndpointSetting)
			} else {
				s.T().Setenv("ROX_ADVERTISED_ENDPOINT", "")
			}
			s.T().Setenv("ROX_NODE_INDEX_MAPPING_URL", tc.mappingURLSetting)
			s.Equal(tc.expectedURL, buildMappingURL())
		})
	}
}

func (s *nodeIndexerSuite) TestExtractHostname() {
	tcs := map[string]struct {
		input     string
		expected  string
		wantError bool
	}{
		"full URL":             {"https://sensor.stackrox.svc:443/scanner/definitions?file=repo2cpe", "sensor.stackrox.svc", false},
		"URL without port":     {"https://sensor.stackrox.svc/scanner/definitions", "sensor.stackrox.svc", false},
		"URL without path":     {"https://example.com", "example.com", false},
		"URL with port":        {"https://example.com:8080", "example.com", false},
		"IP address with port": {"https://10.96.0.1:443/path", "10.96.0.1", false},
		"empty string":         {"", "", false},
		"just hostname":        {"sensor.stackrox.svc", "sensor.stackrox.svc", false},
		"hostname with port":   {"sensor.stackrox.svc:443", "sensor.stackrox.svc", false},
		"invalid URL":          {"https://[invalid", "", true},
	}
	for name, tc := range tcs {
		s.Run(name, func() {
			got, err := extractHostname(tc.input)
			if tc.wantError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(tc.expected, got)
			}
		})
	}
}

func (s *nodeIndexerSuite) TestIndexerE2E() {
	server := s.createTestServer(true)

	s.Run("override Repo2CPEMappingURL", func() {
		cfg := DefaultNodeIndexerConfig()
		cfg.HostPath = "testdata"
		cfg.Repo2CPEMappingURL = server.URL
		cfg.PackageDBFilter = rhcosPackageDB

		report, err := NewNodeIndexer(cfg).IndexNode(context.Background())
		s.NoError(err)
		s.NotNil(report)
		s.True(report.GetSuccess())
		s.Len(report.GetContents().GetPackages(), 106)
		s.Len(report.GetContents().GetRepositories(), 2)
	})

	s.Run("explicit ROX_NODE_INDEX_MAPPING_URL", func() {
		mappingURL := server.URL + "/scanner/definitions?file=repo2cpe"
		s.T().Setenv("ROX_NODE_INDEX_MAPPING_URL", mappingURL)
		s.T().Setenv("ROX_ADVERTISED_ENDPOINT", "ignored.example.svc:9999")

		cfg := DefaultNodeIndexerConfig()
		s.Equal(buildMappingURL(), cfg.Repo2CPEMappingURL)
		s.Nil(cfg.Client)
		cfg.HostPath = "testdata"
		cfg.PackageDBFilter = rhcosPackageDB

		report, err := NewNodeIndexer(cfg).IndexNode(context.Background())
		s.NoError(err)
		s.NotNil(report)
		s.True(report.GetSuccess())
		s.Len(report.GetContents().GetPackages(), 106)
		s.Len(report.GetContents().GetRepositories(), 2)
	})
}

func (s *nodeIndexerSuite) TestIndexerE2ESeparateOSReleasePath() {
	// Use a plain HTTP mapping server + explicit client so this test does not
	// depend on the process-wide mTLS certwatch state left by other suite tests.
	server := s.createTestServer(false)
	cfg := DefaultNodeIndexerConfig()
	cfg.HostPath = "testdata"
	cfg.OSReleasePath = "testdata-rhcos"
	cfg.Repo2CPEMappingURL = server.URL
	cfg.Client = server.Client()
	cfg.PackageDBFilter = rhcosPackageDB
	indexer := NewNodeIndexer(cfg)

	report, err := indexer.IndexNode(context.Background())
	s.NoError(err)

	s.NotNil(report)
	s.True(report.GetSuccess())
	// 106 RPM packages + 2 rhcos packages (binary + source)
	s.Len(report.GetContents().GetPackages(), 108, "Expected 106 RPM + 2 rhcos packages")
	// 2 RPM repositories + 1 rhcos repository
	s.Len(report.GetContents().GetRepositories(), 3, "Expected 2 RPM + 1 rhcos repositories")

	var hasRHCOS bool
	for _, pkg := range report.GetContents().GetPackages() {
		if pkg.GetName() == "openshift/ose-rhel-coreos-9" {
			hasRHCOS = true
			s.Equal("9.6.20260324-0", pkg.GetVersion())
			break
		}
	}
	s.True(hasRHCOS, "Expected ose-rhel-coreos package from labels.json in report")
}

func (s *nodeIndexerSuite) TestIndexerE2ENoPath() {
	server := s.createTestServer(false)
	cfg := DefaultNodeIndexerConfig()
	cfg.Client = server.Client()
	cfg.HostPath = "doesnotexist"
	cfg.Repo2CPEMappingURL = server.URL
	cfg.PackageDBFilter = rhcosPackageDB
	indexer := NewNodeIndexer(cfg)

	report, err := indexer.IndexNode(context.Background())

	s.ErrorContains(err, "no such file or directory")
	s.ErrorIs(err, os.ErrNotExist)
	s.Nil(report)
}

func (s *nodeIndexerSuite) TestParseOSRelease() {
	osRel, err := parseOSRelease(context.Background(), "testdata-rhcos")
	s.Require().NoError(err)

	s.Equal("coreos", osRel["VARIANT_ID"])
	s.Equal("9.6.20260324-0", osRel["VERSION"])
	s.Equal("9.6", osRel["VERSION_ID"])
	s.Equal("4.21", osRel["OPENSHIFT_VERSION"])
}

func (s *nodeIndexerSuite) TestParseOSReleaseNotFound() {
	_, err := parseOSRelease(context.Background(), s.T().TempDir())
	s.ErrorContains(err, "os-release not found")
}

func (s *nodeIndexerSuite) TestOSReleaseInvalidVersion() {
	tmpDir := s.T().TempDir()
	etcDir := filepath.Join(tmpDir, "etc")
	s.Require().NoError(os.MkdirAll(etcDir, 0755))

	osReleaseContent := `VARIANT_ID=coreos
VERSION=invalid-version-format
VERSION_ID=9.6
OPENSHIFT_VERSION=4.21
`
	s.Require().NoError(os.WriteFile(filepath.Join(etcDir, "os-release"), []byte(osReleaseContent), 0644))

	_, err := osRelease(context.Background(), tmpDir)
	s.ErrorContains(err, "failed to parse RHCOS version")
}

func (s *nodeIndexerSuite) TestAddRHCOSPackageToReport() {
	rel, err := osRelease(context.Background(), "testdata-rhcos")
	s.Require().NoError(err)

	report := &claircore.IndexReport{
		Packages:     make(map[string]*claircore.Package),
		Repositories: make(map[string]*claircore.Repository),
		Environments: make(map[string][]*claircore.Environment),
	}

	addRHCOS(rel, buildLabels{}, "x86_64", report)

	s.Len(report.Packages, 2)
	s.Len(report.Repositories, 1)
	s.Len(report.Environments, 1)

	var binPkg *claircore.Package
	for _, p := range report.Packages {
		if p.Kind == types.BinaryPackage && p.Name == "rhcos" {
			binPkg = p
			break
		}
	}
	s.Require().NotNil(binPkg)
	s.Equal("rhcos", binPkg.Name)
	s.Equal("9.6.20260324-0", binPkg.Version)
	s.Equal("x86_64", binPkg.Arch)
	s.Equal("rhcc", binPkg.RepositoryHint)

	var repo *claircore.Repository
	for _, r := range report.Repositories {
		repo = r
		break
	}
	s.Require().NotNil(repo)
	s.Equal("rhcc-container-repository", repo.Key)
	s.Contains(repo.Name, "cpe:")
	s.Contains(repo.Name, "openshift")
	s.Contains(repo.Name, "4.21")
}

func (s *nodeIndexerSuite) TestAddRHCOSPackageFromLabels() {
	rel, err := osRelease(context.Background(), "testdata-rhcos")
	s.Require().NoError(err)

	labels, path, err := parseLabelsJSON("testdata-rhcos")
	s.Require().NoError(err)
	s.Contains(path, "labels.json")
	s.Equal("openshift/ose-rhel-coreos-9", labels.Name)

	report := &claircore.IndexReport{
		Packages:     make(map[string]*claircore.Package),
		Repositories: make(map[string]*claircore.Repository),
		Environments: make(map[string][]*claircore.Environment),
	}

	addRHCOS(rel, labels, "", report)

	var binPkg *claircore.Package
	for _, p := range report.Packages {
		if p.Kind == types.BinaryPackage {
			binPkg = p
			break
		}
	}
	s.Require().NotNil(binPkg)
	s.Equal("openshift/ose-rhel-coreos-9", binPkg.Name)
	s.Equal("9.6.20260324-0", binPkg.Version, "version still comes from os-release")
	s.Equal("x86_64", binPkg.Arch)

	var repo *claircore.Repository
	for _, r := range report.Repositories {
		repo = r
		break
	}
	s.Require().NotNil(repo)
	s.Contains(repo.Name, "openshift")
	s.Contains(repo.Name, "4.21")
}

func (s *nodeIndexerSuite) TestValidateOSRelease() {
	cases := map[string]struct {
		osRel         map[string]string
		expectError   string
		expectNoError bool
	}{
		"valid": {
			osRel: map[string]string{
				"VARIANT_ID":        "coreos",
				"VERSION":           "9.6.20260324-0",
				"VERSION_ID":        "9.6",
				"OPENSHIFT_VERSION": "4.21",
			},
			expectNoError: true,
		},
		"missing VERSION": {
			osRel: map[string]string{
				"VARIANT_ID":        "coreos",
				"VERSION_ID":        "9.6",
				"OPENSHIFT_VERSION": "4.21",
			},
			expectError: "VERSION not found",
		},
		"missing OPENSHIFT_VERSION": {
			osRel: map[string]string{
				"VARIANT_ID": "coreos",
				"VERSION":    "9.6.20260324-0",
				"VERSION_ID": "9.6",
			},
			expectError: "OPENSHIFT_VERSION not found",
		},
		"missing VERSION_ID": {
			osRel: map[string]string{
				"VARIANT_ID":        "coreos",
				"VERSION":           "9.6.20260324-0",
				"OPENSHIFT_VERSION": "4.21",
			},
			expectError: "VERSION_ID not found",
		},
		"not RHCOS": {
			osRel: map[string]string{
				"VARIANT_ID":        "server",
				"VERSION":           "9.6",
				"VERSION_ID":        "9.6",
				"OPENSHIFT_VERSION": "4.21",
			},
			expectError: "not RHCOS",
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			err := validateOSRelease(tc.osRel)
			if tc.expectNoError {
				s.NoError(err)
			} else {
				s.ErrorContains(err, tc.expectError)
			}
		})
	}
}

func (s *nodeIndexerSuite) TestExtractArch() {
	cases := map[string]struct {
		report   *claircore.IndexReport
		pkgs     []*claircore.Package
		expected string
	}{
		"from distribution": {
			report:   &claircore.IndexReport{Distributions: map[string]*claircore.Distribution{"1": {Arch: "x86_64"}}},
			pkgs:     nil,
			expected: "x86_64",
		},
		"from packages when no distribution": {
			report:   &claircore.IndexReport{},
			pkgs:     []*claircore.Package{{Arch: "aarch64"}},
			expected: "aarch64",
		},
		"skips noarch distribution": {
			report:   &claircore.IndexReport{Distributions: map[string]*claircore.Distribution{"1": {Arch: "noarch"}}},
			pkgs:     []*claircore.Package{{Arch: "x86_64"}},
			expected: "x86_64",
		},
		"skips noarch packages": {
			report:   &claircore.IndexReport{},
			pkgs:     []*claircore.Package{{Arch: "noarch"}, {Arch: "x86_64"}},
			expected: "x86_64",
		},
		"empty when all noarch": {
			report:   &claircore.IndexReport{Distributions: map[string]*claircore.Distribution{"1": {Arch: "noarch"}}},
			pkgs:     []*claircore.Package{{Arch: "noarch"}},
			expected: "",
		},
		"empty when no data": {
			report:   &claircore.IndexReport{},
			pkgs:     nil,
			expected: "",
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			got := extractArch(tc.report, tc.pkgs)
			s.Equal(tc.expected, got)
		})
	}
}
