package centralproxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls/clientca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

func TestProxyTLSVerifiesOptionalPrometheusCertificate(t *testing.T) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "OpenShift test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)
	serverCert := signedTestCertificate(t, ca, caKey, "localhost", x509.ExtKeyUsageServerAuth)
	clientCert := signedTestCertificate(t, ca, caKey, env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	expiredClientCert := signedTestCertificate(t, ca, caKey, env.SecureMetricsClientCertCN.Setting(),
		x509.ExtKeyUsageClientAuth, time.Now().Add(-time.Minute))
	otherClientCert := signedTestCertificate(t, ca, caKey, "other-service-account", x509.ExtKeyUsageClientAuth)
	untrustedKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	untrustedDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &untrustedKey.PublicKey, untrustedKey)
	require.NoError(t, err)
	untrustedCA, err := x509.ParseCertificate(untrustedDER)
	require.NoError(t, err)
	untrustedClientCert := signedTestCertificate(t, untrustedCA, untrustedKey,
		env.SecureMetricsClientCertCN.Setting(), x509.ExtKeyUsageClientAuth)
	config, onClientCA, _ := proxyTLSConfig(serverCert)
	updateClientCA := clientca.OnUpdate(onClientCA)
	updateClientCA(&v1.ConfigMap{
		Data: map[string]string{env.SecureMetricsClientCAKey.Setting(): string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))},
	})
	assert.True(t, config.SessionTicketsDisabled)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/console" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := authenticateMetricsScrape(r); err != nil {
			http.Error(w, "invalid client", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = config
	server.StartTLS()
	t.Cleanup(server.Close)
	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(ca)
	consoleTransport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCAs,
		ServerName: "localhost", MinVersion: tls.VersionTLS12}}
	t.Cleanup(consoleTransport.CloseIdleConnections)
	consoleResponse, err := (&http.Client{Transport: consoleTransport}).Get(server.URL + "/console")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, consoleResponse.StatusCode)
	require.NoError(t, consoleResponse.Body.Close())

	for _, tc := range []struct {
		name     string
		certs    []tls.Certificate
		wantCode int
		wantErr  bool
	}{
		{"Prometheus certificate", []tls.Certificate{clientCert}, http.StatusOK, false},
		{"no certificate", nil, http.StatusUnauthorized, false},
		{"other service account", []tls.Certificate{otherClientCert}, http.StatusUnauthorized, false},
		{"expired certificate", []tls.Certificate{expiredClientCert}, 0, true},
		{"untrusted signer", []tls.Certificate{untrustedClientCert}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCAs, Certificates: tc.certs,
				ServerName: "localhost", MinVersion: tls.VersionTLS12}}
			t.Cleanup(transport.CloseIdleConnections)
			response, err := (&http.Client{Transport: transport}).Get(server.URL)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantCode, response.StatusCode)
			require.NoError(t, response.Body.Close())
		})
	}

	updateClientCA(&v1.ConfigMap{Data: map[string]string{env.SecureMetricsClientCAKey.Setting(): "invalid-PEM"}})
	liveConfig, err := config.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, liveConfig.ClientCAs.Subjects())

	updateClientCA(&v1.ConfigMap{Data: map[string]string{
		env.SecureMetricsClientCAKey.Setting(): string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: untrustedDER})),
	}})
	for _, test := range []struct {
		name    string
		cert    tls.Certificate
		allowed bool
	}{
		{"rotated signer is accepted", untrustedClientCert, true},
		{"previous signer is rejected", clientCert, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCAs,
				Certificates: []tls.Certificate{test.cert}, ServerName: "localhost", MinVersion: tls.VersionTLS12}}
			t.Cleanup(transport.CloseIdleConnections)
			response, err := (&http.Client{Transport: transport}).Get(server.URL)
			if !test.allowed {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.StatusCode)
			require.NoError(t, response.Body.Close())
		})
	}
}

func signedTestCertificate(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, cn string,
	usage x509.ExtKeyUsage, expiry ...time.Time,
) tls.Certificate {
	t.Helper()
	notAfter := time.Now().Add(time.Hour)
	if len(expiry) > 0 {
		notAfter = expiry[0]
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	require.NoError(t, err)
	leaf, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der, ca.Raw}, PrivateKey: key, Leaf: leaf}
}
