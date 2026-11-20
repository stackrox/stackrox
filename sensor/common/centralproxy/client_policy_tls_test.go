package centralproxy

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyTLSAcceptsCertificatesFromReferencedCAs(t *testing.T) {
	serverCA, serverKey, _ := clientPolicyTestCA(t)
	clientCA, clientKey, _ := clientPolicyTestCA(t)
	serverCert := signedTestCertificate(t, serverCA, serverKey, "localhost", x509.ExtKeyUsageServerAuth)
	clientCert := signedTestCertificate(t, clientCA, clientKey, "custom-client", x509.ExtKeyUsageClientAuth)
	config, updateCA, updateCert := proxyTLSConfig(serverCert)
	(&ClientPolicy{}).configureTLS(config, updateCA)
	live, err := config.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.Nil(t, live.ClientCAs, "policy verification must not advertise a shared CA pool")
	assert.Equal(t, tls.RequestClientCert, live.ClientAuth)
	rotatedCert := signedTestCertificate(t, serverCA, serverKey, "localhost", x509.ExtKeyUsageServerAuth)
	updateCert(&rotatedCert)
	live, err = config.GetConfigForClient(nil)
	require.NoError(t, err)
	assert.Equal(t, rotatedCert.Certificate, live.Certificates[0].Certificate)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "missing certificate", http.StatusUnauthorized)
			return
		}
		assert.Empty(t, r.TLS.VerifiedChains)
		assert.Equal(t, "custom-client", r.TLS.PeerCertificates[0].Subject.CommonName)
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = config
	server.StartTLS()
	t.Cleanup(server.Close)
	roots := x509.NewCertPool()
	roots.AddCert(serverCA)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: "localhost",
		Certificates: []tls.Certificate{clientCert}, MinVersion: tls.VersionTLS12}}
	t.Cleanup(transport.CloseIdleConnections)
	response, err := (&http.Client{Transport: transport}).Get(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())
}
