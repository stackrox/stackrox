package aws

import (
	"strings"
	"testing"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAWSCerts(t *testing.T) {
	certs := awsCerts()
	require.NotEmpty(t, certs)
	for _, cert := range certs {
		assert.Contains(t, strings.Join(cert.Subject.Organization, ","), "Amazon Web Services")
	}

	certsAgain := awsCerts()
	require.Len(t, certsAgain, len(certs))
	for i := range certs {
		assert.Same(t, certs[i], certsAgain[i], "awsCerts() should memoize and return the same parsed certificates on every call")
	}
}

func BenchmarkAWSCertsParsing(b *testing.B) {
	for range b.N {
		helpers.ParseCertificatesPEM(awsCertsPEM)
	}
}
