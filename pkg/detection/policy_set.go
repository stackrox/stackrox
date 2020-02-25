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
	Compiler() PolicyCompiler

	ForOne(policyID string, pt PolicyExecutor) error
	ForEach(pt PolicyExecutor) error
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
