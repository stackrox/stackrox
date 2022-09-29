package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// EnsureConvertedToLatest converts the given policy to the latest version (as defined by CurrentVersion), if it isn't already
// The policy is modified in place.
func EnsureConvertedToLatest(p *storage.Policy) error {
	return EnsureConvertedTo(p, CurrentVersion())
}

// EnsureConvertedTo converts the given policy to requested version
// The policy is modified in place.
func EnsureConvertedTo(p *storage.Policy, toVersion PolicyVersion) error {
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

	switch diff := Compare(ver, toVersion); {
	case diff > 0:
		// No downgrade
		utils.CrashOnError(errors.Errorf("Unexpected version %s, cannot downgrade policy version to %s", ver.String(), toVersion.String()))
	case diff < 0:
		// If it's below the requested version, delegate to the upgrader
		if err := upgradePolicyTo(p, toVersion); err != nil {
			return err
		}
	default:
	}

	if p.PolicyVersion != toVersion.String() {
		return errors.Errorf("converted from version %q to version %q", p.PolicyVersion, toVersion.String())
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
