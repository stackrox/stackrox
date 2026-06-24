package helper

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	require.NoError(t, err)
	serial.Add(serial, big.NewInt(1))

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	ski := sha256.Sum256(pubDER)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "localhost test certificate"},
		DNSNames:              []string{"localhost"},
		NotBefore:             now,
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		SubjectKeyId:          ski[:20],
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	ca, err := mtls.LoadCAForSigning(certPEM, keyPEM)
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
