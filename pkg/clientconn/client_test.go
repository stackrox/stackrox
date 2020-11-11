package clientconn

import (
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const centralEndpoint = "central.stackrox:443"

func TestAddRootCA(t *testing.T) {
	const certCount = 2
	cert := &x509.Certificate{SubjectKeyId: []byte(`SubjectKeyId1`), RawSubject: []byte(`RawSubject1`)}
	cert2 := &x509.Certificate{SubjectKeyId: []byte(`SubjectKeyId2`), RawSubject: []byte(`RawSubject2`)}

	opts, err := OptionsForEndpoint(centralEndpoint, AddRootCAs(cert, cert2))
	require.NoError(t, err)

	// read system root CAs
	sysCertPool, err := x509.SystemCertPool()
	require.NoError(t, err)

	addedCertsCount := len(opts.TLS.RootCAs.Subjects()) - len(sysCertPool.Subjects())
	assert.Equalf(t, addedCertsCount, certCount, "Expected %d certificates being added", certCount)
}

func TestRootCA_WithNilCA_ShouldPanic(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = OptionsForEndpoint(centralEndpoint, AddRootCAs(nil))
	})
}
