package helper

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func NewCentralHTTPTestServer(t *testing.T) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/kernel-objects/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("probe"))
	})
	handler.HandleFunc("/api/extensions/scannerdefinitions", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("definitions"))
	})

	ca := generateAdditionalCA(t)
	serverCert, err := ca.IssueCertForSubject(mtls.CentralSubject)
	require.NoError(t, err)

	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{issuedCertToTLSCertificate(t, serverCert)},
	}
	server.StartTLS()
	return server
}

func generateAdditionalCA(t *testing.T) mtls.CA {
	req := csr.CertificateRequest{
		CN:         "localhost test certificate", // should NOT be StackRox CA Common Name
		KeyRequest: csr.NewKeyRequest(),
		Hosts:      []string{"localhost"},
	}

	caCert, _, caKey, err := initca.New(&req)
	require.NoError(t, err)
	ca, err := mtls.LoadCAForSigning(caCert, caKey)
	require.NoError(t, err)
	return ca
}

func NewHTTPTestClient(t *testing.T, serviceType storage.ServiceType) *http.Client {
	issuedCert, err := mtls.IssueNewCert(
		mtls.NewInitSubject(
			centralsensor.RegisteredInitCertClusterID,
			serviceType,
			uuid.NewV4()))
	require.NoError(t, err)
	clientCert := issuedCertToTLSCertificate(t, issuedCert)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{clientCert},
		},
	}

	return &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}
}

func issuedCertToTLSCertificate(t *testing.T, collectorCert *mtls.IssuedCert) tls.Certificate {
	clientCert, err := tls.X509KeyPair(collectorCert.CertPEM, collectorCert.KeyPEM)
	require.NoError(t, err)
	clientCert.Leaf = collectorCert.X509Cert
	return clientCert
}
