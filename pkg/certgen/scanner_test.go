package certgen

import (
	"crypto/tls"
	"testing"

	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/x509utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIssueScannerCerts(t *testing.T) {
	suite.Run(t, new(issueScannerCertTestSuite))
}

type issueScannerCertTestSuite struct {
	suite.Suite
}

func (s *issueScannerCertTestSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

func (s *issueScannerCertTestSuite) TestIssueScannerCertsSAN() {
	cases := []struct {
		name          string
		namespace     string
		scannerSANs   []string
		scannerDBSANs []string
	}{
		{
			name:          "no namespace",
			scannerSANs:   []string{"scanner.stackrox", "scanner.stackrox.svc"},
			scannerDBSANs: []string{"scanner-db.stackrox", "scanner-db.stackrox.svc"},
		},
		{
			name:          "stackrox namespace",
			namespace:     "stackrox",
			scannerSANs:   []string{"scanner.stackrox", "scanner.stackrox.svc"},
			scannerDBSANs: []string{"scanner-db.stackrox", "scanner-db.stackrox.svc"},
		},
		{
			name:          "custom namespace",
			namespace:     "custom",
			scannerSANs:   []string{"scanner.stackrox", "scanner.stackrox.svc", "scanner.custom", "scanner.custom.svc"},
			scannerDBSANs: []string{"scanner-db.stackrox", "scanner-db.stackrox.svc", "scanner-db.custom", "scanner-db.custom.svc"},
		},
	}

	for _, tc := range cases {
		s.T().Run(tc.name, func(t *testing.T) {
			var opts []mtls.IssueCertOption
			if tc.namespace != "" {
				opts = []mtls.IssueCertOption{mtls.WithNamespace(tc.namespace)}
			}
			fileMap := make(map[string][]byte)
			ca, err := mtls.LoadDefaultCA()
			require.NoError(t, err)
			err = IssueScannerCerts(fileMap, ca, opts...)
			require.NoError(t, err)
			cert, err := tls.X509KeyPair(fileMap["scanner-cert.pem"], fileMap["scanner-key.pem"])
			require.NoError(t, err)
			assertSANs(t, cert, tc.scannerSANs)
			cert, err = tls.X509KeyPair(fileMap["scanner-db-cert.pem"], fileMap["scanner-db-key.pem"])
			require.NoError(t, err)
			assertSANs(t, cert, tc.scannerDBSANs)
		})
	}
}

func assertSANs(t *testing.T, cert tls.Certificate, sans []string) {
	chain, err := x509utils.ParseCertificateChain(cert.Certificate)
	require.NoError(t, err)
	assert.ElementsMatch(t, sans, chain[0].DNSNames)
}
