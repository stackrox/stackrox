package processor

import (
	"context"

	"github.com/stackrox/stackrox/central/notifiers"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	notifierSAC = sac.ForResource(resources.Notifier)
)

// NotifierSet is a set that coordinates present policies and notifiers.
type NotifierSet interface {
	HasNotifiers() bool
	HasEnabledAuditNotifiers() bool

	ForEach(ctx context.Context, f func(context.Context, notifiers.Notifier, AlertSet))

	UpsertNotifier(ctx context.Context, notifier notifiers.Notifier)
	RemoveNotifier(ctx context.Context, id string)
	GetNotifier(ctx context.Context, id string) notifiers.Notifier
}

// NewNotifierSet returns a new instance of a NotifierSet
func NewNotifierSet() NotifierSet {
	return &notifierSetImpl{
		notifiers: make(map[string]notifiers.Notifier),
		failures:  make(map[string]AlertSet),
	}
}

// Implementation of the notifier set.
//////////////////////////////////////

type notifierSetImpl struct {
	lock sync.RWMutex

	notifiers map[string]notifiers.Notifier
	failures  map[string]AlertSet
}

// HasNotifiers returns if there are any notifiers in the set.
func (p *notifierSetImpl) HasNotifiers() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.notifiers) > 0
}

// HasEnabledAuditNotifiers returns if there are any enabled notifiers in the set.
func (p *notifierSetImpl) HasEnabledAuditNotifiers() bool {
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

// ForEachesFailures performs a function on each notifier.
func (p *notifierSetImpl) ForEach(ctx context.Context, f func(context.Context, notifiers.Notifier, AlertSet)) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for notifierID, notifier := range p.notifiers {
		f(ctx, notifier, p.failures[notifierID])
	}
}

// UpsertNotifier adds or updates a notifier in the set.
func (p *notifierSetImpl) UpsertNotifier(ctx context.Context, notifier notifiers.Notifier) {
	p.lock.Lock()
	defer p.lock.Unlock()

	notifierID := notifier.ProtoNotifier().GetId()
	if _, exists := p.failures[notifierID]; !exists {
		p.failures[notifierID] = NewAlertSet()
	}
	if knownNotifier := p.notifiers[notifierID]; knownNotifier != nil && knownNotifier != notifier {
		if err := knownNotifier.Close(ctx); err != nil {
			log.Error("failed to close notifier instance", logging.Err(err))
		}
	}
	p.notifiers[notifierID] = notifier
}

// RemoveNotifier removes a notifier from the set.
func (p *notifierSetImpl) RemoveNotifier(ctx context.Context, id string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if notifier := p.notifiers[id]; notifier != nil {
		if err := notifier.Close(ctx); err != nil {
			log.Error("failed to close notifier instance", logging.Err(err))
		}
	}

	delete(p.notifiers, id)
	delete(p.failures, id)
}

// GetNotifier gets a notifier from the set.
func (p *notifierSetImpl) GetNotifier(ctx context.Context, id string) notifiers.Notifier {
	if ok, err := notifierSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	return p.notifiers[id]
}
