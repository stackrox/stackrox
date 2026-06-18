package clientconn

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const centralEndpoint = "central.stackrox:443"

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite
}

func (t *ClientTestSuite) SetupTest() {
	wd, _ := os.Getwd()
	testdata := path.Join(wd, "testdata")

	t.T().Setenv("ROX_MTLS_CA_FILE", path.Join(testdata, "ca.pem"))
}

func (t *ClientTestSuite) TestAddRootCA() {
	const certCount = 2
	cert := &x509.Certificate{Raw: []byte(`cert data`), SubjectKeyId: []byte(`SubjectKeyId1`), RawSubject: []byte(`RawSubject1`)}
	cert2 := &x509.Certificate{Raw: []byte(`cert data2`), SubjectKeyId: []byte(`SubjectKeyId2`), RawSubject: []byte(`RawSubject2`)}

	opts, err := OptionsForEndpoint(centralEndpoint, AddRootCAs(cert, cert2))
	t.Require().NoError(err)

	// read system root CAs
	sysCertPool, err := verifier.SystemCertPool()
	t.Require().NoError(err)

	addedCertsCount := len(opts.TLS.RootCAs.Subjects()) - len(sysCertPool.Subjects())
	t.Equalf(addedCertsCount, certCount, "Expected %d certificates being added", certCount)
}

func (t *ClientTestSuite) TestRootCA_WithNilCA_ShouldPanic() {
	t.Panics(func() {
		_, _ = OptionsForEndpoint(centralEndpoint, AddRootCAs(nil))
	})
}

func (t *ClientTestSuite) TestAuthenticatedHTTPTransport_WebSocket() {
	noopServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseTransport := httputil.DefaultTransport()

	testcases := []struct {
		name   string
		scheme string
		valid  bool
	}{
		{
			name:   "valid wss",
			scheme: "wss",
			valid:  true,
		},
		{
			name:   "invalid wss",
			scheme: "wss",
			valid:  false,
		},
		{
			name:   "valid ws",
			scheme: "ws",
			valid:  true,
		},
		{
			name:   "invalid ws",
			scheme: "ws",
			valid:  false,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func() {
			// Ensure the request's URL drops the WebSocket.
			baseTransport.Proxy = func(r *http.Request) (*url.URL, error) {
				if !testcase.valid {
					t.FailNow("Should not make it this far")
				}

				// http because TLS is disabled.
				t.Equal("http://central.stackrox.svc:443/hello/howdy?file=rhelv2%2Frepository-to-cpe.json&uuid=f81dbc6b-5899-433b-bc86-9127219a9d89", r.URL.String())

				// Forward traffic to the NO-OP Server
				return url.Parse(noopServer.URL)
			}

			host := testcase.scheme + "://central.stackrox.svc:443"
			// This is sorted by key.
			rawQuery := url.Values{
				"uuid": []string{"f81dbc6b-5899-433b-bc86-9127219a9d89"},
				"file": []string{"rhelv2/repository-to-cpe.json"},
			}.Encode()
			endpoint := (&url.URL{Path: "/hello/howdy", RawQuery: rawQuery}).String()
			if !testcase.valid {
				endpoint = (&url.URL{
					Scheme:   "https",
					Host:     host,
					Path:     "hello/howdy",
					RawQuery: rawQuery,
				}).String()
			}

			req, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if testcase.valid {
				t.NoError(err)
			} else {
				errEndpoint := `"https://` + testcase.scheme + `:%2F%2Fcentral.stackrox.svc:443/hello/howdy?file=rhelv2%2Frepository-to-cpe.json&uuid=f81dbc6b-5899-433b-bc86-9127219a9d89"`
				errString := `parse ` + errEndpoint + `: invalid port ":%2F%2Fcentral.stackrox.svc:443" after host`
				t.EqualError(err, errString)
				return
			}

			transport, err := AuthenticatedHTTPTransport(host, mtls.CentralSubject, baseTransport, UseInsecureNoTLS(true))
			t.Require().NoError(err)
			client := &http.Client{
				Transport: transport,
				Timeout:   0,
			}

			resp, err := client.Do(req)
			t.NoError(err)
			t.Equal(http.StatusOK, resp.StatusCode)
		})
	}
}

func writeCertToFiles(t testing.TB, certDir string, cert *tls.Certificate) {
	t.Helper()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyBytes, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	require.NoError(t, os.WriteFile(filepath.Join(certDir, "cert.pem"), certPEM, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(certDir, "key.pem"), keyPEM, 0600))
}

func (t *ClientTestSuite) TestGetClientCertificate_ReloadsFromDisk() {
	certDir := t.T().TempDir()
	t.T().Setenv(mtls.CertFilePathEnvName, filepath.Join(certDir, "cert.pem"))
	t.T().Setenv(mtls.KeyFileEnvName, filepath.Join(certDir, "key.pem"))

	cert1 := testutils.IssueSelfSignedCert(t.T(), "first-cert")
	writeCertToFiles(t.T(), certDir, &cert1)

	conf, err := TLSConfig(mtls.CentralSubject, TLSConfigOptions{
		UseClientCert:      MustUseClientCert,
		InsecureSkipVerify: true,
	})
	t.Require().NoError(err)
	t.Require().NotNil(conf.GetClientCertificate)

	got1, err := conf.GetClientCertificate(nil)
	t.Require().NoError(err)
	t.Equal(cert1.Certificate[0], got1.Certificate[0])

	cert2 := testutils.IssueSelfSignedCert(t.T(), "second-cert")
	writeCertToFiles(t.T(), certDir, &cert2)

	t.Require().Eventually(func() bool {
		got, err := conf.GetClientCertificate(nil)
		return err == nil && len(got.Certificate) > 0 &&
			bytes.Equal(got.Certificate[0], cert2.Certificate[0])
	}, 30*time.Second, 200*time.Millisecond, "expected rotated cert to be picked up by certwatch")
}
