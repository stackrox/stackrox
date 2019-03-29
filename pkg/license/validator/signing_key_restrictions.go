package validator

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// SigningKeyRestrictions determines restrictions as to what a signing key can be used for. Please refer to the license
// proto definition for an explanation of the individual fields.
type SigningKeyRestrictions struct {
	EarliestNotValidBefore time.Time
	LatestNotValidAfter    time.Time

	MaxDuration time.Duration

	EnforcementURLs []string
	AllowOffline    bool

	MaxNodeLimit     int
	AllowNoNodeLimit bool

	BuildFlavors                  []string
	AllowNoBuildFlavorRestriction bool

	DeploymentEnvironments                  []string
	AllowNoDeploymentEnvironmentRestriction bool
}

// Check checks that the given license restrictions are within the bounds set by the signing key restrictions.
func (r *SigningKeyRestrictions) Check(licenseRestrictions *licenseproto.License_Restrictions) error {
	errs := errorhelpers.NewErrorList("checking restrictions of signing key")

	notValidBefore, err := types.TimestampFromProto(licenseRestrictions.GetNotValidBefore())
	if err != nil {
		return errors.Errorf("converting NotValidBefore: %v", err)
	}

	notValidAfter, err := types.TimestampFromProto(licenseRestrictions.GetNotValidAfter())
	if err != nil {
		return errors.Errorf("converting NotValidAfter: %v", err)
	}

	if !r.EarliestNotValidBefore.IsZero() {
		if notValidBefore.Before(r.EarliestNotValidBefore) {
			errs.AddStringf("license NotValidBefore of %v is earlier than earliest allowed NotValidBefore %v of signing key", notValidBefore, r.EarliestNotValidBefore)
		}
	}
	if !r.LatestNotValidAfter.IsZero() {
		if notValidAfter.After(r.LatestNotValidAfter) {
			errs.AddStringf("license NotValidAfter of %v is later than latest allowed NotValidAfter %v of signing key", notValidAfter, r.LatestNotValidAfter)
		}
	}

	if r.MaxDuration != 0 {
		if licenseDuration := notValidAfter.Sub(notValidBefore); licenseDuration > r.MaxDuration {
			errs.AddStringf("validity duration of license %v is longer than maximum allowed duration %v of signing key", licenseDuration, r.MaxDuration)
		}
	}

	if !r.AllowNoNodeLimit {
		if licenseRestrictions.GetNoNodeRestriction() {
			errs.AddString("license has no node count restriction, but signing key does not allow unlimited node counts")
		} else if licenseRestrictions.GetMaxNodes() > int32(r.MaxNodeLimit) {
			errs.AddStringf("node limit %d of license is higher than maximum allowed node limit %d of signing key", licenseRestrictions.GetMaxNodes(), r.MaxNodeLimit)
		}
	}

	if !r.AllowOffline {
		if licenseRestrictions.GetAllowOffline() {
			errs.AddString("license allows offline use, but signing key is not valid for offline use licenses")
		} else {
			if sliceutils.StringFind(r.EnforcementURLs, licenseRestrictions.GetEnforcementUrl()) == -1 {
				errs.AddStringf("enforcement URL %q of license is not in the set %v of enforcement URLs allowed by the signing key", licenseRestrictions.GetEnforcementUrl(), r.EnforcementURLs)
			}
		}
	}

	if !r.AllowNoBuildFlavorRestriction {
		if licenseRestrictions.GetNoBuildFlavorRestriction() {
			errs.AddString("license allows use for all build flavors, but signing key is not valid for all build flavors")
		} else {
			// len(licenseRestrictions.GetBuildFlavors()) > 0 handled by well-formedness check
			for _, flavor := range licenseRestrictions.GetBuildFlavors() {
				if sliceutils.StringFind(r.BuildFlavors, flavor) == -1 {
					errs.AddStringf("build flavor %q allowed by license not in the set %v of build flavors allowed by the signing key", flavor, r.BuildFlavors)
				}
			}
		}
	}

	if !r.AllowNoDeploymentEnvironmentRestriction {
		if licenseRestrictions.GetNoDeploymentEnvironmentRestriction() {
			errs.AddString("license allows use in any deployment environment, but signing key is not valid for all deployment environments")
		} else {
			// len(licenseRestrictions.GetNoDeploymentEnvironmentRestriction()) > 0 handled by well-formedness check
			for _, deploymentEnv := range licenseRestrictions.GetDeploymentEnvironments() {
				if sliceutils.StringFind(r.DeploymentEnvironments, deploymentEnv) == -1 {
					errs.AddStringf("deployment environment %q allowed by license not in the set %v of deployment environments allowed by the signing key", deploymentEnv, r.DeploymentEnvironments)
				}
			}
		}
	}

	return errs.ToError()
}
