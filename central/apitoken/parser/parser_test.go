package parser

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/central/apitoken/signer"
	"github.com/stackrox/rox/pkg/auth/permissions"
	pkgJWT "github.com/stackrox/rox/pkg/jwt"
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

func (mockRoleStore) GetRoles() []permissions.Role {
	panic("Not implemented")
}

type mockTokenRevocationChecker struct {
	ReturnError bool
}

func (m *mockTokenRevocationChecker) CheckTokenRevocation(id string) error {
	if m.ReturnError {
		return errors.New("error")
	}
	return nil
}

func headersFrom(token string) map[string][]string {
	return map[string][]string{
		"authorization": {fmt.Sprintf("Bearer %s", token)},
	}
}

type APITokenTestSuite struct {
	suite.Suite
	signer                     signer.Signer
	parser                     Parser
	mockTokenRevocationChecker mockTokenRevocationChecker
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
	suite.mockTokenRevocationChecker = mockTokenRevocationChecker{}
	suite.parser = New(suite.signer, mockRoleStore{}, &suite.mockTokenRevocationChecker)
}

// Happy path
func (suite *APITokenTestSuite) TestWorksWithExistingRole() {
	before := time.Now()
	token, id, issuedAt, expiration, err := suite.signer.SignedJWT(fakeRole)
	suite.Require().NoError(err)
	after := time.Now()

	suite.True(before.Before(issuedAt))
	suite.True(after.After(issuedAt))
	identity, err := suite.parser.ParseToken(token)
	suite.Require().NoError(err)
	suite.Equal(id, identity.ID())
	suite.Equal(fakeRole, identity.Role().Name())
	suite.True(identity.Expiration().After(time.Now()))
	suite.True(identity.Expiration().Before(time.Now().Add(366 * 24 * time.Hour)))
	suite.Equal(expiration.Unix(), identity.Expiration().Unix())
}

func (suite *APITokenTestSuite) TestErrorsOutWithIncorrectSignature() {
	// We create a "malicious" token which is the same as the previous token, but signed with a different,
	// random private key, and make sure our parser rejects it.
	legitToken, _, _, _, err := suite.signer.SignedJWT(fakeRole)
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
	token, _, _, _, err := suite.signer.SignedJWT("NONEXISTENT")
	suite.Require().NoError(err)
	_, err = suite.parser.Parse(headersFrom(token), nil)
	suite.Error(err, "Expected error for non-existent role")
}

func (suite *APITokenTestSuite) TestReturnsErrorIfHeadersDontMatchFormatting() {
	token, _, _, _, err := suite.signer.SignedJWT(fakeRole)
	suite.Require().NoError(err)
	_, err = suite.parser.Parse(map[string][]string{
		"authorization": {fmt.Sprintf("BEARE %s", token)},
	}, nil)
	suite.Error(err, "Expected error for badly formatted header")
}

func (suite *APITokenTestSuite) TestRevocationChecks() {
	token, _, _, _, err := suite.signer.SignedJWT(fakeRole)
	suite.Require().NoError(err)

	_, err = suite.parser.Parse(headersFrom(token), nil)
	suite.NoError(err)

	_, err = suite.parser.ParseToken(token)
	suite.NoError(err)

	suite.mockTokenRevocationChecker.ReturnError = true

	_, err = suite.parser.Parse(headersFrom(token), nil)
	suite.Error(err)

	_, err = suite.parser.ParseToken(token)
	suite.Error(err)
}

func TestAPITokenTestSuite(t *testing.T) {
	suite.Run(t, new(APITokenTestSuite))
}
