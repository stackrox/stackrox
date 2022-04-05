package policyversion

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// EnsureConvertedToLatest converts the given policy into a Boolean policy version 1.1, if it is not one already.
func EnsureConvertedToLatest(p *storage.Policy) error {
	if p == nil {
		return errors.New("nil policy")
	}
	if p.PolicyVersion != CurrentVersion().String() {
		return fmt.Errorf("unsupported policy version %s", p.PolicyVersion)
	}
	return nil
}
