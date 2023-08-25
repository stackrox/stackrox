package testutils

import (
	"crypto/tls"
	"testing"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/stretchr/testify/require"
)

// IssueSelfSignedCert issues a self-signed certificate.
func IssueSelfSignedCert(t *testing.T, commonName string, dnsNames ...string) tls.Certificate {
	req := csr.CertificateRequest{
		CN:         commonName,
		KeyRequest: csr.NewKeyRequest(),
		Hosts:      dnsNames,
	}

	caCert, _, caKey, err := initca.New(&req)
	require.NoError(t, err)
	cert, err := tls.X509KeyPair(caCert, caKey)
	require.NoError(t, err)
	return cert
}
