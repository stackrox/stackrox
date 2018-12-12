package predicate

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scopecomp"
)

func init() {
	compilers = append(compilers, newScopePredicate)
}

// Return true if the deployment is within any scope present.
func newScopePredicate(policy *storage.Policy) (Predicate, error) {
	var predicate Predicate
	for _, scope := range policy.GetScope() {
		wrap := &scopeWrapper{scope: scope}
		predicate = predicate.Or(wrap.shouldProcess)
	}
	return predicate, nil
}

type scopeWrapper struct {
	scope *storage.Scope
}

func (p *scopeWrapper) shouldProcess(deployment *storage.Deployment) bool {
	if scopecomp.WithinScope(p.scope, deployment) {
		return true
	}
	return false
}
