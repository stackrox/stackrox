package centralclient

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/centralsensor"
	"github.com/stackrox/stackrox/pkg/certgen"
	"github.com/stackrox/stackrox/pkg/cryptoutils/mocks"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const (
	endpoint = "localhost:8000"

	// Receiving trust info examples from a running cluster:
	// roxcurl /v1/tls-challenge?"challengeToken=h83_PGhSqS8OAvplb8asYMfPHy1JhVVMKcajYyKmrIU="
	// Copy trust-info and signature from the json response
	// Note that tests here are likely to start failing again some time in November 2022 due to cert expiration.
	// TODO(ROX-8661): Make these tests not fail after a year.
	trustInfoExample = "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"
	signatureExample = "MEYCIQDaJRmuxWGArjO4us5XVjukNZqQz78zAWydzBZISxXKfQIhAN47i+VSmyGVpI5WlzR5Tq4GN74l9vml0VWxyopsGtl4"
	// invalidSignature signature signed by a different private key
	invalidSignature = "MEUCIQDTYU+baqRR2RPy9Y50u5xc+ZrwrxCbqgHsgyf+QrjZQQIgJgqMmvRRvtgLU9O6WfzNifA1X8vwaBZ98CCniRH2pGs="

	// trustInfoUntrustedCentral trust info generated from another central installation that is not trusted by the test data
	trustInfoUntrustedCentral = "CtIEMIICTjCCAfSgAwIBAgIJANYUBtnEPMvRMAoGCCqGSM49BAMCMEcxJzAlBgNVBAMTHlN0YWNrUm94IENlcnRpZmljYXRlIEF1dGhvcml0eTEcMBoGA1UEBRMTNTkzMTk2NjM4NzcxMzkwNTgzMjAeFw0yMTEwMjEwOTAyMDBaFw0yMjEwMjExMDAyMDBaMFwxGDAWBgNVBAsMD0NFTlRSQUxfU0VSVklDRTEhMB8GA1UEAwwYQ0VOVFJBTF9TRVJWSUNFOiBDZW50cmFsMR0wGwYDVQQFExQxNTQyNTk2MjE1NjAyMDc3OTk4NTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABNeN6Vr6JzdqYuhbMYywuGzVxNLYmuiOt7vBd0n3y/0+hqhw57u9cRlVUqDYzrQgV5kWLqOG8x9eW+FGbyP4ZM6jgbMwgbAwDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMB0GA1UdDgQWBBQCPd9tI81J+WfhCi3tfZHw0vwPZTAfBgNVHSMEGDAWgBSsFJ+sB5YiXsxIwlyAOZk/z4aVSTAxBgNVHREEKjAoghBjZW50cmFsLnN0YWNrcm94ghRjZW50cmFsLnN0YWNrcm94LnN2YzAKBggqhkjOPQQDAgNIADBFAiEAtgK8ueDNBKtowtHSQl6+DdXJNiJZIyNteRqO2lK2LNkCIGeMhGX5gNli98NU26odZ+QrxWsLa39iK710jsVTj6nwCtYDMIIB0jCCAXigAwIBAgIUYDi0j/ypoh0u5w8FU70RzDHH9MMwCgYIKoZIzj0EAwIwRzEnMCUGA1UEAxMeU3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRwwGgYDVQQFExM1OTMxOTY2Mzg3NzEzOTA1ODMyMB4XDTIxMTAyMTA5NTcwMFoXDTI2MTAyMDA5NTcwMFowRzEnMCUGA1UEAxMeU3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRwwGgYDVQQFExM1OTMxOTY2Mzg3NzEzOTA1ODMyMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEASqFQTVprF72w5TH2C62JjnHRlA50n/xRgRCLCWmnSj8V8jgXc5wOpc8dbSLh1fn0cZ320j6F5erwQaloZc3GaNCMEAwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFKwUn6wHliJezEjCXIA5mT/PhpVJMAoGCCqGSM49BAMCA0gAMEUCIQDbiBkLqvuX6YC32zion11nYTO9p5eo3RVVFkvusgNAWQIgX/BADqhoAuGNXTO6qosJwwO40E/0bT5rtVjBNoN4XTASLGg4M19QR2hTcVM4T0F2cGxiOGFzWU1mUEh5MUpoVlZNS2Nhall5S21ySVU9GixGRm5sT2tqc29HcVJmZkYxczl0MUdJamNUYTBnMkN3eXo2UGp5b0NVUEpjPQ=="
	signatureUntrustedCentral = "MEUCIQDz2vnle9zrByV7KgwawvQkkXPNTMHxeAt2+hlLRch2QQIgFU+uu9w7LrjzuknVnZRq2ZzdmIbYVkzWYQkZhCH8kSQ="

	exampleChallengeToken = "h83_PGhSqS8OAvplb8asYMfPHy1JhVVMKcajYyKmrIU="
)

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite

	envIsolator   *envisolator.EnvIsolator
	clientCertDir string
	mockCtrl      *gomock.Controller
}

func (t *ClientTestSuite) SetupSuite() {
	t.envIsolator = envisolator.NewEnvIsolator(t.T())

	t.mockCtrl = gomock.NewController(t.T())

	cwd, err := os.Getwd()
	t.Require().NoError(err)
	t.envIsolator.Setenv(mtls.CAFileEnvName, filepath.Join(cwd, "testdata", "central-ca.pem"))

	// Generate a client certificate (this does not need to be related to the central CA from testdata).
	ca, err := certgen.GenerateCA()
	t.Require().NoError(err)

	t.clientCertDir = t.T().TempDir()

	leafCert, err := ca.IssueCertForSubject(mtls.SensorSubject)
	t.Require().NoError(err)

	t.Require().NoError(os.WriteFile(filepath.Join(t.clientCertDir, "cert.pem"), leafCert.CertPEM, 0644))
	t.Require().NoError(os.WriteFile(filepath.Join(t.clientCertDir, "key.pem"), leafCert.KeyPEM, 0600))
	t.envIsolator.Setenv(mtls.CertFilePathEnvName, filepath.Join(t.clientCertDir, "cert.pem"))
	t.envIsolator.Setenv(mtls.KeyFileEnvName, filepath.Join(t.clientCertDir, "key.pem"))
}

func (t *ClientTestSuite) TearDownSuite() {
	t.envIsolator.RestoreAll()
}

func (t *ClientTestSuite) newSelfSignedCertificate(commonName string) *tls.Certificate {
	req := csr.CertificateRequest{
		CN:         commonName,
		KeyRequest: csr.NewKeyRequest(),
		Hosts:      []string{"host"},
	}

	caCert, _, caKey, err := initca.New(&req)
	t.Require().NoError(err)
	cert, err := tls.X509KeyPair(caCert, caKey)
	t.Require().NoError(err)

	return &cert
}

func (t *ClientTestSuite) TestGetMetadata() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Equal(metadataRoute, r.URL.Path)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"version":       "3.0.51.x-47-g15440b8be2",
			"buildFlavor":   "development",
			"releaseBuild":  false,
			"licenseStatus": "VALID",
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	metadata, err := c.GetMetadata(context.Background())
	t.Require().NoError(err)

	t.Equal("3.0.51.x-47-g15440b8be2", metadata.GetVersion())
	t.Equal(v1.Metadata_LicenseStatus(4), metadata.GetLicenseStatus())
	t.False(metadata.GetReleaseBuild())
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_ErrorHandling() {
	testCases := map[string]struct {
		Name                 string
		ServerTLSCertificate *tls.Certificate
		Error                error
		TrustInfoSerialized  string
		TrustInfoSignature   string
	}{
		"Sensor connecting to Central with a different CA should fail": {
			ServerTLSCertificate: t.newSelfSignedCertificate("StackRox Certificate Authority"),
			Error:                errMismatchingCentralInstallation,
			TrustInfoSerialized:  trustInfoUntrustedCentral,
			TrustInfoSignature:   signatureUntrustedCentral,
		},
		"Sensor connecting to a peer with an untrusted certificate fails": {
			Error:               errAdditionalCANeeded,
			TrustInfoSerialized: trustInfoExample,
			TrustInfoSignature:  signatureExample,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func() {
			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Contains(r.URL.String(), "/v1/tls-challenge?challengeToken=")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"trustInfoSerialized": testCase.TrustInfoSerialized,
					"signature":           testCase.TrustInfoSignature,
				})
			}))

			if testCase.ServerTLSCertificate != nil {
				ts.TLS = &tls.Config{
					Certificates: []tls.Certificate{*testCase.ServerTLSCertificate},
				}
			}

			ts.StartTLS()
			defer ts.Close()

			c, err := NewClient(ts.URL)
			t.Require().NoError(err)

			mockNonceGenerator := mocks.NewMockNonceGenerator(t.mockCtrl)
			mockNonceGenerator.EXPECT().Nonce().Times(1).Return(exampleChallengeToken, nil)
			c.nonceGenerator = mockNonceGenerator

			_, err = c.GetTLSTrustedCerts(context.Background())
			if testCase.Error != nil {
				t.Require().ErrorIs(err, testCase.Error)
			} else {
				t.Require().NoError(err)
			}
		})
	}
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_GetCertificate() {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Contains(r.URL.String(), "/v1/tls-challenge?challengeToken=")

		sensorChallengeToken := r.URL.Query().Get(challengeTokenParamName)
		t.Assert().Equal(exampleChallengeToken, sensorChallengeToken)

		sensorChallengeTokenBytes, err := base64.URLEncoding.DecodeString(sensorChallengeToken)
		t.Require().NoError(err)
		t.Assert().Len(sensorChallengeTokenBytes, centralsensor.ChallengeTokenLength)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trustInfoSerialized": trustInfoExample,
			"signature":           signatureExample,
		})
	}))

	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{*t.newSelfSignedCertificate("StackRox Certificate Authority")},
	}

	ts.StartTLS()
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	mockNonceGenerator := mocks.NewMockNonceGenerator(t.mockCtrl)
	mockNonceGenerator.EXPECT().Nonce().Times(1).Return(exampleChallengeToken, nil)
	c.nonceGenerator = mockNonceGenerator

	certs, err := c.GetTLSTrustedCerts(context.Background())
	t.Require().NoError(err)

	t.Require().Len(certs, 1)
	t.Equal("Root LoadBalancer Certificate Authority", certs[0].Subject.CommonName)
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithSignatureSignedByAnotherPrivateKey_ShouldFail() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trustInfoSerialized": trustInfoExample,
			"signature":           invalidSignature,
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
	t.Require().ErrorIs(err, errInvalidTrustInfoSignature)
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithInvalidTrustInfo() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trustInfoSerialized": base64.StdEncoding.EncodeToString([]byte("Invalid trust info")),
			"signature":           signatureExample,
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
	t.True(errors.Is(err, io.ErrUnexpectedEOF))
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithInvalidSignature() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trustInfoSerialized": trustInfoExample,
			"signature":           base64.StdEncoding.EncodeToString([]byte("Invalid signature")),
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
	t.Require().ErrorIs(err, errInvalidTrustInfoSignature)
}

func (t *ClientTestSuite) TestGetTLSTrustedCertsWithDifferentSensorChallengeShouldFail() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"trustInfoSerialized": trustInfoExample, "signature": signatureExample})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	mockNonceGenerator := mocks.NewMockNonceGenerator(t.mockCtrl)
	mockNonceGenerator.EXPECT().Nonce().Times(1).Return("some_token", nil)
	c.nonceGenerator = mockNonceGenerator

	_, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
	t.Contains(err.Error(), fmt.Sprintf(`validating Central response failed: Sensor token "some_token" did not match received token %q`, exampleChallengeToken))
}

func (t *ClientTestSuite) TestNewClientReplacesProtocols() {
	// By default HTTPS will be prepended
	c, err := NewClient(endpoint)
	t.Require().NoError(err)
	t.Equal(fmt.Sprintf("https://%s", endpoint), c.endpoint.String())

	// HTTPS is accepted
	c, err = NewClient(fmt.Sprintf("https://%s", endpoint))
	t.Require().NoError(err)
	t.Equal(fmt.Sprintf("https://%s", endpoint), c.endpoint.String())

	// WebSockets are converted to HTTPS
	c, err = NewClient(fmt.Sprintf("wss://%s", endpoint))
	t.Require().NoError(err)
	t.Equal(fmt.Sprintf("https://%s", endpoint), c.endpoint.String())

	// HTTP is not accepted
	_, err = NewClient(fmt.Sprintf("http://%s", endpoint))
	t.Require().Error(err)
	t.Equal("creating client unsupported scheme http", err.Error())
}
