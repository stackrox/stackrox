package tlsconfig

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/sliceutils"
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
	s.Require().Len(additionalCAs, 6, "Could not decode all certs")
	s.True(s.isCommonNameInCerts(additionalCAs, "CENTRAL_SERVICE: Central"), "Could not find cert from multiple crt file")

	// non .crt or .pem files should not be ignored
	s.FileExists("testdata/foo.txt")
	s.FileExists("testdata/cert.txt")
}

func (s *tlsConfigTestSuite) TestGetAdditionalCAFilePaths() {
	s.T().Setenv("ROX_MTLS_ADDITIONAL_CA_DIR", "testdata")
	filePaths, err := GetAdditionalCAFilePaths()
	s.Require().NoError(err)
	s.Len(filePaths, 5)
	s.Contains(filePaths, "testdata/cert.txt")
	s.Contains(filePaths, "testdata/cert.pem")
	s.Contains(filePaths, "testdata/crt01.crt")
	s.Contains(filePaths, "testdata/multiple_certs.crt")
	s.Contains(filePaths, "testdata/symlinked.pem")
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

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldLoadChainWithOneCert() {

	certs, keys, err := createCertChainWithLengthOf(1)
	s.Require().NoError(err)

	tmpDir := s.T().TempDir()

	s.Require().NoError(writeCerts(tmpDir, certs...))
	s.Require().NoError(writeKey(tmpDir, keys[0]))

	result, err := MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	s.Len(result.Certificate, 1)
	s.Equal("Cert 0", result.Leaf.Subject.CommonName)

	leaf, err := x509.ParseCertificate(result.Certificate[0])
	s.Require().NoError(err)
	s.Equal("Cert 0", leaf.Subject.CommonName)
}

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldLoadChainWithMultipleCerts() {

	certs, keys, err := createCertChainWithLengthOf(3)
	s.Require().NoError(err)

	tmpDir := s.T().TempDir()

	s.Require().NoError(writeCerts(tmpDir, certs...))
	s.Require().NoError(writeKey(tmpDir, keys[0]))

	result, err := MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	s.Len(result.Certificate, 3)
	s.Equal("Cert 2", result.Leaf.Subject.CommonName)

	leaf, err := x509.ParseCertificate(result.Certificate[0])
	s.Require().NoError(err)
	s.Equal("Cert 2", leaf.Subject.CommonName)

	intermediate, err := x509.ParseCertificate(result.Certificate[1])
	s.Require().NoError(err)
	s.Equal("Cert 1", intermediate.Subject.CommonName)

	root, err := x509.ParseCertificate(result.Certificate[2])
	s.Require().NoError(err)
	s.Equal("Cert 0", root.Subject.CommonName)

}

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldIgnoreWhenKeyIsMissing() {
	certs, _, err := createCertChainWithLengthOf(3)
	s.Require().NoError(err)

	tmpDir := s.T().TempDir()
	s.Require().NoError(writeCerts(tmpDir, certs...))

	actual, err := MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.NoError(err)
	s.Nil(actual)

}

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldIgnoreWhenCertIsMissing() {
	tmpDir := s.T().TempDir()
	_, keys, err := createCertChainWithLengthOf(3)
	s.Require().NoError(err)
	s.Require().NoError(writeKey(tmpDir, keys[0]))

	actual, err := MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.NoError(err)
	s.Nil(actual)

}

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldIgnoreWhenBothCertAndKeyAreMissing() {
	tmpDir := s.T().TempDir()
	actual, err := MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.NoError(err)
	s.Nil(actual)
}

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldReturnErrorWhenTheCertIsMalformed() {
	tmpDir := s.T().TempDir()

	_, keys, err := createCertChainWithLengthOf(1)
	s.Require().NoError(err)
	s.Require().NoError(writeKey(tmpDir, keys[0]))

	badCertFile, err := os.Create(path.Join(tmpDir, TLSCertFileName))
	s.Require().NoError(err)
	_, err = badCertFile.WriteString("invalid cert")
	s.Require().NoError(err)
	s.Require().NoError(badCertFile.Close())

	_, err = MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.ErrorContains(err, "failed to find any PEM data in certificate input")
}

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldReturnAnErrorWhenKeyIsMalformed() {
	tmpDir := s.T().TempDir()

	certs, _, err := createCertChainWithLengthOf(1)
	s.Require().NoError(err)
	s.Require().NoError(writeCerts(tmpDir, certs...))

	badKeyFile, err := os.Create(path.Join(tmpDir, TLSKeyFileName))
	s.Require().NoError(err)
	_, err = badKeyFile.WriteString("invalid key")
	s.Require().NoError(err)
	s.Require().NoError(badKeyFile.Close())

	_, err = MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.ErrorContains(err, "failed to find any PEM data in key input")
}

func (s *tlsConfigTestSuite) TestMaybeGetDefaultTLSCertificate_ShouldReturnErrorAndWarnTheUserWhenCertsAreInWrongOrder() {
	tmpDir := s.T().TempDir()

	certs, keys, err := createCertChainWithLengthOf(3)
	s.Require().NoError(err)
	s.Require().NoError(writeCerts(tmpDir, certs[1], certs[0], certs[2]))
	s.Require().NoError(writeKey(tmpDir, keys[0]))

	_, err = MaybeGetDefaultTLSCertificateFromDirectory(tmpDir)
	s.ErrorContains(err, "private key does not match public key")
	s.ErrorContains(err, "ensure that the certificate chain is in the correct order")
}

func createCertChainWithLengthOf(length int) ([]*x509.Certificate, []*ecdsa.PrivateKey, error) {
	var certs []*x509.Certificate
	var privateKeys []*ecdsa.PrivateKey
	for i := 0; i < length; i++ {
		var parentCert = &x509.Certificate{}
		var parentKey = &ecdsa.PrivateKey{}
		if i > 0 {
			parentCert = certs[i-1]
			parentKey = privateKeys[i-1]
		}
		serialNumber := big.NewInt(int64(i) + 1)
		template := &x509.Certificate{
			SerialNumber: serialNumber,
			IsCA:         i != length-1,
			Subject: pkix.Name{
				CommonName: fmt.Sprintf("Cert %d", i),
			},
		}
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		if i == 0 {
			parentKey = privateKey
		}
		certBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, &privateKey.PublicKey, parentKey)
		if err != nil {
			return nil, nil, err
		}
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			return nil, nil, err
		}
		certs = append(certs, cert)
		privateKeys = append(privateKeys, privateKey)
	}
	// inverse the order of the certs so that the leaf is first
	sliceutils.ReverseInPlace(certs)
	sliceutils.ReverseInPlace(privateKeys)
	return certs, privateKeys, nil
}

func writeCerts(dir string, certs ...*x509.Certificate) error {
	certFilePath := path.Join(dir, TLSCertFileName)
	certFile, err := os.Create(certFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = certFile.Close()
	}()

	for _, cert := range certs {
		if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			return err
		}
	}
	return nil
}

func writeKey(dir string, privateKey *ecdsa.PrivateKey) error {
	keyFilePath := path.Join(dir, TLSKeyFileName)
	keyFile, err := os.Create(keyFilePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = keyFile.Close()
	}()
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	return pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes})
}
