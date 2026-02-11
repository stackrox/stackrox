package detection

import (
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/scopecomp"
)

// PolicySet is a set of policies.
type PolicySet interface {
	detection.PolicySet

	RemoveNotifier(notifierID string) error
}

// NewPolicySet returns a new instance of a PolicySet using the provided label providers.

func NewPolicySet(store policyDatastore.DataStore, clusterProvider scopecomp.ClusterLabelProvider, namespaceProvider scopecomp.NamespaceLabelProvider) PolicySet {
	return &setImpl{
		PolicySet:   detection.NewPolicySet(clusterProvider, namespaceProvider),
		policyStore: store,
	}
}
