package detection

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
)

var (
	log = logging.LoggerForModule()
)

// PolicySet is a set of policies.
//
//go:generate mockgen-wrapper
type PolicySet interface {
	ForOne(policyID string, f func(CompiledPolicy) error) error
	ForEach(func(CompiledPolicy) error) error
	GetCompiledPolicies() map[string]CompiledPolicy

	Exists(id string) bool
	UpsertPolicy(*storage.Policy) error
	RemovePolicy(policyID string)
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet() PolicySet {
	return &setImpl{
		policyIDToCompiled: maputil.NewFastRMap[string, CompiledPolicy](),
	}
}
