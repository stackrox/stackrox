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

	pubKey1, pubKey2, pubKey3 []byte

	signer1, signer2, signer3 cryptoutils.Signer
	validator                 Validator
}

func TestValidator(t *testing.T) {
	suite.Run(t, new(validatorTestSuite))
}

func (s *validatorTestSuite) generatePublicKeyAndSigner() ([]byte, cryptoutils.Signer) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s.Require().NoError(err)

	marshalled, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	s.Require().NoError(err)
	return marshalled, cryptoutils.NewECDSASigner(key, crypto.SHA256)
}

func (s *validatorTestSuite) SetupSuite() {
	var err error

	s.pubKey1, s.signer1 = s.generatePublicKeyAndSigner()

	s.validator = newValidator()
	err = s.validator.RegisterSigningKey(EC256, s.pubKey1, &SigningKeyRestrictions{
		MaxDuration:                             7 * 24 * time.Hour,
		EnforcementURLs:                         []string{"https://license-enforcement.stackrox.io/api/v1/validate"},
		AllowNoNodeLimit:                        true,
		AllowNoDeploymentEnvironmentRestriction: true,
		AllowNoBuildFlavorRestriction:           true,
	})
	s.Require().NoError(err)

	s.pubKey2, s.signer2 = s.generatePublicKeyAndSigner()

	s.Require().NoError(s.validator.RegisterSigningKey(EC256, s.pubKey2, &SigningKeyRestrictions{}))
	s.Require().NoError(err)

	s.pubKey3, s.signer3 = s.generatePublicKeyAndSigner()

	s.Require().NoError(s.validator.RegisterSigningKey(EC256, s.pubKey3, nil))

}

func (s *validatorTestSuite) SetupTest() {
	s.license = &licenseproto.License{
		Metadata: &licenseproto.License_Metadata{
			Id:              uuid.NewV4().String(),
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

func (s *validatorTestSuite) generateLicenseKey(pubKey []byte, signer cryptoutils.Signer) string {
	s.license.Metadata.SigningKeyId = license.SigningKeyFingerprint(pubKey)
	licenseBytes, err := proto.Marshal(s.license)
	s.Require().NoError(err)

	signatureBytes, err := signer.Sign(licenseBytes, rand.Reader)
	s.Require().NoError(err)

	return license.EncodeLicenseKey(licenseBytes, signatureBytes)
}

func (s *validatorTestSuite) TestValid() {
	licenseKey := s.generateLicenseKey(s.pubKey1, s.signer1)
	validated, err := s.validator.ValidateLicenseKey(licenseKey)
	s.NoError(err)
	s.Equal(s.license, validated)
}

func (s *validatorTestSuite) TestInvalidSigningKeyID() {
	licenseKey := s.generateLicenseKey([]byte("blah"), s.signer1)
	_, err := s.validator.ValidateLicenseKey(licenseKey)
	s.Error(err)
}

func (s *validatorTestSuite) TestInvalidSignature() {
	licenseKey := s.generateLicenseKey(s.pubKey2, s.signer1)
	_, err := s.validator.ValidateLicenseKey(licenseKey)
	s.Error(err)
}

func (s *validatorTestSuite) TestViolatesRestrictions() {
	s.license.Restrictions.NotValidAfter = protoconv.ConvertTimeToTimestamp(time.Now().Add(14 * 24 * time.Hour))
	licenseKey := s.generateLicenseKey(s.pubKey1, s.signer1)
	_, err := s.validator.ValidateLicenseKey(licenseKey)
	s.Error(err)
}

func (s *validatorTestSuite) TestNoRestrictions() {
	licenseKey := s.generateLicenseKey(s.pubKey3, s.signer3)
	_, err := s.validator.ValidateLicenseKey(licenseKey)
	s.NoError(err)
}
