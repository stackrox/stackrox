package detection

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// PolicySet is a set of policies.
//go:generate mockgen-wrapper
type PolicySet interface {
	ForOne(policyID string, f func(CompiledPolicy) error) error
	ForEach(func(CompiledPolicy) error) error
	GetCompiledPolicies() map[string]CompiledPolicy

	UpsertPolicy(*storage.Policy) error
	RemovePolicy(policyID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(compiler PolicyCompiler) PolicySet {
	return &setImpl{
		policyIDToCompiled: NewStringCompiledPolicyFastRMap(),
		compiler:           compiler,
	}
}
