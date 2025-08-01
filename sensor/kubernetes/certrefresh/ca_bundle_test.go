package certrefresh

import (
	"crypto/x509"
	"testing"

	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/x509utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCABundleData(t *testing.T) {
	primaryCA := createTestCertificate(t, "Primary CA")
	secondaryCA := createTestCertificate(t, "Secondary CA")
	certs := []*x509.Certificate{primaryCA, secondaryCA}

	data, err := createCABundleData(certs)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	caBundle, exists := data[pkgKubernetes.TLSCABundleKey]
	assert.True(t, exists, "CA bundle should be present")
	assert.NotEmpty(t, caBundle, "CA bundle should not be empty")

	assert.Contains(t, caBundle, "-----BEGIN CERTIFICATE-----", "CA bundle should contain PEM certificates")
	assert.Contains(t, caBundle, "-----END CERTIFICATE-----", "CA bundle should contain PEM certificates")

	parsedCerts, err := x509utils.ConvertPEMTox509Certs([]byte(caBundle))
	require.NoError(t, err)
	require.Len(t, parsedCerts, 2, "Should contain exactly 2 certificates")

	commonNames := make([]string, len(parsedCerts))
	for i, cert := range parsedCerts {
		commonNames[i] = cert.Subject.CommonName
	}
	assert.Contains(t, commonNames, "Primary CA", "Bundle should contain Primary CA")
	assert.Contains(t, commonNames, "Secondary CA", "Bundle should contain Secondary CA")
}

func TestCreateCABundleDataEmptyInput(t *testing.T) {
	data, err := createCABundleData([]*x509.Certificate{})
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestCreateCABundleDataSingleCert(t *testing.T) {
	cert := createTestCertificate(t, "Single CA")
	certs := []*x509.Certificate{cert}

	data, err := createCABundleData(certs)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	caBundle, exists := data[pkgKubernetes.TLSCABundleKey]
	assert.True(t, exists)

	parsedCerts, err := x509utils.ConvertPEMTox509Certs([]byte(caBundle))
	require.NoError(t, err)
	require.Len(t, parsedCerts, 1, "Should contain exactly 1 certificate")
	assert.Equal(t, "Single CA", parsedCerts[0].Subject.CommonName)
}

func createTestCertificate(t *testing.T, commonName string) *x509.Certificate {
	tlsCert := testutils.IssueSelfSignedCert(t, commonName)
	return tlsCert.Leaf
}
