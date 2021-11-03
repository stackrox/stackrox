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
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/cryptoutils/mocks"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const (
	endpoint = "localhost:8000"

	// Receiving trust info examples from a running cluster:
	// roxcurl /v1/tls-challenge?"challengeToken=h83_PGhSqS8OAvplb8asYMfPHy1JhVVMKcajYyKmrIU="
	// Copy trust-info and signature from the json response
	trustInfoExample = "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"
	signatureExample = "MEUCIDH0aciHWPf/edzSRvBZshIFsN9ihqDd9I4s3VSxnjMWAiEAuIvfnw1mEWcYiWyQO2RntligA/k7UR5+GyJs5UbJ3ss="
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

	t.clientCertDir, err = os.MkdirTemp("", "client-certs")
	t.Require().NoError(err)

	leafCert, err := ca.IssueCertForSubject(mtls.SensorSubject)
	t.Require().NoError(err)

	t.Require().NoError(os.WriteFile(filepath.Join(t.clientCertDir, "cert.pem"), leafCert.CertPEM, 0644))
	t.Require().NoError(os.WriteFile(filepath.Join(t.clientCertDir, "key.pem"), leafCert.KeyPEM, 0600))
	t.envIsolator.Setenv(mtls.CertFilePathEnvName, filepath.Join(t.clientCertDir, "cert.pem"))
	t.envIsolator.Setenv(mtls.KeyFileEnvName, filepath.Join(t.clientCertDir, "key.pem"))
}

func (t *ClientTestSuite) TearDownSuite() {
	_ = os.RemoveAll(t.clientCertDir)
	t.envIsolator.RestoreAll()
}

func (t *ClientTestSuite) newSelfSignedCertificate(commonName string) *tls.Certificate {
	req := csr.CertificateRequest{
		CN:         commonName,
		KeyRequest: csr.NewBasicKeyRequest(),
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
	t.Equal("LoadBalancer Certificate Authority", certs[0].Subject.CommonName)
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
