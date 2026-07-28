package aws

import (
	"strings"
	"testing"

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
		assert.Equal(t, certs[i], certsAgain[i], "awsCerts() should return the same parsed certificates on every call")
	}
}
