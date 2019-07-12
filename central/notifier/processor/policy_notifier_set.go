package processor

import (
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// Definition of the notifierSet interface.
///////////////////////////////////////////

type policyNotifierSet interface {
	hasNotifiers() bool
	hasEnabledAuditNotifiers() bool

	forEach(f func(notifiers.Notifier))
	forEachIntegratedWith(policyID string, f func(notifiers.Notifier))

	upsertNotifier(notifier notifiers.Notifier)
	removeNotifier(id string)

	upsertPolicy(policy *storage.Policy)
	removePolicy(policy *storage.Policy)
}

// Construct a new notifierSet.
///////////////////////////////

func newPolicyNotifierSet() policyNotifierSet {
	return &policyNotifierSetImpl{
		notifiers:           make(map[string]notifiers.Notifier),
		notifiersToPolicies: make(map[string]set.StringSet),
	}
}

// Implementation of the notifier set.
//////////////////////////////////////

type policyNotifierSetImpl struct {
	lock sync.RWMutex

	notifiers           map[string]notifiers.Notifier
	notifiersToPolicies map[string]set.StringSet
}

func (p *policyNotifierSetImpl) hasNotifiers() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.notifiers) > 0
}

func (p *policyNotifierSetImpl) hasEnabledAuditNotifiers() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, n := range p.notifiers {
		auditN, ok := n.(notifiers.AuditNotifier)
		if ok && auditN.AuditLoggingEnabled() {
			return true
		}
	}
	return false
}

func (p *policyNotifierSetImpl) forEach(f func(notifiers.Notifier)) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, notifier := range p.notifiers {
		f(notifier)
	}
}

func (p *policyNotifierSetImpl) forEachIntegratedWith(policyID string, f func(notifiers.Notifier)) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for notifierID, notifier := range p.notifiers {
		if !p.notifiersToPolicies[notifierID].Contains(policyID) {
			continue
		}
		f(notifier)
	}
}

func (p *policyNotifierSetImpl) upsertNotifier(notifier notifiers.Notifier) {
	p.lock.Lock()
	defer p.lock.Unlock()

	notifierID := notifier.ProtoNotifier().GetId()
	p.notifiers[notifierID] = notifier
	if _, hasPolicySet := p.notifiersToPolicies[notifierID]; !hasPolicySet {
		p.notifiersToPolicies[notifierID] = set.NewStringSet()
	}
}

func (p *policyNotifierSetImpl) removeNotifier(id string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.notifiers, id)
	delete(p.notifiersToPolicies, id)
}

func (p *policyNotifierSetImpl) upsertPolicy(policy *storage.Policy) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Remove old refs to the policy id.
	for _, policyIDS := range p.notifiersToPolicies {
		policyIDS.Remove(policy.GetId())
	}

	// Add new refs.
	for _, notifierID := range policy.GetNotifiers() {
		if _, hasPolicySet := p.notifiersToPolicies[notifierID]; !hasPolicySet {
			p.notifiersToPolicies[notifierID] = set.NewStringSet()
		}
		p.notifiersToPolicies[notifierID].Add(policy.GetId())
	}
}

func (p *policyNotifierSetImpl) removePolicy(policy *storage.Policy) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, policyIDS := range p.notifiersToPolicies {
		policyIDS.Remove(policy.GetId())
	}
}
