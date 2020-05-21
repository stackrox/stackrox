package booleanpolicy

import (
	"github.com/stackrox/rox/generated/storage"
)

const (
	// Version is the current version of boolean policies that is handled by this package.
	Version       = "1"
	legacyVersion = ""
)

// IsBooleanPolicy returns true if the policy has policy version equal to the current version of boolean policies
func IsBooleanPolicy(p *storage.Policy) bool {
	return p.GetPolicyVersion() == Version
}
