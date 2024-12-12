package x509utils

import (
	"bytes"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertX509Certificates(t *testing.T) {
	// test PEM conversion to x509
	certs, err := ConvertPEMTox509Certs([]byte(pemChain))
	require.NoError(t, err)
	assert.Len(t, certs, 3)

	// test empty cert returns err
	certs, err = ConvertPEMTox509Certs([]byte{})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid PEM")
	require.Empty(t, certs)

	// test invalid certificate
	certs, err = ConvertPEMTox509Certs([]byte("invalid PEM"))
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid PEM")
	require.Empty(t, certs)
}

func TestInvalidX509CertShouldReturnError(t *testing.T) {
	invalidX509PEM := bytes.NewBuffer([]byte{})
	err := pem.Encode(invalidX509PEM, &pem.Block{
		Headers: map[string]string{
			"header-key": "value",
		},
		Type:  "EXAMPLE TEST",
		Bytes: make([]byte, 0),
	})
	require.NoError(t, err)

	certs, err := ConvertPEMTox509Certs(invalidX509PEM.Bytes())
	require.Error(t, err)
	require.ErrorContains(t, err, "could not convert cert: x509: malformed certificate")
	require.Empty(t, certs)
}
