package centralclient

import (
	"context"
	"crypto/tls"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	cTLS "github.com/google/certificate-transparency-go/tls"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/cryptoutils/mocks"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	endpoint = "localhost:8000"

	// invalidSignature signature signed by a different private key
	invalidSignature = "MEUCIQDTYU+baqRR2RPy9Y50u5xc+ZrwrxCbqgHsgyf+QrjZQQIgJgqMmvRRvtgLU9O6WfzNifA1X8vwaBZ98CCniRH2pGs="

	// trustInfoUntrustedCentral trust info generated from another central installation that is not trusted by the test data
	trustInfoUntrustedCentral = "CtIEMIICTjCCAfSgAwIBAgIJANYUBtnEPMvRMAoGCCqGSM49BAMCMEcxJzAlBgNVBAMTHlN0YWNrUm94IENlcnRpZmljYXRlIEF1dGhvcml0eTEcMBoGA1UEBRMTNTkzMTk2NjM4NzcxMzkwNTgzMjAeFw0yMTEwMjEwOTAyMDBaFw0yMjEwMjExMDAyMDBaMFwxGDAWBgNVBAsMD0NFTlRSQUxfU0VSVklDRTEhMB8GA1UEAwwYQ0VOVFJBTF9TRVJWSUNFOiBDZW50cmFsMR0wGwYDVQQFExQxNTQyNTk2MjE1NjAyMDc3OTk4NTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABNeN6Vr6JzdqYuhbMYywuGzVxNLYmuiOt7vBd0n3y/0+hqhw57u9cRlVUqDYzrQgV5kWLqOG8x9eW+FGbyP4ZM6jgbMwgbAwDgYDVR0PAQH/BAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMB0GA1UdDgQWBBQCPd9tI81J+WfhCi3tfZHw0vwPZTAfBgNVHSMEGDAWgBSsFJ+sB5YiXsxIwlyAOZk/z4aVSTAxBgNVHREEKjAoghBjZW50cmFsLnN0YWNrcm94ghRjZW50cmFsLnN0YWNrcm94LnN2YzAKBggqhkjOPQQDAgNIADBFAiEAtgK8ueDNBKtowtHSQl6+DdXJNiJZIyNteRqO2lK2LNkCIGeMhGX5gNli98NU26odZ+QrxWsLa39iK710jsVTj6nwCtYDMIIB0jCCAXigAwIBAgIUYDi0j/ypoh0u5w8FU70RzDHH9MMwCgYIKoZIzj0EAwIwRzEnMCUGA1UEAxMeU3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRwwGgYDVQQFExM1OTMxOTY2Mzg3NzEzOTA1ODMyMB4XDTIxMTAyMTA5NTcwMFoXDTI2MTAyMDA5NTcwMFowRzEnMCUGA1UEAxMeU3RhY2tSb3ggQ2VydGlmaWNhdGUgQXV0aG9yaXR5MRwwGgYDVQQFExM1OTMxOTY2Mzg3NzEzOTA1ODMyMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEASqFQTVprF72w5TH2C62JjnHRlA50n/xRgRCLCWmnSj8V8jgXc5wOpc8dbSLh1fn0cZ320j6F5erwQaloZc3GaNCMEAwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFKwUn6wHliJezEjCXIA5mT/PhpVJMAoGCCqGSM49BAMCA0gAMEUCIQDbiBkLqvuX6YC32zion11nYTO9p5eo3RVVFkvusgNAWQIgX/BADqhoAuGNXTO6qosJwwO40E/0bT5rtVjBNoN4XTASLGg4M19QR2hTcVM4T0F2cGxiOGFzWU1mUEh5MUpoVlZNS2Nhall5S21ySVU9GixGRm5sT2tqc29HcVJmZkYxczl0MUdJamNUYTBnMkN3eXo2UGp5b0NVUEpjPQ=="
	signatureUntrustedCentral = "MEUCIQDz2vnle9zrByV7KgwawvQkkXPNTMHxeAt2+hlLRch2QQIgFU+uu9w7LrjzuknVnZRq2ZzdmIbYVkzWYQkZhCH8kSQ="

	// To create new data you can do it manually via roxcurl, or use ./update-testdata.sh
	//#nosec G101 -- This is a false positive
	exampleChallengeToken = "h83_PGhSqS8OAvplb8asYMfPHy1JhVVMKcajYyKmrIU="
)

//go:embed testdata/*
var testdata embed.FS

func TestClient(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite

	clientCertDir    string
	mockCtrl         *gomock.Controller
	trustInfoExample string
	signatureExample string
}

func (t *ClientTestSuite) SetupSuite() {
	t.mockCtrl = gomock.NewController(t.T())

	cwd, err := os.Getwd()
	t.Require().NoError(err)
	t.T().Setenv(mtls.CAFileEnvName, filepath.Join(cwd, "testdata", "central", "ca.pem"))

	// Generate a client certificate (this does not need to be related to the central CA from testdata).
	ca, err := certgen.GenerateCA()
	t.Require().NoError(err)

	t.clientCertDir = t.T().TempDir()

	leafCert, err := ca.IssueCertForSubject(mtls.SensorSubject)
	t.Require().NoError(err)

	t.Require().NoError(os.WriteFile(filepath.Join(t.clientCertDir, "cert.pem"), leafCert.CertPEM, 0644))
	t.Require().NoError(os.WriteFile(filepath.Join(t.clientCertDir, "key.pem"), leafCert.KeyPEM, 0600))
	t.T().Setenv(mtls.CertFilePathEnvName, filepath.Join(t.clientCertDir, "cert.pem"))
	t.T().Setenv(mtls.KeyFileEnvName, filepath.Join(t.clientCertDir, "key.pem"))

	signature, err := testdata.ReadFile("testdata/signature.example")
	t.Require().NoError(err)
	t.signatureExample = strings.TrimRight(string(signature), "\n")

	trustInfo, err := testdata.ReadFile("testdata/trust_info_serialized.example")
	t.Require().NoError(err)
	t.trustInfoExample = strings.TrimRight(string(trustInfo), "\n")
}

func (t *ClientTestSuite) newSelfSignedCertificate(commonName string) *tls.Certificate {
	req := csr.CertificateRequest{
		CN:         commonName,
		KeyRequest: csr.NewKeyRequest(),
		Hosts:      []string{"host", "central.stackrox", "central.stackrox.svc"}, // Include required SANs
	}

	caCert, _, caKey, err := initca.New(&req)
	t.Require().NoError(err)
	cert, err := tls.X509KeyPair(caCert, caKey)
	t.Require().NoError(err)

	return &cert
}

func (t *ClientTestSuite) TestGetPingOK() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Equal(pingRoute, r.URL.Path)

		_ = json.NewEncoder(w).Encode(
			v1.PongMessage{Status: "ok"},
		)
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	pong, err := c.GetPing(context.Background())
	t.Require().NoError(err)
	t.Equal("ok", pong.GetStatus())
}

func (t *ClientTestSuite) TestGetPingFailure() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Equal(pingRoute, r.URL.Path)

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, err = c.GetPing(context.Background())
	t.Require().Error(err)
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
			TrustInfoSerialized: t.trustInfoExample,
			TrustInfoSignature:  t.signatureExample,
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

			_, _, err = c.GetTLSTrustedCerts(context.Background())
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
			"trustInfoSerialized": t.trustInfoExample,
			"signature":           t.signatureExample,
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

	certs, internalCerts, err := c.GetTLSTrustedCerts(context.Background())
	t.Require().NoError(err)

	t.Require().Len(certs, 2)
	t.Equal("Root LoadBalancer Certificate Authority", certs[0].Subject.CommonName)
	t.Equal("StackRox Certificate Authority", certs[1].Subject.CommonName)

	t.Require().Len(internalCerts, 1)
	t.Equal("StackRox Certificate Authority", internalCerts[0].Subject.CommonName)
	t.Equal(certs[1].Raw, internalCerts[0].Raw)
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithSignatureSignedByAnotherPrivateKey_ShouldFail() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trustInfoSerialized": t.trustInfoExample,
			"signature":           invalidSignature,
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, _, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
	t.Require().ErrorIs(err, errInvalidTrustInfoSignature)
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithInvalidTrustInfo() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trustInfoSerialized": base64.StdEncoding.EncodeToString([]byte("Invalid trust info")),
			"signature":           t.signatureExample,
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, _, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_WithInvalidSignature() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"trustInfoSerialized": t.trustInfoExample,
			"signature":           base64.StdEncoding.EncodeToString([]byte("Invalid signature")),
		})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	_, _, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
	t.Require().ErrorIs(err, errInvalidTrustInfoSignature)
}

func (t *ClientTestSuite) TestGetTLSTrustedCertsWithDifferentSensorChallengeShouldFail() {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"trustInfoSerialized": t.trustInfoExample, "signature": t.signatureExample})
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	t.Require().NoError(err)

	mockNonceGenerator := mocks.NewMockNonceGenerator(t.mockCtrl)
	mockNonceGenerator.EXPECT().Nonce().Times(1).Return("some_token", nil)
	c.nonceGenerator = mockNonceGenerator

	_, _, err = c.GetTLSTrustedCerts(context.Background())
	t.Require().Error(err)
	t.Contains(err.Error(), fmt.Sprintf(`validating Central response failed: Sensor token "some_token" did not match received token %q`, exampleChallengeToken))
}

func (t *ClientTestSuite) TestGetTLSTrustedCerts_SecondaryCA() {
	// Base setup: primary chain must fail verification so the code evaluates the secondary path
	// This generates a self-signed primary CA that is not trusted by Sensor.
	primaryCA := t.newSelfSignedCertificate(mtls.ServiceCACommonName)
	primaryLeaf := primaryCA // use the primary CA as the leaf cert for simplicity

	// Load trusted secondary chain from testdata (used in cases where we want secondary verify to succeed)
	trustedCACertPEM, err := testdata.ReadFile("testdata/central/ca.pem")
	t.Require().NoError(err)
	trustedCAKeyPEM, err := testdata.ReadFile("testdata/central/ca-key.pem")
	t.Require().NoError(err)
	trustedLeafCertPEM, err := testdata.ReadFile("testdata/central/cert.pem")
	t.Require().NoError(err)
	trustedLeafKeyPEM, err := testdata.ReadFile("testdata/central/key.pem")
	t.Require().NoError(err)

	goodSecondaryCA, err := tls.X509KeyPair(trustedCACertPEM, trustedCAKeyPEM)
	t.Require().NoError(err)
	goodSecondaryLeaf, err := tls.X509KeyPair(trustedLeafCertPEM, trustedLeafKeyPEM)
	t.Require().NoError(err)

	createSignature := func(cert *tls.Certificate, data []byte) []byte {
		sign, err := cTLS.CreateSignature(cryptoutils.DerefPrivateKey(cert.PrivateKey), cTLS.SHA256, data)
		t.Require().NoError(err)
		return sign.Signature
	}

	// Untrusted secondary chain (verification will fail)
	badSecondaryCA := t.newSelfSignedCertificate("Untrusted Secondary CA")
	badSecondaryLeaf := badSecondaryCA

	testCases := []struct {
		name                string
		secondaryChain      [][]byte
		buildSecondarySign  func(trustInfoBytes []byte) []byte
		expectedErrContains string
		expectSuccess       bool
	}{
		{
			name: "secondary fallback succeeds",
			secondaryChain: [][]byte{
				goodSecondaryLeaf.Certificate[0],
				goodSecondaryCA.Certificate[0],
			},
			buildSecondarySign: func(b []byte) []byte { return createSignature(&goodSecondaryLeaf, b) },
			expectSuccess:      true,
		},
		{
			name:                "secondary chain empty when primary verification failed",
			secondaryChain:      [][]byte{},
			buildSecondarySign:  func(_ []byte) []byte { return nil },
			expectedErrContains: "validating primary Central certificate chain (no secondary certificate chain present)",
		},
		{
			name: "verifying secondary certificate chain fails",
			secondaryChain: [][]byte{
				badSecondaryLeaf.Certificate[0],
				badSecondaryCA.Certificate[0],
			},
			buildSecondarySign:  func(b []byte) []byte { return createSignature(badSecondaryLeaf, b) },
			expectedErrContains: "verifying secondary Central certificate chain",
		},
		{
			name: "validating payload signature with secondary CA fails",
			secondaryChain: [][]byte{
				goodSecondaryLeaf.Certificate[0],
				goodSecondaryCA.Certificate[0],
			},
			buildSecondarySign: func(b []byte) []byte {
				// Sign with a different key to trigger signature validation failure
				other := t.newSelfSignedCertificate("Different Signer")
				return createSignature(other, b)
			},
			expectedErrContains: "verifying payload signature with secondary CA",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			// Build TrustInfo with failing primary chain and scenario-specific secondary chain
			trustInfo := &v1.TrustInfo{
				SensorChallenge:  exampleChallengeToken,
				CentralChallenge: "central-challenge",
				CertChain: [][]byte{
					primaryLeaf.Certificate[0],
					primaryCA.Certificate[0],
				},
				SecondaryCertChain: tc.secondaryChain,
			}

			trustInfoBytes, err := trustInfo.MarshalVT()
			t.Require().NoError(err)

			primarySignature := createSignature(primaryLeaf, trustInfoBytes)
			secondarySignature := tc.buildSecondarySign(trustInfoBytes)

			// Test server that returns the TLSChallengeResponse built above
			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Contains(r.URL.String(), "/v1/tls-challenge?challengeToken=")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"trustInfoSerialized":  base64.StdEncoding.EncodeToString(trustInfoBytes),
					"signature":            base64.StdEncoding.EncodeToString(primarySignature),
					"signatureSecondaryCa": base64.StdEncoding.EncodeToString(secondarySignature),
				})
			}))

			ts.TLS = &tls.Config{Certificates: []tls.Certificate{*primaryCA}}
			ts.StartTLS()
			defer ts.Close()

			c, err := NewClient(ts.URL)
			t.Require().NoError(err)

			mockNonceGenerator := mocks.NewMockNonceGenerator(t.mockCtrl)
			mockNonceGenerator.EXPECT().Nonce().Times(1).Return(exampleChallengeToken, nil)
			c.nonceGenerator = mockNonceGenerator

			certs, internalCAs, err := c.GetTLSTrustedCerts(context.Background())
			if tc.expectSuccess {
				t.Require().NoError(err)
				t.Require().Len(certs, 2)
				t.Require().Len(internalCAs, 2)
				t.ElementsMatch(certs, internalCAs)

				// Verify that internalCAs contains the exact certificates from the TrustInfo
				expectedCAs := [][]byte{
					trustInfo.GetCertChain()[1],
					trustInfo.GetSecondaryCertChain()[1],
				}
				actualCAs := [][]byte{
					internalCAs[0].Raw,
					internalCAs[1].Raw,
				}
				t.ElementsMatch(expectedCAs, actualCAs)
			} else {
				t.Require().Error(err)
				t.Contains(err.Error(), tc.expectedErrContains)
			}
		})
	}
}

func (t *ClientTestSuite) TestExtractCentralCAsFromTrustInfo() {
	primaryCert := t.newSelfSignedCertificate("Primary CA")
	secondaryCert := t.newSelfSignedCertificate("Secondary CA")

	testCases := []struct {
		name                    string
		trustInfo               *v1.TrustInfo
		expectedCACount         int
		expectedPrimaryCAName   string
		expectedSecondaryCAName string
	}{
		{
			name: "with both chains",
			trustInfo: &v1.TrustInfo{
				CertChain: [][]byte{
					[]byte("leaf-cert-der"),
					primaryCert.Certificate[0],
				},
				SecondaryCertChain: [][]byte{
					[]byte("secondary-leaf-cert-der"),
					secondaryCert.Certificate[0],
				},
			},
			expectedCACount:         2,
			expectedPrimaryCAName:   "Primary CA",
			expectedSecondaryCAName: "Secondary CA",
		},
		{
			name: "only primary chain",
			trustInfo: &v1.TrustInfo{
				CertChain: [][]byte{
					[]byte("leaf-cert-der"),
					primaryCert.Certificate[0],
				},
				SecondaryCertChain: [][]byte{},
			},
			expectedCACount:       1,
			expectedPrimaryCAName: "Primary CA",
		},
		{
			name: "short chains",
			trustInfo: &v1.TrustInfo{
				CertChain: [][]byte{
					[]byte("only-leaf-cert-der"),
				},
				SecondaryCertChain: [][]byte{
					[]byte("only-secondary-leaf-cert-der"),
				},
			},
			expectedCACount: 0,
		},
		{
			name: "empty CA fields",
			trustInfo: &v1.TrustInfo{
				CertChain: [][]byte{
					[]byte("leaf-cert-der"),
					{},
				},
				SecondaryCertChain: [][]byte{
					[]byte("secondary-leaf-cert-der"),
					{},
				},
			},
			expectedCACount: 0,
		},
		{
			name: "invalid CA cert",
			trustInfo: &v1.TrustInfo{
				CertChain: [][]byte{
					[]byte("leaf-cert-der"),
					[]byte("invalid-ca-cert-data"),
				},
				SecondaryCertChain: [][]byte{},
			},
			expectedCACount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func() {
			centralCAs := extractCentralCAsFromTrustInfo(tc.trustInfo)

			t.Require().Len(centralCAs, tc.expectedCACount)

			if tc.expectedCACount >= 1 && tc.expectedPrimaryCAName != "" {
				t.Equal(tc.expectedPrimaryCAName, centralCAs[0].Subject.CommonName)
			}

			if tc.expectedCACount >= 2 && tc.expectedSecondaryCAName != "" {
				t.Equal(tc.expectedSecondaryCAName, centralCAs[1].Subject.CommonName)
			}
		})
	}
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
