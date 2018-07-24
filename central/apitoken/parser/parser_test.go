package parser

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"strings"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	"bitbucket.org/stack-rox/apollo/pkg/auth/permissions"
	"bitbucket.org/stack-rox/apollo/pkg/auth/tokenbased"
	pkgJWT "bitbucket.org/stack-rox/apollo/pkg/jwt"
	"github.com/stretchr/testify/suite"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

const fakeRole = "FAKEROLE"

type mockRoleStore struct {
}

func (mockRoleStore) GetRole(name string) (role permissions.Role, exists bool) {
	if name == fakeRole {
		return permissions.NewAllAccessRole(fakeRole), true
	}
	return
}

func headersFrom(token string) map[string][]string {
	return map[string][]string{
		"authorization": {fmt.Sprintf("Bearer %s", token)},
	}
}

type APITokenTestSuite struct {
	suite.Suite
	signer signer.Signer
	parser tokenbased.IdentityParser
}

// create a new signer with a fresh random public key, fataling
// out if any errors are encountered along the way.
func (suite *APITokenTestSuite) createSigner() signer.Signer {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	suite.Require().NoError(err)
	s, err := signer.NewFromBytes(x509.MarshalPKCS1PrivateKey(privateKey))
	suite.Require().NoError(err)
	return s
}

func (suite *APITokenTestSuite) SetupTest() {
	suite.signer = suite.createSigner()
	suite.parser = New(suite.signer, mockRoleStore{})
}

// Happy path
func (suite *APITokenTestSuite) TestWorksWithExistingRole() {
	token, err := suite.signer.SignedJWT(fakeRole)
	suite.Require().NoError(err)
	identity, err := suite.parser.Parse(headersFrom(token), nil)
	suite.Require().NoError(err)
	suite.Equal(fakeRole, identity.Role().Name())
	suite.True(identity.Expiration().After(time.Now()))
	suite.True(identity.Expiration().Before(time.Now().Add(366 * 24 * time.Hour)))
}

func (suite *APITokenTestSuite) TestErrorsOutWithIncorrectSignature() {
	// We create a "malicious" token which is the same as the previous token, but signed with a different,
	// random private key, and make sure our parser rejects it.
	legitToken, err := suite.signer.SignedJWT(fakeRole)
	suite.Require().NoError(err)

	t, err := jwt.ParseSigned(legitToken)
	suite.Require().NoError(err)
	suite.Require().Len(t.Headers, 1)

	key, exists := suite.signer.Key(t.Headers[0].KeyID)
	suite.Require().True(exists)
	var claims jwt.Claims
	err = t.Claims(key, &claims)
	suite.Require().NoError(err)

	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	suite.Require().NoError(err)

	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", t.Headers[0].KeyID))
	suite.Require().NoError(err)

	maliciousToken, err := jwt.Signed(sig).Claims(claims).CompactSerialize()
	suite.Require().NoError(err)

	_, err = suite.parser.Parse(headersFrom(maliciousToken), nil)
	suite.Error(err)
	suite.True(strings.Contains(err.Error(), pkgJWT.ErrUnverifiableToken.Error()), err.Error())
	// This ensures that it failed because of the wrong secret key being used, and not because
	// of some error in another part of the chain.
	suite.True(strings.Contains(err.Error(), "error in cryptographic primitive"), err.Error())
}

func (suite *APITokenTestSuite) TestReturnsErrorIfRoleNotFound() {
	token, err := suite.signer.SignedJWT("NONEXISTENT")
	suite.Require().NoError(err)
	_, err = suite.parser.Parse(headersFrom(token), nil)
	suite.Error(err, "Expected error for non-existent role")
}

func (suite *APITokenTestSuite) TestReturnsErrorIfHeadersDontMatchFormatting() {
	token, err := suite.signer.SignedJWT(fakeRole)
	suite.Require().NoError(err)
	_, err = suite.parser.Parse(map[string][]string{
		"authorization": {fmt.Sprintf("BEARE %s", token)},
	}, nil)
	suite.Error(err, "Expected error for badly formatted header")
}

func TestAPITokenTestSuite(t *testing.T) {
	suite.Run(t, new(APITokenTestSuite))
}
