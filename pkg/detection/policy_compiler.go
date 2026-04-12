package detection

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
)

// CompilePolicy returns a lazily-compiled policy. The expensive compilation
// (regexp building, matcher construction) is deferred until the policy is
// first evaluated via Match* or AppliesTo. This saves ~6 MB on idle sensors
// where most of ~100 default policies are never checked against any resource.
func CompilePolicy(policy *storage.Policy, clusterLabelProvider scopecomp.ClusterLabelProvider, namespaceLabelProvider scopecomp.NamespaceLabelProvider) (CompiledPolicy, error) {
	cloned := policy.CloneVT()
	return newLazyCompiledPolicy(cloned, clusterLabelProvider, namespaceLabelProvider), nil
}
