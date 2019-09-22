package tests

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientCARequested(t *testing.T) {
	t.Parallel()

	clientCAFile := os.Getenv("CLIENT_CA_PATH")
	require.NotEmpty(t, clientCAFile, "no client CA file path set")
	pemBytes, err := ioutil.ReadFile(clientCAFile)
	require.NoErrorf(t, err, "Could not read client CA file %s", clientCAFile)

	caCert, err := helpers.ParseCertificatePEM(pemBytes)
	require.NoError(t, err, "Could not parse client CA PEM data")

	var acceptableCAs [][]byte
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "central.stackrox",
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			acceptableCAs = cri.AcceptableCAs
			return &tls.Certificate{}, nil
		},
	}

	conn, err := tls.Dial("tcp", testutils.RoxAPIEndpoint(t), tlsConf)
	require.NoError(t, err, "could not connect to central")
	_ = conn.Handshake()
	_ = conn.Close()

	found := false
	for _, acceptableCA := range acceptableCAs {
		if bytes.Equal(acceptableCA, caCert.RawSubject) {
			found = true
			break
		}
	}

	assert.True(t, found, "server did not request appropriate client certs")
}
