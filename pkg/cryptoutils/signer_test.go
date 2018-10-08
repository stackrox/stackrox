package cryptoutils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	mathRand "math/rand"
	"testing"

	// Needed for accessing the function behind crypto.SHA256.
	_ "crypto/sha256"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ed25519"
)

type signerTestSuite struct {
	signer Signer
	suite.Suite
}

func TestED25519(t *testing.T) {
	t.Parallel()

	_, pk, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	testSuite := &signerTestSuite{
		signer: NewED25519Signer(pk),
	}
	suite.Run(t, testSuite)
}

func TestECDSA256(t *testing.T) {
	t.Parallel()

	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	testSuite := &signerTestSuite{
		signer: NewECDSA256Signer(pk, crypto.SHA256),
	}
	suite.Run(t, testSuite)
}

func (s *signerTestSuite) TestSignEmptyMessage() {
	sig, err := s.signer.Sign(nil, rand.Reader)
	s.Require().NoError(err)

	verifyErr := s.signer.Verify(nil, sig)
	s.NoError(verifyErr)
}

func (s *signerTestSuite) TestSignRandomShortMessage() {
	msg := make([]byte, 10)
	_, err := rand.Read(msg)
	s.Require().NoError(err)

	sig, err := s.signer.Sign(msg, rand.Reader)
	s.Require().NoError(err)

	verifyErr := s.signer.Verify(msg, sig)
	s.NoError(verifyErr)
}

func (s *signerTestSuite) TestSignRandomLongMessage() {
	msg := make([]byte, 2048)
	_, err := rand.Read(msg)
	s.Require().NoError(err)

	sig, err := s.signer.Sign(msg, rand.Reader)
	s.Require().NoError(err)

	verifyErr := s.signer.Verify(msg, sig)
	s.NoError(verifyErr)
}

func (s *signerTestSuite) TestTamperWithMessage() {
	msg := make([]byte, 2048)
	_, err := rand.Read(msg)
	s.Require().NoError(err)

	sig, err := s.signer.Sign(msg, rand.Reader)
	s.Require().NoError(err)

	msg[mathRand.Intn(len(msg))]++
	verifyErr := s.signer.Verify(msg, sig)
	s.Error(verifyErr)
}

func (s *signerTestSuite) TestTamperWithSignature() {
	msg := make([]byte, 2048)
	_, err := rand.Read(msg)
	s.Require().NoError(err)

	sig, err := s.signer.Sign(msg, rand.Reader)
	s.Require().NoError(err)

	sig[mathRand.Intn(len(sig))]++
	verifyErr := s.signer.Verify(msg, sig)
	s.Error(verifyErr)
}
