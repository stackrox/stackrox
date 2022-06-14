package detection

import (
	policyDatastore "github.com/stackrox/stackrox/central/policy/datastore"
	"github.com/stackrox/stackrox/pkg/detection"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// PolicySet is a set of policies.
type PolicySet interface {
	detection.PolicySet

	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet.
func NewPolicySet(store policyDatastore.DataStore) PolicySet {
	return &setImpl{
		PolicySet:   detection.NewPolicySet(),
		policyStore: store,
	}
}
