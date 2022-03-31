package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// EnsureConvertedToLatest converts the given policy into a Boolean policy version 1.1, if it is not one already.
func EnsureConvertedToLatest(p *storage.Policy) error {
	if p == nil {
		return errors.New("nil policy")
	}
	p.PolicyVersion = CurrentVersion().String()
	return nil
}

// MustEnsureConverted converts the passed policy if required.
// The passed policy is modified in-place, but returned for convenience.
// Any error in conversion results in a panic.
// ONLY USE in program initialization blocks, similar to regexp.MustCompile.
func MustEnsureConverted(p *storage.Policy) *storage.Policy {
	utils.Must(EnsureConvertedToLatest(p))
	return p
}
