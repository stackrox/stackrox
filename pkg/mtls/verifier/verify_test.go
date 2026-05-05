package verifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLeafCertFromDirectory(t *testing.T) {
	ca, err := certgen.GenerateCA()
	require.NoError(t, err)

	issuedCert, err := ca.IssueCertForSubject(mtls.CentralSubject)
	require.NoError(t, err)

	t.Run("valid cert and key", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, mtls.ServiceCertFileName), issuedCert.CertPEM, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, mtls.ServiceKeyFileName), issuedCert.KeyPEM, 0600))

		cert, err := loadLeafCertFromDirectory(dir)
		require.NoError(t, err)
		require.NotNil(t, cert)
		require.NotNil(t, cert.Leaf)
		assert.Contains(t, cert.Leaf.DNSNames, "central.stackrox.svc")
	})

	t.Run("missing files returns error", func(t *testing.T) {
		dir := t.TempDir()
		cert, err := loadLeafCertFromDirectory(dir)
		assert.Error(t, err)
		assert.Nil(t, cert)
	})

	t.Run("cert only, no key returns error", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, mtls.ServiceCertFileName), issuedCert.CertPEM, 0644))

		cert, err := loadLeafCertFromDirectory(dir)
		assert.Error(t, err)
		assert.Nil(t, cert)
	})
}
