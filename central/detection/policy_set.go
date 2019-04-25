package detection

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// PolicySet is a set of policies.
type PolicySet interface {
	Compiler() PolicyCompiler

	ForOne(policyID string, pt PolicyExecutor) error
	ForEach(pt PolicyExecutor) error

	UpsertPolicy(*storage.Policy) error
	RemovePolicy(policyID string) error
	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore, compiler PolicyCompiler) PolicySet {
	return &setImpl{
		policyIDToCompiled: make(map[string]CompiledPolicy),
		policyStore:        store,
		compiler:           compiler,
	}
}
