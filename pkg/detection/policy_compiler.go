package detection

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
)

// CompilePolicy compiles the given policy with label providers, making it ready for matching.
// The providers enable cluster_label and namespace_label scope matching.
// Pass nil for providers if label-based scoping is not needed.
func CompilePolicy(policy *storage.Policy, clusterLabelProvider scopecomp.ClusterLabelProvider, namespaceLabelProvider scopecomp.NamespaceLabelProvider) (CompiledPolicy, error) {
	cloned := policy.CloneVT()
	return newCompiledPolicy(cloned, clusterLabelProvider, namespaceLabelProvider)
}
