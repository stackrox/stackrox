//go:build test_e2e

package tests

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalCert(t *testing.T) {

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "central.stackrox",
	}

	conn, err := tls.Dial("tcp", centralgrpc.RoxAPIEndpoint(t), tlsConf)
	require.NoError(t, err)
	defer utils.IgnoreError(conn.Close)

	certs := conn.ConnectionState().PeerCertificates
	require.NotEmpty(t, certs)
	leaf := certs[0]

	subj := mtls.SubjectFromCommonName(leaf.Subject.CommonName)
	assert.Equal(t, mtls.CentralSubject, subj)
}

func TestCustomCert(t *testing.T) {

	testCentralCertCAPEM := os.Getenv("ROX_TEST_CA_PEM")
	if testCentralCertCAPEM == "" {
		t.Skip("No test CA pem specified")
	}

	centralCN := mustGetEnv(t, "ROX_TEST_CENTRAL_CN")

	trustPool := x509.NewCertPool()
	ok := trustPool.AppendCertsFromPEM([]byte(testCentralCertCAPEM))
	require.True(t, ok)

	tlsConf := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         centralCN,
		RootCAs:            trustPool,
	}

	conn, err := tls.Dial("tcp", centralgrpc.RoxAPIEndpoint(t), tlsConf)
	require.NoError(t, err)
	defer utils.IgnoreError(conn.Close)

	certChains := conn.ConnectionState().VerifiedChains
	require.Len(t, certChains, 1)
}
