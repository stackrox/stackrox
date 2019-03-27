package validator

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	invalidGoTimeSecondsNeg = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).Unix() - 1000
	invalidGoTimeSecondsPos = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC).Unix() + 1000
)

type wellFormednessTestSuite struct {
	suite.Suite

	license *v1.License
}

func TestCheckLicenseIsWellFormed(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(wellFormednessTestSuite))
}

func (s *wellFormednessTestSuite) SetupTest() {
	s.license = &v1.License{
		Metadata: &v1.License_Metadata{
			Id:              uuid.NewV4().String(),
			SigningKeyId:    "project/key/version",
			IssueDate:       types.TimestampNow(),
			LicensedForId:   "test",
			LicensedForName: "Test",
		},
		Restrictions: &v1.License_Restrictions{
			NotValidBefore:         types.TimestampNow(),
			NotValidAfter:          protoconv.ConvertTimeToTimestamp(time.Now().Add(24 * time.Hour)),
			EnforcementUrl:         "https://license-enforcement.stackrox.io/api/v1/validate",
			MaxNodes:               100,
			BuildFlavors:           []string{"development"},
			DeploymentEnvironments: []string{"gcp/stackrox-dev"},
		},
	}
}

func (s *wellFormednessTestSuite) TestAllValid() {
	s.NoError(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestNoMetadata() {
	s.license.Metadata = nil
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidID() {
	s.license.Metadata.Id = "This is not a UUID"
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Metadata.Id = ""
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidSigningKeyID() {
	s.license.Metadata.SigningKeyId = ""
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidIssueTimestamp() {
	s.license.Metadata.IssueDate.Seconds = invalidGoTimeSecondsNeg
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Metadata.IssueDate = nil
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidLicensedForID() {
	s.license.Metadata.LicensedForId = ""
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidLicensedForName() {
	s.license.Metadata.LicensedForName = ""
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestNoRestrictions() {
	s.license.Restrictions = nil
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidNotValidBefore() {
	s.license.Restrictions.NotValidBefore.Seconds = invalidGoTimeSecondsNeg
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Restrictions.NotValidBefore = nil
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidNotValidAfter() {
	s.license.Restrictions.NotValidAfter.Seconds = invalidGoTimeSecondsPos
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Restrictions.NotValidAfter = nil
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidValidityRange() {
	s.license.Restrictions.NotValidBefore, s.license.Restrictions.NotValidAfter = s.license.Restrictions.NotValidAfter, s.license.Restrictions.NotValidBefore
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Restrictions.NotValidBefore, s.license.Restrictions.NotValidAfter = nil, nil
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidMaxNodes() {
	s.license.Restrictions.MaxNodes = -10
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestUnrestrictedNodes() {
	s.license.Restrictions.MaxNodes = 0
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Restrictions.NoNodeRestriction = true
	s.NoError(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestNoEnforcementURL() {
	s.license.Restrictions.EnforcementUrl = ""
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Restrictions.AllowOffline = true
	s.NoError(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestInvalidEnforcementURL() {
	s.license.Restrictions.EnforcementUrl = "http://www.stackrox.com/"
	s.Error(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestNoDeploymentEnvRestriction() {
	s.license.Restrictions.DeploymentEnvironments = nil
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Restrictions.NoDeploymentEnvironmentRestriction = true
	s.NoError(CheckLicenseIsWellFormed(s.license))
}

func (s *wellFormednessTestSuite) TestNoBuildFlavorRestriction() {
	s.license.Restrictions.BuildFlavors = nil
	s.Error(CheckLicenseIsWellFormed(s.license))

	s.license.Restrictions.NoBuildFlavorRestriction = true
	s.NoError(CheckLicenseIsWellFormed(s.license))
}
