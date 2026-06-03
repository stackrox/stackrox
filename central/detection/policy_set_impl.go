package detection

import (
	"context"
	"errors"

	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	log = logging.LoggerForModule()

	policyCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
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
			// Policy may have been deleted concurrently (e.g., config-controller removing a declarative policy).
			// Update of categories will fail because of foreign key constraint.
			if errors.Is(err, errox.ReferencedObjectNotFound) {
				log.Warnf("Skipping notifier removal from policy %s: %v", policy.GetId(), err)
				continue
			}

			return err
		}
	}

	return nil
}
