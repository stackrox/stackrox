package tlsconfig

import (
	"crypto/x509"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestTlsConfig(t *testing.T) {
	suite.Run(t, new(tlsConfigTestSuite))
}

type tlsConfigTestSuite struct {
	suite.Suite
}

func (s *tlsConfigTestSuite) TestGetAdditionalCAs() {
	s.T().Setenv("ROX_MTLS_ADDITIONAL_CA_DIR", "testdata")

	additionalCAs, err := GetAdditionalCAs()
	s.Require().NoError(err)
	s.Require().Len(additionalCAs, 3, "Could not decode all certs")
	s.True(s.isCommonNameInCerts(additionalCAs, "CENTRAL_SERVICE: Central"), "Could not find cert from multiple crt file")

	// non .crt files should be ignored
	s.FileExists("testdata/no_ca_cert.pem")
}

func (s *tlsConfigTestSuite) isCommonNameInCerts(DERs [][]byte, commonName string) bool {
	var result bool
	for _, DER := range DERs {
		c, err := x509.ParseCertificate(DER)
		s.Require().NoError(err)
		fmt.Println(c.Subject.CommonName)
		if c.Subject.CommonName == commonName {
			result = true
		}
	}
	return result
}
