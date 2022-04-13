package detection

import (
	"context"

	policyDatastore "github.com/stackrox/stackrox/central/policy/datastore"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/detection"
	"github.com/stackrox/stackrox/pkg/sac"
)

var (
	policyCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
)

type setImpl struct {
	policyStore policyDatastore.DataStore
	detection.PolicySet
}

// RemoveNotifier removes a given notifier from any policies in the set that use it.
func (p *setImpl) RemoveNotifier(notifierID string) error {
	m := p.PolicySet.GetCompiledPolicies()

	for _, compiled := range m {
		policy := compiled.Policy()

		notifiers := policy.GetNotifiers()
		outIdx := 0
		for i, n := range policy.GetNotifiers() {
			if n != notifierID {
				if i != outIdx {
					notifiers[outIdx] = n
				}
				outIdx++
			}
		}
		if outIdx >= len(notifiers) { // no change
			continue
		}
		policy.Notifiers = notifiers[:outIdx]

		err := p.policyStore.UpdatePolicy(policyCtx, policy)
		if err != nil {
			return err
		}
	}

	return nil
}
