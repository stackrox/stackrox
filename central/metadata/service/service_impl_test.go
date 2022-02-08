package service

import (
	"context"
	"crypto/x509"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	cTLS "github.com/google/certificate-transparency-go/tls"
	v1 "github.com/stackrox/rox/generated/api/v1"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const (
	validChallengeToken   = "h83_PGhSqS8OAvplb8asYMfPHy1JhVVMKcajYyKmrIU="
	invalidChallengeToken = "invalid"
)

func TestServiceImpl(t *testing.T) {
	suite.Run(t, new(serviceImplTestSuite))
}

type serviceImplTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *serviceImplTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *serviceImplTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *serviceImplTestSuite) SetupTest() {
	wd, err := os.Getwd()
	s.Require().NoError(err)

	testdata := filepath.Join(wd, "testdata")
	s.envIsolator.Setenv("ROX_MTLS_ADDITIONAL_CA_DIR", path.Join(testdata, "additional-ca"))

	err = testutilsMTLS.LoadTestMTLSCerts(s.envIsolator)
	s.Require().NoError(err)
}

func (s *serviceImplTestSuite) TestTLSChallenge() {
	service := serviceImpl{}
	req := &v1.TLSChallengeRequest{
		ChallengeToken: validChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().NoError(err)

	trustInfo := &v1.TrustInfo{}
	err = proto.Unmarshal(resp.GetTrustInfoSerialized(), trustInfo)
	s.Require().NoError(err)

	// Verify that additional CAs were received
	s.Require().Len(trustInfo.GetAdditionalCas(), 2)
	additionalCACert, err := x509.ParseCertificate(trustInfo.GetAdditionalCas()[0])
	s.Require().NoError(err)
	s.Equal("nginx LoadBalancer", additionalCACert.Subject.CommonName)

	// Verify signature
	s.Require().Len(trustInfo.GetCertChain(), 2)
	cert, err := x509.ParseCertificate(trustInfo.GetCertChain()[0])
	s.Require().NoError(err)

	err = verifySignature(cert, resp)
	s.Require().NoError(err, "Could not verify central auth signature")

	s.Contains(cert.DNSNames, "central.stackrox", "Expected leaf certificate.")
}

func (s *serviceImplTestSuite) TestTLSChallenge_VerifySignatureWithCACert_ShouldFail() {
	service := serviceImpl{}
	req := &v1.TLSChallengeRequest{
		ChallengeToken: validChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().NoError(err)

	trustInfo := &v1.TrustInfo{}
	err = proto.Unmarshal(resp.GetTrustInfoSerialized(), trustInfo)
	s.Require().NoError(err)

	// Read root CA from response
	caCert := trustInfo.CertChain[len(trustInfo.CertChain)-1]
	cert, err := x509.ParseCertificate(caCert)
	s.Require().NoError(err)
	s.Equal(cert.Subject.CommonName, "StackRox Certificate Authority", "Not a root CA?")
	s.True(cert.IsCA)

	err = verifySignature(cert, resp)
	s.Error(err)
	s.Equal("failed to verify rsa signature: crypto/rsa: verification error", err.Error())
}

func (s *serviceImplTestSuite) TestTLSChallenge_ShouldFailWithInvalidToken() {
	service := serviceImpl{}
	req := &v1.TLSChallengeRequest{
		ChallengeToken: invalidChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().Error(err)
	s.EqualError(err, "challenge token must be a valid base64 string: illegal base64 data at input byte 4: invalid arguments")
	s.Nil(resp)
}

func verifySignature(cert *x509.Certificate, resp *v1.TLSChallengeResponse) error {
	return cTLS.VerifySignature(cert.PublicKey, resp.GetTrustInfoSerialized(), cTLS.DigitallySigned{
		Signature: resp.GetSignature(),
		Algorithm: cTLS.SignatureAndHashAlgorithm{
			Hash:      cTLS.SHA256,
			Signature: cTLS.SignatureAlgorithmFromPubKey(cert.PublicKey),
		},
	})
}
