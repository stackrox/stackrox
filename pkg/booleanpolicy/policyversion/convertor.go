package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// EnsureConvertedToLatest converts the given policy into a Boolean policy, if it is not one already.
func EnsureConvertedToLatest(p *storage.Policy) error {
	if p == nil {
		return errors.New("nil policy")
	}
	policyVersion, err := FromString(p.GetPolicyVersion())
	if err != nil {
		return err
	}

	if Compare(policyVersion, Version1()) >= 0 && len(p.GetPolicySections()) == 0 {
		return errors.New("empty sections")
	}
	if Compare(policyVersion, Version1()) < 0 {
		// If a policy is sent with legacyVersion but contains sections, that's okay --
		// we will use those sections as-is, and infer that it's of the newer version.
		if p.GetFields() == nil && len(p.GetPolicySections()) == 0 {
			return errors.New("empty policy")
		}

		upgradeLegacyToVersion1(p)
	}
	if Compare(policyVersion, Version1()) > 0 && len(p.GetWhitelists()) > 0 {
		// Policy.whitelists is deprecated in favor of Policy.exclusions in all
		// versions greater than Version1.
		return errors.New("field 'whitelists' is deprecated in this version")
	}
	if Compare(policyVersion, Version1()) <= 0 {
		// It's fine to receive exclusions but not both exclusions and whitelists.
		if len(p.GetWhitelists()) > 0 && len(p.GetExclusions()) > 0 {
			return errors.New("both 'exclusions' and 'whitelists' fields are set")
		}

		upgradeVersion1ToVersion1_1(p)
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
