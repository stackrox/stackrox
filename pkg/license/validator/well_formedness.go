package validator

import (
	"errors"
	"strings"

	"github.com/gogo/protobuf/types"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/uuid"
)

func validateUUID(input string) error {
	if id, err := uuid.FromString(input); err != nil {
		return err
	} else if id == uuid.Nil {
		return errors.New("UUID is nil")
	}
	return nil
}

func checkMetadataIsWellFormed(md *v1.License_Metadata, errs *errorhelpers.ErrorList) {
	if err := validateUUID(md.GetId()); err != nil {
		errs.AddStringf("invalid license ID: %v", err)
	}
	if signingKeyID := md.GetSigningKeyId(); signingKeyID == "" {
		errs.AddStringf("invalid signing key ID: %v", signingKeyID)
	}
	if _, err := types.TimestampFromProto(md.GetIssueDate()); err != nil {
		errs.AddStringf("invalid issue timestamp: %v", err)
	}

	if md.GetLicensedForId() == "" {
		errs.AddString("missing `licensed for` ID")
	}
	if md.GetLicensedForName() == "" {
		errs.AddString("missing `licensed for` name")
	}
}

func checkRestrictionsAreWellFormed(restr *v1.License_Restrictions, errs *errorhelpers.ErrorList) {
	nvb, err := types.TimestampFromProto(restr.GetNotValidBefore())
	if err != nil {
		errs.AddStringf("invalid NotValidBefore: %v", err)
	}
	nva, err := types.TimestampFromProto(restr.GetNotValidAfter())
	if err != nil {
		errs.AddStringf("invalid NotValidAfter: %v", err)
	}

	if !nva.IsZero() && !nvb.IsZero() && !nvb.Before(nva) {
		errs.AddStringf("NotValidBefore (%v) is not before NotValidAfter (%v)", nvb, nva)
	}

	if restr.GetMaxNodes() < 0 {
		errs.AddStringf("MaxNodes is negative (%d)", restr.GetMaxNodes())
	} else if restr.GetNoNodeRestriction() && restr.GetMaxNodes() != 0 {
		errs.AddStringf("license has no node count restriction, but MaxNodes is nonzero")
	} else if !restr.GetNoNodeRestriction() && restr.GetMaxNodes() == 0 {
		errs.AddStringf("license does not allow unrestricted node count, but MaxNodes is zero")
	}

	if restr.GetAllowOffline() && restr.GetEnforcementUrl() != "" {
		errs.AddStringf("license allows offline use, but has a non-empty enforcement URL of %q", restr.GetEnforcementUrl())
	} else if !restr.GetAllowOffline() {
		if restr.GetEnforcementUrl() == "" {
			errs.AddString("license does not allow offline use, but does not specify an enforcement URL")
		} else if !strings.HasPrefix(restr.GetEnforcementUrl(), "https://") {
			errs.AddString("license enforcement URL is not a HTTPS URL")
		}
	}

	if restr.GetNoBuildFlavorRestriction() && len(restr.GetBuildFlavors()) != 0 {
		errs.AddStringf("license has no build flavors restriction, but specifies a set of allowed build flavors of %v", restr.GetBuildFlavors())
	} else if !restr.GetNoBuildFlavorRestriction() && len(restr.GetBuildFlavors()) == 0 {
		errs.AddString("license does not allow use with any build flavor, but set of allowed build flavors is empty")
	}

	if restr.GetNoDeploymentEnvironmentRestriction() && len(restr.GetDeploymentEnvironments()) != 0 {
		errs.AddStringf("license has no deployment environment restriction, but specifies a set of allowed deployment environments of %v", restr.GetDeploymentEnvironments())
	} else if !restr.GetNoDeploymentEnvironmentRestriction() && len(restr.GetDeploymentEnvironments()) == 0 {
		errs.AddString("license does not allow use in any deployment environment, but set of allowed deployment environments is empty")
	}
}

// CheckLicenseIsWellFormed ensures that the given license is well-formed (i.e., all required fields are populated
// with values that make sense).
func CheckLicenseIsWellFormed(license *v1.License) error {
	errs := errorhelpers.NewErrorList("validating license well-formedness")

	if md := license.GetMetadata(); md == nil {
		errs.AddString("license does not have metadata")
	} else {
		checkMetadataIsWellFormed(md, errs)
	}

	if restr := license.GetRestrictions(); restr == nil {
		errs.AddString("license does not have restrictions")
	} else {
		checkRestrictionsAreWellFormed(restr, errs)
	}

	return errs.ToError()
}
