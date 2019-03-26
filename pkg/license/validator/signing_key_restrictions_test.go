package validator

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/suite"
)

type signingKeyRestrictionsTestSuite struct {
	suite.Suite

	keyRestrictions     SigningKeyRestrictions
	licenseRestrictions *v1.License_Restrictions
}

func TestSigningKeyRestrictions(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(signingKeyRestrictionsTestSuite))
}

func (s *signingKeyRestrictionsTestSuite) SetupTest() {
	s.keyRestrictions = SigningKeyRestrictions{
		EarliestNotValidBefore: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
		LatestNotValidAfter:    time.Date(2018, 12, 31, 23, 59, 59, 0, time.UTC),
		MaxDuration:            7 * 24 * time.Hour,
		AllowOffline:           false,
		EnforcementURLs:        []string{"https://license-enforcement.stackrox.io/api/v1/validate"},
		MaxNodeLimit:           100,
		BuildFlavors:           []string{"development", "rc"},
		DeploymentEnvironments: []string{"gcp/ultra-current-825", "gcp/stackrox-ci"},
	}
	s.licenseRestrictions = &v1.License_Restrictions{
		NotValidBefore:         protoconv.ConvertTimeToTimestamp(time.Date(2018, 5, 3, 0, 0, 0, 0, time.UTC)),
		NotValidAfter:          protoconv.ConvertTimeToTimestamp(time.Date(2018, 5, 8, 23, 59, 59, 0, time.UTC)),
		AllowOffline:           false,
		EnforcementUrl:         "https://license-enforcement.stackrox.io/api/v1/validate",
		MaxNodes:               90,
		BuildFlavors:           []string{"development"},
		DeploymentEnvironments: []string{"gcp/ultra-current-825"},
	}
}

func (s *signingKeyRestrictionsTestSuite) TestAllValid() {
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}

func (s *signingKeyRestrictionsTestSuite) TestViolateNotValidBefore() {
	s.licenseRestrictions.NotValidBefore.Seconds -= 365 * 86400
	s.licenseRestrictions.NotValidAfter.Seconds -= 365 * 86400

	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.keyRestrictions.EarliestNotValidBefore = time.Time{}
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}

func (s *signingKeyRestrictionsTestSuite) TestViolateNotValidAfter() {
	s.licenseRestrictions.NotValidBefore.Seconds += 365 * 86400
	s.licenseRestrictions.NotValidAfter.Seconds += 365 * 86400

	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.keyRestrictions.LatestNotValidAfter = time.Time{}
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}

func (s *signingKeyRestrictionsTestSuite) TestViolateMaxDuration() {
	s.licenseRestrictions.NotValidAfter.Seconds += 3 * 86400

	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.keyRestrictions.MaxDuration = 0
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}

func (s *signingKeyRestrictionsTestSuite) TestViolateAllowOffline() {
	s.licenseRestrictions.AllowOffline = true

	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.keyRestrictions.AllowOffline = true
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}

func (s *signingKeyRestrictionsTestSuite) TestViolateMaxNodeLimit() {
	s.licenseRestrictions.MaxNodes = 110
	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.licenseRestrictions.MaxNodes = 0
	s.licenseRestrictions.NoNodeRestriction = true
	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.keyRestrictions.MaxNodeLimit = 0
	s.keyRestrictions.AllowNoNodeLimit = true
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))

	s.licenseRestrictions.MaxNodes = 110
	s.licenseRestrictions.NoNodeRestriction = false
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}

func (s *signingKeyRestrictionsTestSuite) TestViolateBuildFlavors() {
	s.licenseRestrictions.BuildFlavors = nil
	s.licenseRestrictions.NoBuildFlavorRestriction = true
	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.licenseRestrictions.BuildFlavors = []string{"release"}
	s.licenseRestrictions.NoBuildFlavorRestriction = false
	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.keyRestrictions.BuildFlavors = nil
	s.keyRestrictions.AllowNoBuildFlavorRestriction = true
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))

	s.licenseRestrictions.BuildFlavors = nil
	s.licenseRestrictions.NoBuildFlavorRestriction = true
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}

func (s *signingKeyRestrictionsTestSuite) TestViolateDeploymentEnvironments() {
	s.licenseRestrictions.DeploymentEnvironments = nil
	s.licenseRestrictions.NoDeploymentEnvironmentRestriction = true
	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.licenseRestrictions.DeploymentEnvironments = []string{"gcp/stackrox-demo"}
	s.licenseRestrictions.NoDeploymentEnvironmentRestriction = false
	s.Error(s.keyRestrictions.Check(s.licenseRestrictions))

	s.keyRestrictions.DeploymentEnvironments = nil
	s.keyRestrictions.AllowNoDeploymentEnvironmentRestriction = true
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))

	s.licenseRestrictions.DeploymentEnvironments = nil
	s.licenseRestrictions.NoDeploymentEnvironmentRestriction = true
	s.NoError(s.keyRestrictions.Check(s.licenseRestrictions))
}
