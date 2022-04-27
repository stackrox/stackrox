package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// EnsureConvertedToLatest converts the given policy to the latest version (as defined by CurrentVersion), if it isn't already
// The policy is modified in place.
func EnsureConvertedToLatest(p *storage.Policy) error {
	if p == nil {
		return errors.New("nil policy")
	}

	ver, err := FromString(p.GetPolicyVersion())
	if err != nil {
		return errors.New("invalid version")
	}

	// If a policy is sent with legacyVersion but contains sections, that's okay --
	// we will use those sections as-is, and infer that it's of the newer version.
	// Other later validation will check to see if the rest of the policy is formatted correctly.
	// NOTE: This will be removed soon, and we will prevent anyone from making an API call without version set
	// This is an intermediate step.
	if ver.String() == legacyVersion {
		p.PolicyVersion = version1_1
	}

	// If it's not the latest version, delegate to the upgrader
	// CurrentVersion should always be the latest, thus this will always involve an upgrade.
	if !IsCurrentVersion(ver) {
		if err := upgradePolicyTo(p, CurrentVersion()); err != nil {
			return err
		}
	}

	if p.PolicyVersion != CurrentVersion().String() {
		return errors.Errorf("converted to version %q, while latest is %q", p.PolicyVersion, CurrentVersion().String())
	}
	return nil
}

// CloneAndEnsureConverted returns a clone of the input that is upgraded if it is a legacy policy
func CloneAndEnsureConverted(p *storage.Policy) (*storage.Policy, error) {
	cloned := p.Clone()
	if err := EnsureConvertedToLatest(cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}

// MustEnsureConverted converts the passed policy if required.
// The passed policy is modified in-place, but returned for convenience.
// Any error in conversion results in a panic.
// ONLY USE in program initialization blocks, similar to regexp.MustCompile.
func MustEnsureConverted(p *storage.Policy) *storage.Policy {
	utils.Must(EnsureConvertedToLatest(p))
	return p
}
