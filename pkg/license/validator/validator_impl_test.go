package validator

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"testing"
	"time"

	// Ensure SHA256 hash function is available.
	_ "crypto/sha256"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/license"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type validatorTestSuite struct {
	suite.Suite

	license *licenseproto.License

	pubKey1, pubKey2 []byte

	signer    cryptoutils.Signer
	validator Validator
}

func TestValidator(t *testing.T) {
	suite.Run(t, new(validatorTestSuite))
}

func (s *validatorTestSuite) SetupSuite() {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)

	s.pubKey1, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
	s.Require().NoError(err)

	s.signer = cryptoutils.NewECDSASigner(key, crypto.SHA256)
	s.validator = newValidator()
	err = s.validator.RegisterSigningKey(EC256, s.pubKey1, SigningKeyRestrictions{
		MaxDuration:                             7 * 24 * time.Hour,
		EnforcementURLs:                         []string{"https://license-enforcement.stackrox.io/api/v1/validate"},
		AllowNoNodeLimit:                        true,
		AllowNoDeploymentEnvironmentRestriction: true,
		AllowNoBuildFlavorRestriction:           true,
	})
	s.Require().NoError(err)

	otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)

	s.pubKey2, err = x509.MarshalPKIXPublicKey(&otherKey.PublicKey)
	s.Require().NoError(err)

	err = s.validator.RegisterSigningKey(EC256, s.pubKey2, SigningKeyRestrictions{})
	s.Require().NoError(err)
}

func (s *validatorTestSuite) SetupTest() {
	s.license = &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id:              uuid.NewV4().String(),
			SigningKeyId:    license.SigningKeyFingerprint(s.pubKey1),
			IssueDate:       types.TimestampNow(),
			LicensedForId:   "test",
			LicensedForName: "Test",
		},
		Restrictions: &licenseproto.License_Restrictions{
			NotValidBefore:                     types.TimestampNow(),
			NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(24 * time.Hour)),
			EnforcementUrl:                     "https://license-enforcement.stackrox.io/api/v1/validate",
			NoNodeRestriction:                  true,
			NoBuildFlavorRestriction:           true,
			NoDeploymentEnvironmentRestriction: true,
		},
	}
}

func (s *validatorTestSuite) generateLicenseKey() string {
	licenseBytes, err := proto.Marshal(s.license)
	s.Require().NoError(err)

	signatureBytes, err := s.signer.Sign(licenseBytes, rand.Reader)
	s.Require().NoError(err)

	return license.EncodeLicenseKey(licenseBytes, signatureBytes)
}

func (s *validatorTestSuite) TestValid() {
	licenseKey := s.generateLicenseKey()
	validated, err := s.validator.ValidateLicenseKey(licenseKey)
	s.NoError(err)
	s.Equal(s.license, validated)
}

func (s *validatorTestSuite) TestInvalidSigningKeyID() {
	s.license.Metadata.SigningKeyId = "test/key/2"
	licenseKey := s.generateLicenseKey()
	_, err := s.validator.ValidateLicenseKey(licenseKey)
	s.Error(err)
}

func (s *validatorTestSuite) TestInvalidSignature() {
	s.license.Metadata.SigningKeyId = license.SigningKeyFingerprint(s.pubKey2)
	licenseKey := s.generateLicenseKey()
	_, err := s.validator.ValidateLicenseKey(licenseKey)
	s.Error(err)
}

func (s *validatorTestSuite) TestViolatesRestrictions() {
	s.license.Restrictions.NotValidAfter = protoconv.ConvertTimeToTimestamp(time.Now().Add(14 * 24 * time.Hour))
	licenseKey := s.generateLicenseKey()
	_, err := s.validator.ValidateLicenseKey(licenseKey)
	s.Error(err)
}
