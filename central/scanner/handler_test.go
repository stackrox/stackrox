package scanner

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/pkg/x509utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(handlerTestSuite))
}

type handlerTestSuite struct {
	suite.Suite
}

func (s *handlerTestSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
	testutils.SetExampleVersion(s.T())
}

func (s *handlerTestSuite) TestGenerateScannerHTTPHandler() {
	s.T().Setenv(defaults.ImageFlavorEnvName, defaults.ImageFlavorNameDevelopmentBuild)
	params := apiparams.Scanner{ClusterType: storage.ClusterType_KUBERNETES_CLUSTER.String(), ScannerImage: "docker.io/stackrox/scanner:latest"}
	_ = s.callServer(params)
}

func (s *handlerTestSuite) callServer(params apiparams.Scanner) *zip.Reader {
	server := httptest.NewServer(Handler())
	defer server.Close()
	marshaledJSON, err := json.Marshal(params)
	s.Require().NoError(err)

	resp, err := http.Post(server.URL, "application/json", bytes.NewReader(marshaledJSON))
	s.Require().NoError(err)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Assert().NotEmpty(body)

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	s.Assert().NoError(err)
	return zipReader
}

func (s *handlerTestSuite) TestCertificateSAN() {
	type testCase struct {
		name          string
		namespace     string
		scannerSANs   []string
		scannerDBSANs []string
	}

	cases := []testCase{
		{
			name:          "no namespace",
			scannerSANs:   []string{"scanner.stackrox", "scanner.stackrox.svc"},
			scannerDBSANs: []string{"scanner-db.stackrox", "scanner-db.stackrox.svc"},
		},
		{
			name:          "stackrox namespace",
			namespace:     "stackrox",
			scannerSANs:   []string{"scanner.stackrox", "scanner.stackrox.svc"},
			scannerDBSANs: []string{"scanner-db.stackrox", "scanner-db.stackrox.svc"},
		},
		{
			name:          "custom namespace",
			namespace:     "custom",
			scannerSANs:   []string{"scanner.stackrox", "scanner.stackrox.svc", "scanner.custom", "scanner.custom.svc"},
			scannerDBSANs: []string{"scanner-db.stackrox", "scanner-db.stackrox.svc", "scanner-db.custom", "scanner-db.custom.svc"},
		},
	}

	for _, tc := range cases {
		s.T().Run(tc.name, func(t *testing.T) {
			if tc.namespace != "" {
				t.Setenv(env.Namespace.EnvVar(), tc.namespace)
			}
			params := apiparams.Scanner{ClusterType: storage.ClusterType_KUBERNETES_CLUSTER.String(), ScannerImage: "docker.io/stackrox/scanner:latest"}
			reader := s.callServer(params)
			filename := "scanner/02-scanner-03-tls-secret.yaml"
			//#nosec G101 -- This is a false positive
			files := reader.File
			idx := slices.IndexFunc(files, func(f *zip.File) bool {
				return f.Name == filename
			})
			require.GreaterOrEqual(t, idx, 0, "%s file not found", filename)
			tlsSecretFile, err := files[idx].Open()
			require.NoError(t, err)
			defer func() {
				_ = tlsSecretFile.Close()
			}()
			content2, err := io.ReadAll(tlsSecretFile)
			require.NoError(t, err)
			content := content2
			decoder := yaml.NewDecoder(bytes.NewReader(content))
			var doc map[string]interface{}
			for decoder.Decode(&doc) == nil {
				tlsSecret := &corev1.Secret{}
				err := runtime.DefaultUnstructuredConverter.FromUnstructured(doc, tlsSecret)
				require.NoError(t, err)
				switch tlsSecret.Name {
				case "scanner-tls":
					assertSANs(t, tlsSecret, tc.scannerSANs)
				case "scanner-db-tls":
					assertSANs(t, tlsSecret, tc.scannerDBSANs)
				}
			}
		})
	}
}

func assertSANs(t *testing.T, tlsSecret *corev1.Secret, sans []string) {
	cert, err := tls.X509KeyPair([]byte(tlsSecret.StringData["cert.pem"]), []byte(tlsSecret.StringData["key.pem"]))
	require.NoError(t, err)
	chain, err := x509utils.ParseCertificateChain(cert.Certificate)
	require.NoError(t, err)
	for _, san := range sans {
		require.Contains(t, chain[0].DNSNames, san)
	}
}
